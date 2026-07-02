package provider

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"

	"github.com/cycloidio/cycloid-cli/client/models"
	middleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_organization_environment"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// defaultOrgEnvType is applied when the practitioner does not set `type`.
// The Cycloid API requires a non-empty environment type on create/update.
const defaultOrgEnvType = "production"

var _ resource.Resource = (*organizationEnvironmentResource)(nil)
var _ resource.ResourceWithImportState = (*organizationEnvironmentResource)(nil)

func NewOrganizationEnvironmentResource() resource.Resource {
	return &organizationEnvironmentResource{}
}

type organizationEnvironmentResourceModel resource_organization_environment.OrganizationEnvironmentModel

type organizationEnvironmentResource struct {
	provider *CycloidProvider
}

func (p *organizationEnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_environment"
}

func (p *organizationEnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_organization_environment.OrganizationEnvironmentResourceSchema(ctx)
}

func (p *organizationEnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (p *organizationEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationEnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := p.provider.Middleware
	org := getOrganizationCanonical(*p.provider, data.Organization)
	canonical := data.Canonical.ValueString()

	environment, _, err := m.GetOrgEnv(org, canonical)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("failed to fetch organization environment from API while reading state", err.Error())
		return
	}

	resp.Diagnostics.Append(orgEnvironmentToValue(ctx, org, environment, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (p *organizationEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationEnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, d := p.createOrUpdateOrgEnvironment(ctx, data, false)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (p *organizationEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data organizationEnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, d := p.createOrUpdateOrgEnvironment(ctx, data, true)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (p *organizationEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationEnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := p.provider.Middleware
	org := getOrganizationCanonical(*p.provider, data.Organization)
	canonical := data.Canonical.ValueString()

	// Real delete: removes the org-scoped environment entity itself, not just a
	// project link. This is the core distinction from cycloid_environment.
	_, err := m.DeleteOrgEnv(org, canonical)
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError("failed to delete organization environment from API", err.Error())
		return
	}
}

func (p *organizationEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("canonical"), req, resp)
}

func orgEnvironmentToValue(ctx context.Context, org string, environment *models.Environment, data *organizationEnvironmentResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if environment == nil {
		diags.AddError("environment is nil for conversion", "This should not happen, please contact the plugin maintainer.")
		return diags
	}

	data.Canonical = types.StringPointerValue(environment.Canonical)
	data.Name = types.StringValue(environment.Name)
	data.Organization = types.StringValue(org)
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
	// They are NOT refreshed from the API in Read — state is preserved from prior plan/state.
	// They ARE set from the plan after Create/Update in createOrUpdateOrgEnvironment.

	return diags
}

func orgEnvTypeFromData(data organizationEnvironmentResourceModel) string {
	if !data.Type.IsNull() && !data.Type.IsUnknown() && data.Type.ValueString() != "" {
		return data.Type.ValueString()
	}
	return ""
}

func (p *organizationEnvironmentResource) createOrUpdateOrgEnvironment(ctx context.Context, incoming organizationEnvironmentResourceModel, isUpdate bool) (organizationEnvironmentResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var data organizationEnvironmentResourceModel

	org := getOrganizationCanonical(*p.provider, incoming.Organization)
	name := incoming.Name.ValueString()
	canonical := incoming.Canonical.ValueString()

	var err error
	name, canonical, err = NameOrCanonical(name, canonical)
	if err != nil {
		diags.AddError("failed to infer canonical", err.Error())
		return data, diags
	}

	description := incoming.Description.ValueString()
	owner := incoming.Owner.ValueString()
	envType := orgEnvTypeFromData(incoming)

	var cloudAccountCanonicals []string
	if !incoming.CloudAccountCanonicals.IsNull() && !incoming.CloudAccountCanonicals.IsUnknown() {
		diags.Append(incoming.CloudAccountCanonicals.ElementsAs(ctx, &cloudAccountCanonicals, false)...)
		if diags.HasError() {
			return data, diags
		}
	}

	var apiVars []*models.EnvironmentVariableItem
	if !incoming.Variables.IsNull() && !incoming.Variables.IsUnknown() {
		var varModels []resource_organization_environment.OrganizationEnvironmentVariableModel
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
		// Preserve the existing type unless the practitioner sets one explicitly.
		updateBody.Type = ptr.Ptr(orgEnvTypeForUpdate(current, envType))
		current, _, err = m.UpdateOrgEnv(org, canonical, updateBody)
		if err != nil {
			diags.AddError("failed to update organization environment from API", err.Error())
			return data, diags
		}
	} else {
		var apiErr *middleware.APIResponseError
		if !stderrors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
			diags.AddError("failed to fetch organization environment from API while updating resource", err.Error())
			return data, diags
		}
		if isUpdate {
			tflog.Info(ctx, "did not find current organization environment, assuming it had been deleted outside the provider, re-creating...", nil)
		}

		createType := envType
		if createType == "" {
			createType = defaultOrgEnvType
		}
		createBody := &models.NewEnvironment{
			Canonical:              canonical,
			Name:                   ptr.Ptr(name),
			Description:            description,
			Owner:                  owner,
			Type:                   createType,
			CloudAccountCanonicals: cloudAccountCanonicals,
			Variables:              apiVars,
		}
		current, _, err = m.CreateOrgEnv(org, createBody)
		if err != nil {
			diags.AddError("failed to create organization environment from API", err.Error())
			return data, diags
		}
	}

	_ = current

	environment, _, err := m.GetOrgEnv(org, canonical)
	if err != nil {
		diags.AddError("failed to fetch organization environment from API after create/update", err.Error())
		return data, diags
	}

	diags.Append(orgEnvironmentToValue(ctx, org, environment, &data)...)

	// Preserve PATCH-semantic fields from the plan — these are Optional-only (not Computed).
	data.CloudAccountCanonicals = incoming.CloudAccountCanonicals
	data.Variables = incoming.Variables

	return data, diags
}

// orgEnvTypeForUpdate resolves the environment type to send on update: the
// explicitly-configured type takes precedence, otherwise the type currently
// stored on the environment, otherwise the default. The API requires it
// non-empty on every update.
func orgEnvTypeForUpdate(current *models.Environment, configured string) string {
	if configured != "" {
		return configured
	}
	if current != nil && current.EnvironmentType != nil && current.EnvironmentType.Canonical != nil && *current.EnvironmentType.Canonical != "" {
		return *current.EnvironmentType.Canonical
	}
	return defaultOrgEnvType
}
