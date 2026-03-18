package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/dynamic"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_component"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ComponentResource{}

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

	m := r.provider.Middleware

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

	components, err := m.ListComponents(org, project, environment)
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
	if component == nil {
		resp.Diagnostics.Append(
			ComponentToModel(ctx, org, nil, nil, nil, &componentState)...,
		)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &componentState)...)
		return
	}

	var inputVariables map[string]map[string]map[string]any
	var currentConfig map[string]map[string]map[string]any
	var diags diag.Diagnostics
	currentConfig, err = m.GetComponentConfig(org, project, environment, canonical)
	if err != nil {
		if isComponentNotFoundError(err) {
			resp.Diagnostics.Append(
				ComponentToModel(ctx, org, nil, nil, nil, &componentState)...,
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

	inputVariables, diags = getInputVariablesForRead(ctx, componentState, currentConfig)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(
		ComponentToModel(ctx, org, component, inputVariables, currentConfig, &componentState)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &componentState)...)
}

func (r *ComponentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var componentPlan componentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &componentPlan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Middleware

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

	components, err := m.ListComponents(org, project, environment)
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

	stackRef := componentPlan.StackRef.ValueString()
	useCase := componentPlan.UseCase.ValueString()
	stackVersion := componentPlan.StackVersion.ValueStringPointer()
	description := componentPlan.Description.ValueStringPointer()

	var tag, branch, commit string
	if stackVersion != nil {
		versions, err := m.ListStackVersions(org, stackRef)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("stack_ref"),
				fmt.Sprintf("Failed to list version for stack %q in org %q", stackRef, org),
				err.Error(),
			)
			return
		}
		tag, branch, commit = matchStackVersion(versions, stackVersion)
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

	component, err = m.CreateAndConfigureComponent(org, project, environment, canonical, ptr.Value(description), name, stackRef, tag, branch, commit, useCase, "", inputVariables)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to create component %q in org %q, project %q, environment %q", canonical, org, project, environment), err.Error())
		return
	}

	currentConfig, err := m.GetComponentConfig(org, project, environment, canonical)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to fetch created config of create component %q in org %q, project %q, environment %q", canonical, org, project, environment), err.Error())
		return
	}

	resp.Diagnostics.Append(
		ComponentToModel(ctx, org, component, inputVariables, currentConfig, &componentPlan)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

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

	m := r.provider.Middleware

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
	stackVersion := componentPlan.StackVersion.ValueStringPointer()
	description := componentPlan.Description.ValueStringPointer()
	allowVersionUpdate := componentPlan.AllowVersionUpdate.ValueBool()
	allowVariableUpdate := componentPlan.AllowVariableUpdate.ValueBool()

	var variables models.FormVariables
	var diags diag.Diagnostics
	if !componentPlan.InputVariables.IsNull() && !componentPlan.InputVariables.IsUnknown() {
		variables, diags = dynamicValueToVariables(ctx, componentPlan.InputVariables)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var inputs models.FormVariables
	if allowVariableUpdate {
		inputs = variables
	}

	var tag, branch, commit string
	if stackVersion != nil && allowVersionUpdate {
		versions, err := m.ListStackVersions(org, stackRef)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("stack_ref"),
				fmt.Sprintf("Failed to list version for stack %q in org %q", stackRef, org),
				err.Error(),
			)
			return
		}
		tag, branch, commit = matchStackVersion(versions, stackVersion)
	} else {
		component, err := m.GetComponent(org, project, environment, canonical)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to get component %q in org %q, project %q, environment %q", canonical, org, project, environment), err.Error())
			return
		}

		if component.Version != nil {
			switch ptr.Value(component.Version.Type) {
			case "tag":
				tag = ptr.Value(component.Version.Name)
			case "branch":
				branch = ptr.Value(component.Version.Name)
			default:
				commit = ptr.Value(component.Version.CommitHash)
			}
		}
	}

	component, err := m.CreateAndConfigureComponent(org, project, environment, canonical, ptr.Value(description), name, stackRef, tag, branch, commit, useCase, "", inputs)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to update component %q in org %q, project %q, environment %q", canonical, org, project, environment), err.Error())
		return
	}

	currentConfig, err := m.GetComponentConfig(org, project, environment, canonical)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to get component config %q in org %q, project %q, environment %q", canonical, org, project, environment), err.Error())
		return
	}

	resp.Diagnostics.Append(
		ComponentToModel(ctx, org, component, variables, currentConfig, &componentPlan)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

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

	m := r.provider.Middleware

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

	components, err := m.ListComponents(org, project, environment)
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
		err = m.DeleteComponent(org, project, environment, canonical)
		if err != nil {
			if isComponentNotFoundError(err) {
				resp.Diagnostics.Append(
					ComponentToModel(ctx, org, nil, nil, nil, &componentState)...,
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
		ComponentToModel(ctx, org, &models.Component{}, nil, nil, &componentState)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &componentState)...)
}

func getInputVariablesForRead(ctx context.Context, componentState componentResourceModel, _ map[string]map[string]map[string]any) (map[string]map[string]map[string]any, diag.Diagnostics) {
	variablesValue, diags := componentState.InputVariables.ToDynamicValue(ctx)
	if diags.HasError() {
		return nil, diags
	}
	inputVariables, diags := dynamicValueToVariables(ctx, variablesValue)
	if diags.HasError() {
		return nil, diags
	}
	return inputVariables, diags
}

func ComponentToModel(ctx context.Context, org string, component *models.Component, inputVariables map[string]map[string]map[string]any, currentConfig map[string]map[string]map[string]any, componentState *componentResourceModel) diag.Diagnostics {
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
	if component.Version != nil {
		switch ptr.Value(component.Version.Type) {
		case "tag":
			componentState.StackVersion = types.StringPointerValue(component.Version.Name)
		case "branch":
			componentState.StackVersion = types.StringPointerValue(component.Version.Name)
		default:
			componentState.StackVersion = types.StringPointerValue(component.Version.CommitHash)
		}
	} else {
		componentState.StackVersion = types.StringNull()
	}

	var diags diag.Diagnostics
	componentState.CurrentConfig, diags = dynamic.AnyToDynamicValue(ctx, currentConfig)
	if diags.HasError() {
		return diags
	}

	return nil
}

func dynamicValueToVariables(ctx context.Context, dynamicValue types.Dynamic) (map[string]map[string]map[string]any, diag.Diagnostics) {
	var output = make(map[string]map[string]map[string]any)
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

func matchStackVersion(versions []*models.ServiceCatalogSourceVersion, stackVersion *string) (tag, branch, commit string) {
	if stackVersion == nil {
		return "", "", ""
	}

	targetVersion := *stackVersion

	for _, version := range versions {
		if version == nil {
			continue
		}

		versionType := ptr.Value(version.Type)
		versionName := ptr.Value(version.Name)

		switch versionType {
		case "tag":
			if versionName == targetVersion {
				tag = versionName
			}
		case "branch":
			if versionName == targetVersion {
				branch = versionName
			}
		default:
			if ptr.Value(version.CommitHash) == targetVersion {
				commit = ptr.Value(version.CommitHash)
			}
		}
	}

	return tag, branch, commit
}

func isComponentNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "component was not found") ||
		strings.Contains(errMsg, "getcomponentconfignotfound")
}
