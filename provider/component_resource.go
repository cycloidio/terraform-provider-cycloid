package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cycloidio/cycloid-cli/cmd/apiclient"
	"github.com/cycloidio/cycloid-cli/gen/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/dynamic"
	"github.com/cycloidio/terraform-provider-cycloid/resource_component"
	"github.com/cycloidio/cycloid-cli/utils/ptr"
)

var (
	_ resource.Resource                = &ComponentResource{}
	_ resource.ResourceWithImportState = &ComponentResource{}
)

type componentResourceModel resource_component.ComponentModel

func NewComponentResource() resource.Resource {
	return &ComponentResource{}
}

type ComponentResource struct {
	provider *CycloidProvider
}

func (r *ComponentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_component"
}

func (r *ComponentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_component.ComponentResourceSchema(ctx)
}

func (r *ComponentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	pv, ok := req.ProviderData.(*CycloidProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider data at Configure()",
			fmt.Sprintf("Expected *CycloidProvider, got: %T. Please report this issue.", req.ProviderData),
		)
		return
	}

	r.provider = pv
}

func (r *ComponentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var componentState componentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &componentState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Client

	org := getOrganizationCanonical(*r.provider, componentState.Organization)
	project := componentState.Project.ValueString()
	environment := componentState.Environment.ValueString()

	var _, canonical string
	var err error
	if componentState.Canonical.IsNull() || componentState.Canonical.IsUnknown() {
		_, canonical, err = NameOrCanonical(componentState.Name.ValueString(), componentState.Canonical.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("failed to infer canonical", err.Error())
			return
		}
	} else {
		_, canonical = componentState.Name.ValueString(), componentState.Canonical.ValueString()
	}

	component, notFound, diags := componentFetch(m, org, project, environment, canonical)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if notFound {
		// The component is not at the address state records. Before dropping it
		// (which would plan a create and orphan a moved component), look for it
		// elsewhere in the project: a move performed outside this resource's
		// last apply leaves it under a different environment. When found, adopt
		// its real environment into state; the config still names the old one,
		// so the next plan surfaces the delta and Update migrates it.
		//
		// A canonical is only unique per (project, environment), so the search
		// matches on the component ID recorded in private state, never on the
		// canonical alone: an unrelated component that shares the canonical in
		// another environment must not be adopted. Without a recorded ID the
		// resource is dropped, as before this search existed.
		wantID, idDiags := componentIDFromPrivate(ctx, req.Private)
		resp.Diagnostics.Append(idDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		movedEnv, movedComponent, moved, mdiags := findComponentInProject(m, org, project, environment, canonical, wantID)
		resp.Diagnostics.Append(mdiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !moved {
			resp.State.RemoveResource(ctx)
			return
		}
		environment = movedEnv
		componentState.Environment = types.StringValue(movedEnv)
		component = movedComponent
	}

	var inputVariables map[string]map[string]map[string]any
	var currentConfig map[string]map[string]map[string]any
	currentConfig, _, err = m.GetComponentConfig(org, project, environment, canonical, "", "", "", 0)
	if err != nil {
		if isComponentNotFoundError(err) {
			resp.Diagnostics.Append(
				ComponentToModel(ctx, org, nil, nil, nil, &componentState, false)...,
			)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &componentState)...)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("failed to get component config in org %q, project %q, environment %q", org, project, environment), err.Error())
		return
	}

	var inputDiags diag.Diagnostics
	inputVariables, inputDiags = getInputVariablesForRead(ctx, componentState, currentConfig)
	resp.Diagnostics.Append(inputDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(
		ComponentToModel(ctx, org, component, inputVariables, currentConfig, &componentState, true)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(setComponentIDPrivate(ctx, resp.Private, component)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &componentState)...)
}

func (r *ComponentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var componentPlan componentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &componentPlan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Client

	org := getOrganizationCanonical(*r.provider, componentPlan.Organization)
	project := componentPlan.Project.ValueString()
	environment := componentPlan.Environment.ValueString()

	var name, canonical string
	var err error
	if componentPlan.Canonical.IsNull() || componentPlan.Canonical.IsUnknown() {
		name, canonical, err = NameOrCanonical(componentPlan.Name.ValueString(), componentPlan.Canonical.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("failed to infer canonical", err.Error())
			return
		}
	} else {
		name, canonical = componentPlan.Name.ValueString(), componentPlan.Canonical.ValueString()
	}

	if _, _, err := m.ListComponents(org, project, environment); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list components in org %q, project %q, environment %q", org, project, environment), err.Error())
		return
	}

	stackRef := componentPlan.StackRef.ValueString()
	useCase := componentPlan.UseCase.ValueString()
	stackVersion := requestedStackVersion(componentPlan.StackVersion)
	description := componentPlan.Description.ValueStringPointer()

	var tag, branch, commit string
	if stackVersion != nil {
		versions, _, err := m.ListStackVersions(org, stackRef)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("stack_ref"),
				fmt.Sprintf("Failed to list version for stack %q in org %q", stackRef, org),
				err.Error(),
			)
			return
		}
		var notFound diag.Diagnostic
		tag, branch, commit, notFound = resolveStackVersionSelector(versions, stackVersion, stackRef)
		if notFound != nil {
			resp.Diagnostics.Append(notFound)
			return
		}
	}

	var inputVariables models.FormVariables
	var diags diag.Diagnostics
	if !componentPlan.InputVariables.IsNull() && !componentPlan.InputVariables.IsUnknown() {
		inputVariables, diags = dynamicValueToVariables(ctx, componentPlan.InputVariables)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	component, _, err := m.CreateOrUpdateComponent(org, project, environment, canonical, ptr.Value(description), name, stackRef, tag, branch, commit, useCase, "", inputVariables)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to create component %q in org %q, project %q, environment %q", canonical, org, project, environment), err.Error())
		return
	}

	currentConfig, _, err := m.GetComponentConfig(org, project, environment, canonical, "", "", "", ptr.Value(component.Version.ID))
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to fetch created config of create component %q in org %q, project %q, environment %q", canonical, org, project, environment), err.Error())
		return
	}

	resp.Diagnostics.Append(
		ComponentToModel(ctx, org, component, inputVariables, currentConfig, &componentPlan, false)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(setComponentIDPrivate(ctx, resp.Private, component)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &componentPlan)...)
}

