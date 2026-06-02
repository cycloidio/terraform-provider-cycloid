package provider

import (
	"context"
	"fmt"
	"net/http"

	cycloidmiddleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/resource_oidc_organization_settings"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*oidcOrganizationSettingsResource)(nil)

func NewOIDCOrganizationSettingsResource() resource.Resource {
	return &oidcOrganizationSettingsResource{}
}

type oidcOrganizationSettingsResource struct {
	provider *CycloidProvider
}

type oidcOrganizationSettingsResourceModel resource_oidc_organization_settings.OidcOrganizationSettingsModel

type oidcOrganizationSettings struct {
	DefaultRoleCanonical string `json:"default_role_canonical,omitempty"`
	OidcManaged          bool   `json:"oidc_managed"`
	OidcNoMatchPolicy    string `json:"oidc_no_match_policy"`
}

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
	settings, err := r.updateSettings(org, oidcOrganizationSettingsBody(&data))
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

	settings, err := r.getSettings(org)
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
	settings, err := r.updateSettings(org, oidcOrganizationSettingsBody(&data))
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to update OIDC settings in org %q", org), err.Error())
		return
	}

	oidcOrganizationSettingsToData(org, settings, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *oidcOrganizationSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// The Cycloid API exposes no delete for per-organization OIDC settings;
	// removing the resource only drops it from Terraform state.
}

func oidcOrganizationSettingsBody(data *oidcOrganizationSettingsResourceModel) *oidcOrganizationSettings {
	return &oidcOrganizationSettings{
		DefaultRoleCanonical: data.DefaultRoleCanonical.ValueString(),
		OidcManaged:          data.OidcManaged.ValueBool(),
		OidcNoMatchPolicy:    data.OidcNoMatchPolicy.ValueString(),
	}
}

func oidcOrganizationSettingsToData(org string, settings *oidcOrganizationSettings, data *oidcOrganizationSettingsResourceModel) {
	data.Organization = types.StringValue(org)
	if settings.DefaultRoleCanonical == "" {
		data.DefaultRoleCanonical = types.StringNull()
	} else {
		data.DefaultRoleCanonical = types.StringValue(settings.DefaultRoleCanonical)
	}
	data.OidcManaged = types.BoolValue(settings.OidcManaged)
	data.OidcNoMatchPolicy = types.StringValue(settings.OidcNoMatchPolicy)
}

func (r *oidcOrganizationSettingsResource) getSettings(org string) (*oidcOrganizationSettings, error) {
	result := &oidcOrganizationSettings{}
	_, err := r.provider.Middleware.GenericRequest(cycloidmiddleware.Request{
		Method:       http.MethodGet,
		Organization: &org,
		Route:        []string{"organizations", org, "oidc-organization-settings"},
	}, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *oidcOrganizationSettingsResource) updateSettings(org string, body *oidcOrganizationSettings) (*oidcOrganizationSettings, error) {
	result := &oidcOrganizationSettings{}
	_, err := r.provider.Middleware.GenericRequest(cycloidmiddleware.Request{
		Method:       http.MethodPut,
		Organization: &org,
		Route:        []string{"organizations", org, "oidc-organization-settings"},
		Body:         body,
	}, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
