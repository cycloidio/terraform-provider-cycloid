package provider

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	apiclient "github.com/cycloidio/cycloid-cli/cmd/apiclient"
	"github.com/cycloidio/cycloid-cli/gen/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/icons"
	"github.com/cycloidio/terraform-provider-cycloid/resource_environment"
	"github.com/cycloidio/cycloid-cli/utils/ptr"
)

var _ resource.Resource = (*environmentResource)(nil)

func NewEnvironmentResource() resource.Resource {
	return &environmentResource{}
}

type environmentResourceModel resource_environment.EnvironmentModel

type environmentResource struct {
	provider *CycloidProvider
}

func (p *environmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (p *environmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_environment.EnvironmentResourceSchema(ctx)
}

func (p *environmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	p.provider = pv
}

func (p *environmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data environmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := p.provider.Middleware
	canonical := data.Canonical.ValueString()
	org := getOrganizationCanonical(*p.provider, data.Organization)
	project := data.Project.ValueString()

	projectEnvs, _, err := m.ListProjectEnvs(org, project)
	if err != nil {
		resp.Diagnostics.AddError("failed to fetch environment from API while reading state", err.Error())
		return
	}

	found := false
	for _, e := range projectEnvs {
		if ptr.Value(e.Canonical) == canonical {
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.Append(environmentToValue(ctx, org, project, &models.Environment{}, &data)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	environment, _, err := m.GetOrgEnv(org, canonical)
	if err != nil {
		resp.Diagnostics.AddError("failed to fetch org environment from API while reading state", err.Error())
		return
	}

	resp.Diagnostics.Append(environmentToValue(ctx, org, project, environment, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (p *environmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data environmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, d := p.createOrUpdateEnvironment(ctx, data, false)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (p *environmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data environmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, d := p.createOrUpdateEnvironment(ctx, data, true)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (p *environmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data environmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := p.provider.Middleware
	org := getOrganizationCanonical(*p.provider, data.Organization)
	project := data.Project.ValueString()
	canonical := data.Canonical.ValueString()

	_, err := m.UnlinkEnvFromProject(org, project, canonical, apiclient.DeleteOptions{})
	if err != nil {
		resp.Diagnostics.AddError("failed to unlink environment from project while deleting resource", err.Error())
		return
	}

	resp.Diagnostics.Append(environmentToValue(ctx, org, project, &models.Environment{}, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func environmentColor(environment *models.Environment) *string {
	if environment == nil || environment.EnvironmentType == nil {
		return nil
	}
	return environment.EnvironmentType.Color
}

func environmentToValue(ctx context.Context, org, project string, environment *models.Environment, data *environmentResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if environment == nil {
		diags.AddError("environment is nil for conversion", "This should not happen, please contact the plugin maintainer.")
		return diags
	}

	data.Canonical = types.StringPointerValue(environment.Canonical)
	data.Name = types.StringValue(environment.Name)
	data.Organization = types.StringValue(org)
	data.Color = types.StringPointerValue(environmentColor(environment))
	data.Project = types.StringValue(project)
	data.Description = types.StringValue(environment.Description)
	data.ID = ptrUint32ToInt64(environment.ID)
	data.CreatedAt = ptrUint64ToInt64(environment.CreatedAt)
	data.UpdatedAt = ptrUint64ToInt64(environment.UpdatedAt)

	if environment.EnvironmentType != nil {
		data.Type = types.StringPointerValue(environment.EnvironmentType.Canonical)
	} else {
		data.Type = types.StringNull()
	}

	if environment.Owner != nil && environment.Owner.Username != nil {
		data.Owner = types.StringPointerValue(environment.Owner.Username)
	} else {
		data.Owner = types.StringValue("")
	}

	// CloudAccountCanonicals and Variables are PATCH-semantic Optional-only attributes.
	// They are NOT updated from the API in Read — state is preserved from prior plan/state.
	// They ARE set after Create/Update in createOrUpdateEnvironment.

	return diags
}

func envTypeFromData(data environmentResourceModel) string {
	if !data.Type.IsNull() && !data.Type.IsUnknown() {
		return data.Type.ValueString()
	}
	return ""
}

func envTypeFromCurrent(environment *models.Environment) string {
	if environment != nil && environment.EnvironmentType != nil && environment.EnvironmentType.Canonical != nil {
		return *environment.EnvironmentType.Canonical
	}
	return ""
}

func (p *environmentResource) createOrUpdateEnvironment(ctx context.Context, incoming environmentResourceModel, isUpdate bool) (environmentResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var data environmentResourceModel

	org := getOrganizationCanonical(*p.provider, incoming.Organization)
	name := incoming.Name.ValueString()
	canonical := incoming.Canonical.ValueString()
	project := incoming.Project.ValueString()
	color := incoming.Color.ValueString()
	if color == "" {
		color = icons.RandomColor()
	}

	var err error
	name, canonical, err = NameOrCanonical(name, canonical)
	if err != nil {
		diags.AddError("failed to infer canonical", err.Error())
		return data, diags
	}

	description := incoming.Description.ValueString()
	owner := incoming.Owner.ValueString()
	envType := envTypeFromData(incoming)

	var cloudAccountCanonicals []string
	if !incoming.CloudAccountCanonicals.IsNull() && !incoming.CloudAccountCanonicals.IsUnknown() {
		diags.Append(incoming.CloudAccountCanonicals.ElementsAs(ctx, &cloudAccountCanonicals, false)...)
		if diags.HasError() {
			return data, diags
		}
	}

	var apiVars []*models.EnvironmentVariableItem
	if !incoming.Variables.IsNull() && !incoming.Variables.IsUnknown() {
		var varModels []resource_environment.EnvironmentVariableModel
		diags.Append(incoming.Variables.ElementsAs(ctx, &varModels, false)...)
		if diags.HasError() {
			return data, diags
		}
		apiVars = make([]*models.EnvironmentVariableItem, len(varModels))
		for i, vm := range varModels {
			key := vm.Key.ValueString()
			typ := vm.Type.ValueString()
			apiVars[i] = &models.EnvironmentVariableItem{
				Key:         &key,
				Type:        &typ,
				Value:       stringToAny(vm.Value, typ),
				Description: vm.Description.ValueString(),
				Sensitive:   vm.Sensitive.ValueBoolPointer(),
			}
		}
	}

	m := p.provider.Middleware
	current, _, err := m.GetOrgEnv(org, canonical)
	if err == nil {
		updateBody := &models.UpdateEnvironment{
			Name:                   ptr.Ptr(name),
			Description:            description,
			Owner:                  owner,
			CloudAccountCanonicals: cloudAccountCanonicals,
			Variables:              apiVars,
		}
		if t := envTypeFromCurrent(current); t != "" {
			updateBody.Type = ptr.Ptr(t)
		}
		if !incoming.Type.IsNull() && !incoming.Type.IsUnknown() && envType != "" {
			updateBody.Type = ptr.Ptr(envType)
		}
		current, _, err = m.UpdateOrgEnv(org, canonical, updateBody)
		if err != nil {
			diags.AddError("failed to update environment from API", err.Error())
			return data, diags
		}
	} else {
		var apiErr *apiclient.APIResponseError
		if !stderrors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
			diags.AddError("failed to fetch environment from API while updating resource", err.Error())
			return data, diags
		}
		if isUpdate {
			tflog.Info(ctx, "did not find current environment, assuming it had been deleted outside the provider, re-creating...", nil)
		}

		createBody := &models.NewEnvironment{
			Canonical:              canonical,
			Name:                   ptr.Ptr(name),
			Description:            description,
			Owner:                  owner,
			CloudAccountCanonicals: cloudAccountCanonicals,
			Variables:              apiVars,
		}
		if envType != "" {
			createBody.Type = envType
		}
		current, _, err = m.CreateOrgEnv(org, createBody)
		if err != nil {
			diags.AddError("failed to create environment from API", err.Error())
			return data, diags
		}
	}

	if _, err = m.LinkEnvToProject(org, project, canonical); err != nil {
		diags.AddError("failed to link environment to project from API", err.Error())
		return data, diags
	}

	environment, _, err := m.GetOrgEnv(org, canonical)
	if err != nil {
		diags.AddError("failed to fetch environment from API after create/update", err.Error())
		return data, diags
	}

	_ = color // color is deprecated on environments; kept in schema for backward compatibility
	_ = current

	diags.Append(environmentToValue(ctx, org, project, environment, &data)...)

	// Preserve PATCH-semantic fields from the plan — these are Optional-only (not Computed).
	data.CloudAccountCanonicals = incoming.CloudAccountCanonicals
	data.Variables = incoming.Variables

	return data, diags
}