func (r *ComponentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var componentState componentResourceModel
	var componentPlan componentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &componentState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &componentPlan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Client

	org := getOrganizationCanonical(*r.provider, componentPlan.Organization)
	project := componentPlan.Project.ValueString()
	environment := componentPlan.Environment.ValueString()

	var name, canonical string
	var err error
	if componentPlan.Canonical.IsNull() || componentPlan.Canonical.IsUnknown() {
		name, canonical, err = NameOrCanonical(componentPlan.Name.ValueString(), componentPlan.Canonical.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("failed to infer canonical", err.Error())
			return
		}
	} else {
		name, canonical = componentPlan.Name.ValueString(), componentPlan.Canonical.ValueString()
	}

	stackRef := componentPlan.StackRef.ValueString()
	useCase := componentPlan.UseCase.ValueString()
	stackVersion := requestedStackVersion(componentPlan.StackVersion)
	description := componentPlan.Description.ValueStringPointer()
	allowVersionUpdate := componentPlan.AllowVersionUpdate.ValueBool()
	allowVariableUpdate := componentPlan.AllowVariableUpdate.ValueBool()

	// A changed project or environment is a move, not an upsert at a new address.
	// Without this, CreateOrUpdateComponent below would create a second component
	// at the destination and leave the original orphaned at the source. Migrate
	// first, then let the normal update tail apply any other attribute change
	// from the same plan at the destination address. The canonical is preserved:
	// a canonical change forces a replacement and never reaches Update. A backend
	// older than the PROD-855 migrate fixes performs a partial move (see the
	// resource documentation's minimum backend version note).
	if componentMovedAddress(componentState, componentPlan) {
		srcProject := componentState.Project.ValueString()
		srcEnv := componentState.Environment.ValueString()
		if _, _, err := m.MigrateComponent(org, srcProject, srcEnv, canonical, project, environment, canonical, name); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("failed to migrate component %q from project %q environment %q to project %q environment %q in org %q", canonical, srcProject, srcEnv, project, environment, org),
				err.Error(),
			)
			return
		}
	}

	// The tail below always runs after a migrate, even when the plan changes
	// nothing but project/environment. Skipping the upsert in that case was
	// considered (TFPRO-76) and rejected: deciding "nothing else changed" means
	// diffing every other attribute, input_variables included, between state
	// and plan, and a wrong answer silently drops a real change. The upsert is
	// also what re-applies the planned variables and version at the
	// destination when an older backend performs only a partial move. The
	// cost is one extra idempotent upsert per move, which is cheap next to a
	// migrate.
	var variables models.FormVariables
	var diags diag.Diagnostics
	if !componentPlan.InputVariables.IsNull() && !componentPlan.InputVariables.IsUnknown() {
		variables, diags = dynamicValueToVariables(ctx, componentPlan.InputVariables)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Resolve the exact version to target. When version management is
	// disabled, keep pointing at the component's own current version instead
	// of leaving tag/branch/commit empty: an empty selector below resolves to
	// the stack's default version, which would silently move the component
	// off the version it is pinned to.
	var tag, branch, commit string
	if stackVersion != nil && allowVersionUpdate {
		versions, _, err := m.ListStackVersions(org, stackRef)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("stack_ref"),
				fmt.Sprintf("Failed to list version for stack %q in org %q", stackRef, org),
				err.Error(),
			)
			return
		}
		var notFound diag.Diagnostic
		tag, branch, commit, notFound = resolveStackVersionSelector(versions, stackVersion, stackRef)
		if notFound != nil {
			resp.Diagnostics.Append(notFound)
			return
		}
	} else {
		existingComponent, _, err := m.GetComponent(org, project, environment, canonical)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to get component %q in org %q, project %q, environment %q", canonical, org, project, environment), err.Error())
			return
		}
		tag, branch = currentVersionSelector(existingComponent.Version)
	}

	// Fetch the configuration the component would have with tag/branch/commit
	// applied; previously configured values are preserved.
	baseVars, _, err := m.GetComponentConfig(org, project, environment, canonical, tag, branch, commit, 0)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to get component config %q in org %q, project %q, environment %q", canonical, org, project, environment), err.Error())
		return
	}

	inputs := baseVars
	if allowVariableUpdate {
		inputs = mergeFormVariables(baseVars, variables)
	}

	component, _, err := m.CreateOrUpdateComponent(org, project, environment, canonical, ptr.Value(description), name, stackRef, tag, branch, commit, useCase, "", inputs)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to update component %q in org %q, project %q, environment %q", canonical, org, project, environment), err.Error())
		return
	}

	currentConfig, _, err := m.GetComponentConfig(org, project, environment, canonical, "", "", "", ptr.Value(component.Version.ID))
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to get component config %q in org %q, project %q, environment %q", canonical, org, project, environment), err.Error())
		return
	}

	resp.Diagnostics.Append(
		ComponentToModel(ctx, org, component, variables, currentConfig, &componentPlan, false)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(setComponentIDPrivate(ctx, resp.Private, component)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &componentPlan)...)
}

