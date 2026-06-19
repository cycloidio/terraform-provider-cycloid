package provider

import (
	"context"
	"fmt"

	cycloidmiddleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/resource_oidc_organization_settings"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*oidcOrganizationSettingsResource)(nil)
var _ resource.ResourceWithImportState = (*oidcOrganizationSettingsResource)(nil)

func NewOIDCOrganizationSettingsResource() resource.Resource {
	return &oidcOrganizationSettingsResource{}
}

type oidcOrganizationSettingsResource struct {
	provider *CycloidProvider
}

type oidcOrganizationSettingsResourceModel resource_oidc_organization_settings.OidcOrganizationSettingsModel

func (r *oidcOrganizationSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_organization_settings"
}

func (r *oidcOrganizationSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_oidc_organization_settings.OidcOrganizationSettingsResourceSchema(ctx)
}

func (r *oidcOrganizationSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *oidcOrganizationSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data oidcOrganizationSettingsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	settings, _, err := r.provider.Middleware.UpdateOIDCOrganizationSettings(org, oidcOrganizationSettingsBody(&data))
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to create OIDC settings in org %q", org), err.Error())
		return
	}

	oidcOrganizationSettingsToData(org, settings, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *oidcOrganizationSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data oidcOrganizationSettingsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)

	settings, _, err := r.provider.Middleware.GetOIDCOrganizationSettings(org)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("failed to read OIDC settings in org %q", org), err.Error())
		return
	}

	oidcOrganizationSettingsToData(org, settings, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *oidcOrganizationSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data oidcOrganizationSettingsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	settings, _, err := r.provider.Middleware.UpdateOIDCOrganizationSettings(org, oidcOrganizationSettingsBody(&data))
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to update OIDC settings in org %q", org), err.Error())
		return
	}

	oidcOrganizationSettingsToData(org, settings, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *oidcOrganizationSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data oidcOrganizationSettingsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)

	// The API has no delete endpoint. Reset to safe defaults so that
	// oidc_managed=true + eject is not left active after terraform destroy.
	_, _, err := r.provider.Middleware.UpdateOIDCOrganizationSettings(org, cycloidmiddleware.UpdateOIDCOrganizationSettings{
		OIDCManaged:       false,
		OIDCNoMatchPolicy: "keep_membership",
	})
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddWarning(
			"Unable to reset OIDC organization settings",
			"The resource was removed from Terraform state, but the server-side settings could not be reset to safe defaults. Error: "+err.Error(),
		)
	}
}

// ImportState supports: terraform import cycloid_oidc_organization_settings.x <organization>
func (r *oidcOrganizationSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data oidcOrganizationSettingsResourceModel
	data.Organization = types.StringValue(req.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func oidcOrganizationSettingsBody(data *oidcOrganizationSettingsResourceModel) cycloidmiddleware.UpdateOIDCOrganizationSettings {
	return cycloidmiddleware.UpdateOIDCOrganizationSettings{
		DefaultRoleCanonical: data.DefaultRoleCanonical.ValueString(),
		OIDCManaged:          data.OidcManaged.ValueBool(),
		OIDCNoMatchPolicy:    data.OidcNoMatchPolicy.ValueString(),
	}
}

func oidcOrganizationSettingsToData(org string, settings *cycloidmiddleware.OIDCOrganizationSettings, data *oidcOrganizationSettingsResourceModel) {
	data.Organization = types.StringValue(org)
	if settings.DefaultRoleCanonical == "" {
		data.DefaultRoleCanonical = types.StringNull()
	} else {
		data.DefaultRoleCanonical = types.StringValue(settings.DefaultRoleCanonical)
	}
	data.OidcManaged = types.BoolValue(settings.OIDCManaged)
	data.OidcNoMatchPolicy = types.StringValue(settings.OIDCNoMatchPolicy)
}