func (r *ComponentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var componentState componentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &componentState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !componentState.AllowDestroy.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("allow_destroy"),
			"Component deletion not allowed",
			"This resource must be deleted carefully. Deleting a component could lead to undelete resource in case of a stack using terraform. Please use the destroy step of your stack to delete this component.",
		)
		return
	}

	m := r.provider.Client

	org := getOrganizationCanonical(*r.provider, componentState.Organization)
	project := componentState.Project.ValueString()
	environment := componentState.Environment.ValueString()

	var canonical string
	var err error
	if componentState.Canonical.IsNull() || componentState.Canonical.IsUnknown() {
		resp.Diagnostics.AddError(
			"Component canonical not found in state",
			"Component canonical should be present in the state at this stage. This indicates an inconsistent state.",
		)
		return
	} else {
		canonical = componentState.Canonical.ValueString()
	}

	components, _, err := m.ListComponents(org, project, environment)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list components in org %q, project %q, environment %q", org, project, environment), err.Error())
		return
	}

	var component *models.Component
	for _, c := range components {
		if ptr.Value(c.Canonical) == canonical {
			component = c
			break
		}
	}

	if component != nil {
		deleteTimeout, tdiags := componentState.Timeouts.Delete(ctx, defaultComponentDeleteHookTimeout)
		resp.Diagnostics.Append(tdiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		var builds []*models.Build
		builds, err = deleteComponentRequest(m, org, project, environment, canonical)
		if err == nil && len(builds) > 0 {
			// on_delete hooks were triggered: the component is still there and
			// the hook job is expected to delete it. Keep state until it does.
			resp.Diagnostics.Append(waitComponentDeletedByHook(ctx, m, org, project, environment, canonical, builds, deleteTimeout, componentDeletePollInterval)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		if err != nil {
			if isComponentNotFoundError(err) {
				resp.Diagnostics.Append(
					ComponentToModel(ctx, org, nil, nil, nil, &componentState, false)...,
				)
				if resp.Diagnostics.HasError() {
					return
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &componentState)...)
				return
			}
			resp.Diagnostics.AddError(
				fmt.Sprintf("failed to delete component %q in org %q, project %q, environment %q", canonical, org, project, environment), err.Error(),
			)
			return
		}
	}

	resp.Diagnostics.Append(
		ComponentToModel(ctx, org, &models.Component{}, nil, nil, &componentState, false)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &componentState)...)
}

func getInputVariablesForRead(ctx context.Context, componentState componentResourceModel, currentConfig map[string]map[string]map[string]any) (map[string]map[string]map[string]any, diag.Diagnostics) {
	if componentState.AllowVariableUpdate.ValueBool() {
		userInputValue, diags := componentState.InputVariables.ToDynamicValue(ctx)
		if diags.HasError() {
			return nil, diags
		}
		userInput, diags := dynamicValueToVariables(ctx, userInputValue)
		if diags.HasError() {
			return nil, diags
		}

		return filterVariablesByUserInput(currentConfig, userInput), diags
	}

	variablesValue, diags := componentState.InputVariables.ToDynamicValue(ctx)
	if diags.HasError() {
		return nil, diags
	}
	fromState, diags := dynamicValueToVariables(ctx, variablesValue)
	if diags.HasError() {
		return nil, diags
	}
	if len(currentConfig) == 0 {
		return fromState, diags
	}
	return applyAPIDriftToInputVariables(fromState, currentConfig), diags
}

func filterVariablesByUserInput(currentConfig, userInput map[string]map[string]map[string]any) map[string]map[string]map[string]any {
	filtered := make(map[string]map[string]map[string]any)

	for sectionName, section := range userInput {
		if _, exists := currentConfig[sectionName]; !exists {
			continue
		}

		filteredSection := make(map[string]map[string]any)
		for groupName, group := range section {
			if _, exists := currentConfig[sectionName][groupName]; !exists {
				continue
			}

			filteredGroup := make(map[string]any)
			for keyName := range group {
				if _, exists := currentConfig[sectionName][groupName][keyName]; exists {
					filteredGroup[keyName] = currentConfig[sectionName][groupName][keyName]
				}
			}

			if len(filteredGroup) > 0 {
				filteredSection[groupName] = filteredGroup
			}
		}

		if len(filteredSection) > 0 {
			filtered[sectionName] = filteredSection
		}
	}

	return filtered
}

func applyAPIDriftToInputVariables(fromState, api map[string]map[string]map[string]any) map[string]map[string]map[string]any {
	out := cloneNestedStringMapAny(fromState)
	for sec, groups := range fromState {
		apiSec := api[sec]
		if apiSec == nil {
			continue
		}
		for grp, vars := range groups {
			apiGrp := apiSec[grp]
			if apiGrp == nil {
				continue
			}
			for k, stateVal := range vars {
				apiVal, ok := apiGrp[k]
				if !ok {
					continue
				}
				if variableValuesEqual(stateVal, apiVal) {
					continue
				}
				if out[sec] == nil {
					out[sec] = make(map[string]map[string]any)
				}
				if out[sec][grp] == nil {
					out[sec][grp] = make(map[string]any)
				}
				out[sec][grp][k] = apiVal
			}
		}
	}
	return out
}

func cloneNestedStringMapAny(m map[string]map[string]map[string]any) map[string]map[string]map[string]any {
	if m == nil {
		return map[string]map[string]map[string]any{}
	}
	out := make(map[string]map[string]map[string]any, len(m))
	for sec, grps := range m {
		out[sec] = make(map[string]map[string]any, len(grps))
		for grp, vars := range grps {
			out[sec][grp] = make(map[string]any, len(vars))
			for k, v := range vars {
				out[sec][grp][k] = v
			}
		}
	}
	return out
}

func variableValuesEqual(a, b any) bool {
	ja, errA := json.Marshal(a)
	jb, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return fmt.Sprint(a) == fmt.Sprint(b)
	}
	return string(ja) == string(jb)
}

func ComponentToModel(ctx context.Context, org string, component *models.Component, inputVariables, currentConfig map[string]map[string]map[string]any, componentState *componentResourceModel, refreshInputVariables bool) diag.Diagnostics {
	if component == nil {
		componentState.Organization = types.StringValue(org)
		componentState.Project = types.StringNull()
		componentState.Environment = types.StringNull()
		componentState.Name = types.StringNull()
		componentState.Canonical = types.StringNull()
		componentState.Description = types.StringNull()
		componentState.StackRef = types.StringNull()
		componentState.StackVersion = types.StringNull()
		componentState.UseCase = types.StringNull()
		componentState.AllowVersionUpdate = types.BoolNull()
		componentState.AllowVariableUpdate = types.BoolNull()
		componentState.AllowDestroy = types.BoolNull()
		componentState.InputVariables = types.DynamicNull()
		componentState.CurrentConfig = types.DynamicNull()
		return nil
	}

	componentState.Organization = types.StringValue(org)
	if component.Project != nil {
		componentState.Project = types.StringPointerValue(component.Project.Canonical)
	} else {
		componentState.Project = types.StringNull()
	}
	if component.Environment != nil {
		componentState.Environment = types.StringValue(ptr.Value(component.Environment.Canonical))
	} else {
		componentState.Environment = types.StringNull()
	}
	componentState.Name = types.StringPointerValue(component.Name)
	componentState.Canonical = types.StringPointerValue(component.Canonical)
	if component.Description == "" && componentState.Description.IsNull() {
		componentState.Description = types.StringNull()
	} else if component.Description == "" {
		componentState.Description = types.StringValue("")
	} else {
		componentState.Description = types.StringValue(component.Description)
	}
	componentState.StackRef = types.StringPointerValue(ptr.Value(component.ServiceCatalog).Ref)
	componentState.UseCase = types.StringValue(component.UseCase)
	// stack_version is never returned by the API after creation; preserve the existing
	// state value so Terraform does not see spurious null drift on every refresh.

	var diags diag.Diagnostics
	if refreshInputVariables || inputVariables != nil {
		componentState.InputVariables, diags = dynamic.AnyToDynamicValue(ctx, inputVariables)
		if diags.HasError() {
			return diags
		}
	}

	componentState.CurrentConfig, diags = dynamic.AnyToDynamicValue(ctx, currentConfig)
	if diags.HasError() {
		return diags
	}

	// stack_version and input_variables are Optional+Computed, so when the user
	// leaves them unset the create/update plan marks them "known after apply".
	// The API never echoes stack_version and returns no input variables for a
	// component with no user input, so collapse any leftover unknown to null —
	// otherwise the framework rejects the result as "invalid result object".
	if componentState.StackVersion.IsUnknown() {
		componentState.StackVersion = types.StringNull()
	}
	if componentState.InputVariables.IsUnknown() {
		componentState.InputVariables = types.DynamicNull()
	}

	return nil
}

func dynamicValueToVariables(ctx context.Context, dynamicValue types.Dynamic) (map[string]map[string]map[string]any, diag.Diagnostics) {
	output := make(map[string]map[string]map[string]any)
	var diags diag.Diagnostics

	if dynamicValue.IsNull() || dynamicValue.IsUnknown() {
		return map[string]map[string]map[string]any{}, nil
	}

	underlyingValue := dynamicValue.UnderlyingValue()
	if underlyingValue.IsNull() || underlyingValue.IsUnknown() {
		return map[string]map[string]map[string]any{}, nil
	}

	switch valueType := underlyingValue.(type) {
	case types.Object:
		_, attrValues := valueType.AttributeTypes(ctx), valueType.Attributes()
		for section, sectionAttrValue := range attrValues {
			sectionObject, ok := sectionAttrValue.(types.Object)
			if !ok {
				diags.AddAttributeError(path.Root("variables"), "sections are missing in variables.", "this may indicate an invalid payload from the API.")
				return nil, diags
			}

			_, sectionValues := sectionObject.AttributeTypes(ctx), sectionObject.Attributes()
			for group, groupAttrValue := range sectionValues {
				groupObject, ok := groupAttrValue.(types.Object)
				if !ok {
					diags.AddAttributeError(path.Root("variables"), "groups are missing in variables.", "this may indicate an invalid payload from the API.")
					return nil, diags
				}

				_, groupValues := groupObject.AttributeTypes(ctx), groupObject.Attributes()
				for key, keyAttrValue := range groupValues {
					keyOutput, diags := dynamic.AttrValueToAny(ctx, keyAttrValue)
					if diags.HasError() {
						return nil, diags
					}

					if output[section] == nil {
						output[section] = map[string]map[string]any{}
					}

					if output[section][group] == nil {
						output[section][group] = map[string]any{}
					}

					output[section][group][key] = keyOutput
				}
			}
		}

	default:
		return nil, diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Failed to convert dynamic value to variables",
				fmt.Sprintf("Unsupported value type: %T. Expected map[string]interface{}", valueType),
			),
		}
	}

	return output, nil
}

// requestedStackVersion reports the stack_version the configuration asks for, or
// nil when it asks for none
//
// ValueStringPointer hands back a pointer to "" for an unknown value as much as
// for an explicit empty string, and neither is a version to resolve: stack_version
// is Optional+Computed, so leaving it unset plans as unknown, and "" has always
// meant "let the API pick the default" — rejecting it now would fail
// configurations that never asked for a specific version
func requestedStackVersion(stackVersion types.String) *string {
	if stackVersion.IsNull() || stackVersion.IsUnknown() || stackVersion.ValueString() == "" {
		return nil
	}

	return stackVersion.ValueStringPointer()
}

// resolveStackVersionSelector matches stackVersion against the stack's known
// versions and reports the tag/branch/commit selector to send to the API
//
// notFound is non-nil when a requested version matched nothing. It has to be an
// error rather than a silent skip: with an empty selector the API resolves the
// stack's DEFAULT version, so the component would be deployed from something the
// configuration never asked for (ENG-302). A nil stackVersion is not a failure —
// it means no version was requested and the API's default applies
func resolveStackVersionSelector(
	versions []*apiclient.StackVersion,
	stackVersion *string,
	stackRef string,
) (tag, branch, commit string, notFound diag.Diagnostic) {
	if stackVersion == nil {
		return "", "", "", nil
	}

	tag, branch, commit = matchStackVersion(versions, stackVersion)
	if tag == "" && branch == "" && commit == "" {
		return "", "", "", stackVersionNotFoundDiag(*stackVersion, stackRef, versions)
	}

	return tag, branch, commit, nil
}

// stackVersionNotFoundDiag reports a configured stack_version that matches none
// of the stack's known versions
//
// This has to be an error rather than a silent skip: with an empty
// tag/branch/commit selector the API resolves the stack's DEFAULT version, so the
// component would be deployed from something the configuration never asked for.
// A stack_version pinned to a commit hash stops resolving as soon as the tag
// carrying it is re-cut on another commit, which is exactly when that silent
// switch used to happen (ENG-302)
func stackVersionNotFoundDiag(
	stackVersion, stackRef string,
	versions []*apiclient.StackVersion,
) diag.Diagnostic {
	known := make([]string, 0, len(versions))
	for _, v := range versions {
		if v == nil {
			continue
		}
		entry := fmt.Sprintf("%s (%s)", ptr.Value(v.Name), ptr.Value(v.Type))
		// The commit is part of the listing because a stack_version may pin one
		// directly, so a rejected hash can be compared against what is resolvable
		if commitHash := ptr.Value(v.CommitHash); commitHash != "" {
			entry += " at " + commitHash
		}
		known = append(known, entry)
	}
	sort.Strings(known)
	if len(known) > stackVersionSuggestionLimit {
		known = append(known[:stackVersionSuggestionLimit:stackVersionSuggestionLimit],
			fmt.Sprintf("… and %d more", len(known)-stackVersionSuggestionLimit))
	}

	detail := strings.Join([]string{
		fmt.Sprintf("%q does not match any tag, branch or commit known for stack %q.", stackVersion, stackRef),
		"",
		"A commit hash only stays resolvable while a tag or branch still points at it, so " +
			"pinning one breaks as soon as that tag is moved to another commit. Pin the tag " +
			"or branch name instead, and if the version was added to the catalog repository " +
			"just now, refresh its versions and retry.",
		"",
		"Known versions: " + strings.Join(known, ", "),
	}, "\n")

	return diag.NewAttributeErrorDiagnostic(
		path.Root("stack_version"),
		fmt.Sprintf("Unknown stack_version %q", stackVersion),
		detail,
	)
}

// stackVersionSuggestionLimit caps how many known versions are listed back in a
// stackVersionNotFoundDiag detail, so a catalog with hundreds of tags does not
// produce an unreadable error
const stackVersionSuggestionLimit = 20

// matchStackVersion resolves stackVersion to the tag/branch/commit selector
// CreateOrUpdateComponent takes: a name against the tag and branch versions,
// otherwise the hash against the commit each listed version points at
func matchStackVersion(versions []*apiclient.StackVersion, stackVersion *string) (tag, branch, commit string) {
	if stackVersion == nil {
		return "", "", ""
	}

	targetVersion := *stackVersion

	for _, version := range versions {
		if version == nil {
			continue
		}

		versionName := ptr.Value(version.Name)

		switch ptr.Value(version.Type) {
		case "tag":
			if versionName == targetVersion {
				tag = versionName
			}
		case "branch":
			if versionName == targetVersion {
				branch = versionName
			}
		}

		// A commit hash is never a version's Name — svccatsrc/version.Type only
		// ever holds "tag" or "branch" — so a raw hash can only be recognized by
		// the commit a listed version currently points at, which is also how the
		// CLI's resolveStackVersion matches one. That is why a hash stops
		// resolving as soon as its tag is re-cut elsewhere (ENG-302)
		if commit == "" && ptr.Value(version.CommitHash) == targetVersion {
			commit = targetVersion
		}
	}

	return tag, branch, commit
}

// currentVersionSelector returns the tag or branch identifying v, so it can be
// passed back to the API to resolve to the exact same version rather than
// falling through to the stack's default version.
func currentVersionSelector(v *models.ServiceCatalogSourceVersion) (tag, branch string) {
	if v == nil {
		return "", ""
	}

	switch ptr.Value(v.Type) {
	case "tag":
		tag = ptr.Value(v.Name)
	case "branch":
		branch = ptr.Value(v.Name)
	}

	return tag, branch
}

// mergeFormVariables overlays overlay's values onto base, so a partial
// variables map only changes the keys it explicitly sets. base is only
// shallow-cloned at the top level, so touched sections/groups are mutated in
// place — callers must not reuse base afterward.
func mergeFormVariables(base, overlay models.FormVariables) models.FormVariables {
	merged := maps.Clone(base)
	if merged == nil {
		merged = make(models.FormVariables, len(overlay))
	}

	for section, groups := range overlay {
		g, ok := merged[section]
		if !ok {
			g = make(map[string]map[string]interface{})
			merged[section] = g
		}
		for group, entities := range groups {
			e, ok := g[group]
			if !ok {
				e = make(map[string]interface{})
				g[group] = e
			}
			maps.Insert(e, maps.All(entities))
		}
	}

	return merged
}

// componentFetch lists components and returns the one matching canonical.
// Returns (component, notFound bool, diags). notFound=true means the org, project,
// environment, or component is gone.
func componentFetch(m apiclient.APIClient, org, project, environment, canonical string) (*models.Component, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	components, _, err := m.ListComponents(org, project, environment)
	if err != nil {
		if isNotFoundError(err) {
			return nil, true, nil
		}
		diags.AddError(fmt.Sprintf("failed to list components in org %q, project %q, environment %q", org, project, environment), err.Error())
		return nil, false, diags
	}
	for _, c := range components {
		if ptr.Value(c.Canonical) == canonical {
			return c, false, nil
		}
	}
	return nil, true, nil
}

func isComponentNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "component was not found") ||
		strings.Contains(errMsg, "getcomponentconfignotfound")
}

// componentMovedAddress reports whether the plan places the component in a
// different project or environment than state records. A canonical change is not
// part of the test: it forces a replacement, so it never reaches Update.
func componentMovedAddress(state, plan componentResourceModel) bool {
	return state.Environment.ValueString() != plan.Environment.ValueString() ||
		state.Project.ValueString() != plan.Project.ValueString()
}

// componentIDPrivateKey is the private state key holding the backend ID of the
// component a resource manages. A migrate keeps the ID, so it identifies the
// component across environments where the canonical cannot.
const componentIDPrivateKey = "component_id"

type privateGetter interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

type privateSetter interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

// setComponentIDPrivate records the component's backend ID in private state.
func setComponentIDPrivate(ctx context.Context, priv privateSetter, component *models.Component) diag.Diagnostics {
	if priv == nil || component == nil || component.ID == nil {
		return nil
	}
	// Encoded with json.Marshal and decoded with json.Unmarshal in
	// componentIDFromPrivate. A uint32 marshals to a bare decimal ("42"), so
	// this is byte-identical to the fmt "%d" encoding earlier builds wrote and
	// state they recorded still decodes.
	raw, err := json.Marshal(*component.ID)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("failed to encode component ID for private state", err.Error())
		return diags
	}
	return priv.SetKey(ctx, componentIDPrivateKey, raw)
}

// componentIDFromPrivate returns the component ID recorded in private state, or
// 0 when none is (state written by an older provider version).
func componentIDFromPrivate(ctx context.Context, priv privateGetter) (uint32, diag.Diagnostics) {
	if priv == nil {
		return 0, nil
	}
	raw, diags := priv.GetKey(ctx, componentIDPrivateKey)
	if diags.HasError() || len(raw) == 0 {
		return 0, diags
	}
	var id uint32
	if err := json.Unmarshal(raw, &id); err != nil {
		// An unreadable ID only disables move detection; it is not fatal, but
		// say so, since it means a moved component would be dropped from state.
		diags.AddWarning(
			"unreadable component ID in private state",
			fmt.Sprintf("Could not decode the recorded component ID (%v); detection of a component moved to another environment is disabled for this read.", err),
		)
		return 0, diags
	}
	return id, diags
}

// findComponentInProject searches the other environments of a project for the
// component with the given backend ID. It backs Read's moved-vs-gone decision:
// a component missing from its recorded environment may have been moved
// elsewhere in the project rather than deleted. It returns the environment it
// was found in.
//
// A zero ID means no identity was recorded, so a moved component cannot be told
// apart from an unrelated one sharing the canonical: the search reports not
// moved. A 403 on the project or on an environment (an API key scoped to its
// own environment) is skipped the same way, keeping the drop-from-state
// behavior such keys had before the search existed.
//
// Only a typed 404 (isStrictNotFoundError) counts as "not there". The loose
// isNotFoundError message match is deliberately not used: an error it
// misclassified would read as "not moved" and drop the resource from state,
// whereas here any other error surfaces as a diagnostic and state is kept.
//
// Cost: one GetProject plus one ListComponents per other environment of the
// project. There is no cheaper lookup in the API client: there is no
// get-component-by-ID route, and the org-wide GET /organizations/{org}/components
// is not wired into the client and filters by canonical only, so it would still
// need an ID match on the result. The search only runs on Read's not-found
// path, i.e. once per refresh of a component that left its recorded address.
func findComponentInProject(m apiclient.APIClient, org, project, skipEnv, canonical string, wantID uint32) (string, *models.Component, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if wantID == 0 {
		return "", nil, false, diags
	}

	proj, _, err := m.GetProject(org, project)
	if err != nil {
		if isStrictNotFoundError(err) || isForbiddenError(err) {
			return "", nil, false, diags
		}
		diags.AddError(fmt.Sprintf("failed to get project %q in org %q", project, org), err.Error())
		return "", nil, false, diags
	}

	for _, env := range proj.Environments {
		envCanonical := ptr.Value(env.Canonical)
		if envCanonical == "" || envCanonical == skipEnv {
			continue
		}
		components, _, err := m.ListComponents(org, project, envCanonical)
		if err != nil {
			if isStrictNotFoundError(err) || isForbiddenError(err) {
				continue
			}
			diags.AddError(fmt.Sprintf("failed to list components in org %q, project %q, environment %q", org, project, envCanonical), err.Error())
			return "", nil, false, diags
		}
		for _, c := range components {
			if ptr.Value(c.Canonical) == canonical && ptr.Value(c.ID) == wantID {
				return envCanonical, c, true, diags
			}
		}
	}

	return "", nil, false, diags
}

// parseComponentImportID splits a cycloid_component import ID. The resource has
// no id attribute: a component is identified by its full address, so the import
// ID has to carry all four segments.
func parseComponentImportID(id string) (org, project, environment, canonical string, err error) {
	const shape = "organization/project/environment/canonical"

	parts := strings.SplitN(id, "/", 4)
	if len(parts) != 4 {
		return "", "", "", "", fmt.Errorf("expected an import ID of the form %s, got %q", shape, id)
	}
	// SplitN leaves everything past the third separator in the last segment, so
	// a five-segment ID would otherwise be accepted with a canonical of "a/b".
	if strings.Contains(parts[3], "/") {
		return "", "", "", "", fmt.Errorf("expected an import ID of the form %s with exactly four segments, got %q", shape, id)
	}
	for _, part := range parts {
		if part == "" {
			return "", "", "", "", fmt.Errorf("expected an import ID of the form %s with no empty segment, got %q", shape, id)
		}
	}

	return parts[0], parts[1], parts[2], parts[3], nil
}

// ImportState adopts an existing component from its address. Only the four
// identity attributes are written here; Read fills in the rest on the refresh
// that immediately follows the import.
func (r *ComponentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	org, project, environment, canonical, err := parseComponentImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("invalid import ID format", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), org)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), project)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment"), environment)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("canonical"), canonical)...)
}

// defaultComponentDeleteHookTimeout bounds how long Delete waits for an
// on_delete hook to remove the component; override with timeouts.delete.
const defaultComponentDeleteHookTimeout = 5 * time.Minute

// componentDeletePollInterval is how often Delete re-checks a component whose
// on_delete hook is running. A var so unit tests can shorten it.
var componentDeletePollInterval = 5 * time.Second

// deleteComponentRequest issues DELETE on the component without skip_hooks and
// returns the hook builds the API triggered. The API answers 204 when it
// deleted the component, and 200 with the triggered builds when the component
// has on_delete hooks: in that case it is NOT deleted yet, the hook job is
// expected to call back with DELETE ?skip_hooks=true. apiclient.DeleteComponent
// discards that body, so the request is made here to keep it.
func deleteComponentRequest(m apiclient.APIClient, org, project, environment, canonical string) ([]*models.Build, error) {
	var builds []*models.Build
	_, err := m.GenericRequest(apiclient.Request{
		Method:       "DELETE",
		Organization: &org,
		Route:        []string{"organizations", org, "projects", project, "environments", environment, "components", canonical},
	}, &builds)
	if err != nil {
		return nil, err
	}
	return builds, nil
}

// waitComponentDeletedByHook polls until the component is gone from its
// environment or the timeout ends. It never reports success while the API
// still returns the component, so Terraform keeps it in state.
func waitComponentDeletedByHook(ctx context.Context, m apiclient.APIClient, org, project, environment, canonical string, builds []*models.Build, timeout, interval time.Duration) diag.Diagnostics {
	var diags diag.Diagnostics
	hooks := describeHookBuilds(builds)
	deadline := time.Now().Add(timeout)
	for {
		gone, err := componentGone(m, org, project, environment, canonical)
		if err != nil {
			diags.AddError(
				fmt.Sprintf("failed to check component %q after triggering its on_delete hook", canonical),
				fmt.Sprintf("The delete of component %q in org %q, project %q, environment %q was handed to its on_delete hook (%s), and checking whether the hook deleted it failed: %s. The component is kept in state.", canonical, org, project, environment, hooks, err),
			)
			return diags
		}
		if gone {
			return diags
		}
		if !time.Now().Add(interval).Before(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			diags.AddError(
				fmt.Sprintf("interrupted while waiting for the on_delete hook of component %q", canonical),
				fmt.Sprintf("The delete of component %q in org %q, project %q, environment %q was handed to its on_delete hook (%s). The component still exists and is kept in state.", canonical, org, project, environment, hooks),
			)
			return diags
		case <-time.After(interval):
		}
	}
	diags.AddError(
		fmt.Sprintf("component %q still exists after its on_delete hook ran", canonical),
		fmt.Sprintf("The delete of component %q in org %q, project %q, environment %q was handed to its on_delete hook (%s), and the component still exists after %s. The hook job is expected to delete it. The component is kept in state: check the hook build, then run destroy again, or raise timeouts.delete if the hook is just slow.", canonical, org, project, environment, hooks, timeout),
	)
	return diags
}

// componentGone reports whether the component is absent from its environment.
// Only a typed 404 or a list without the canonical counts as gone: any other
// error is returned, so a misread error never drops state.
func componentGone(m apiclient.APIClient, org, project, environment, canonical string) (bool, error) {
	components, _, err := m.ListComponents(org, project, environment)
	if err != nil {
		var apiErr *apiclient.APIResponseError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return true, nil
		}
		return false, err
	}
	for _, c := range components {
		if ptr.Value(c.Canonical) == canonical {
			return false, nil
		}
	}
	return true, nil
}

// describeHookBuilds names the triggered hook builds, e.g.
// "pipeline p1 job on-delete build 3".
func describeHookBuilds(builds []*models.Build) string {
	parts := make([]string, 0, len(builds))
	for _, b := range builds {
		if b == nil {
			continue
		}
		var part []string
		if b.PipelineName != "" {
			part = append(part, "pipeline "+b.PipelineName)
		}
		if b.JobName != "" {
			part = append(part, "job "+b.JobName)
		}
		if b.Name != nil && *b.Name != "" {
			part = append(part, "build "+*b.Name)
		}
		if len(part) > 0 {
			parts = append(parts, strings.Join(part, " "))
		}
	}
	if len(parts) == 0 {
		return "on_delete hook build"
	}
	return strings.Join(parts, ", ")
}
