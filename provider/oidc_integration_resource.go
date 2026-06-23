package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apiclient "github.com/cycloidio/cycloid-cli/cmd/apiclient"
	"github.com/cycloidio/terraform-provider-cycloid/resource_oidc_integration"
)

var (
	_ resource.Resource                = (*oidcIntegrationResource)(nil)
	_ resource.ResourceWithImportState = (*oidcIntegrationResource)(nil)
)

// NewOIDCIntegrationResource is the constructor registered in provider.go.
func NewOIDCIntegrationResource() resource.Resource {
	return &oidcIntegrationResource{}
}

type oidcIntegrationResource struct {
	provider *CycloidProvider
}

type oidcIntegrationResourceModel resource_oidc_integration.OidcIntegrationModel

func (r *oidcIntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_integration"
}

func (r *oidcIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_oidc_integration.OidcIntegrationResourceSchema(ctx)
}

func (r *oidcIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *oidcIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data oidcIntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	integration, _, err := r.provider.Middleware.UpdateOIDCIntegration(org, oidcIntegrationConfig(&data))
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to create OIDC integration in org %q", org), err.Error())
		return
	}

	oidcIntegrationToData(org, integration, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *oidcIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data oidcIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)

	integration, _, err := r.provider.Middleware.GetOIDCIntegration(org)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("failed to read OIDC integration in org %q", org), err.Error())
		return
	}

	// Map non-secret fields + presence flags. client_secret and ca_cert are
	// intentionally NOT overwritten: the API never returns them, so we preserve
	// whatever value is already in state to avoid perpetual drift.
	oidcIntegrationToData(org, integration, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *oidcIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data oidcIntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	integration, _, err := r.provider.Middleware.UpdateOIDCIntegration(org, oidcIntegrationConfig(&data))
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to update OIDC integration in org %q", org), err.Error())
		return
	}

	oidcIntegrationToData(org, integration, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete has no API counterpart. We disable the integration (PUT with
// enabled=false) and then remove it from Terraform state. Server-side secrets
// and config are preserved unchanged.
func (r *oidcIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data oidcIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)

	_, _, err := r.provider.Middleware.UpdateOIDCIntegration(org, map[string]interface{}{
		"type":    "AuthenticationOIDC",
		"enabled": false,
	})
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("failed to disable OIDC integration in org %q", org), err.Error())
	}
}

// ImportState supports: terraform import cycloid_oidc_integration.x <organization>
func (r *oidcIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data oidcIntegrationResourceModel
	data.Organization = types.StringValue(req.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// oidcIntegrationConfig builds the map[string]interface{} body for
// UpdateOIDCIntegration. "type" and "enabled" are always sent (required by the
// backend). All other non-secret fields are included when non-null/non-unknown.
// Secrets are only included when the plan carries a non-empty string, so that
// an absent or empty value leaves the stored secret/cert untouched on the
// server (merge semantics).
func oidcIntegrationConfig(data *oidcIntegrationResourceModel) map[string]interface{} {
	cfg := map[string]interface{}{
		"type":    "AuthenticationOIDC",
		"enabled": data.Enabled.ValueBool(),
	}

	if !data.DisplayName.IsNull() && !data.DisplayName.IsUnknown() {
		cfg["oidc_display_name"] = data.DisplayName.ValueString()
	}
	if !data.ClientID.IsNull() && !data.ClientID.IsUnknown() {
		cfg["oidc_client_id"] = data.ClientID.ValueString()
	}
	if !data.Issuer.IsNull() && !data.Issuer.IsUnknown() {
		cfg["oidc_issuer"] = data.Issuer.ValueString()
	}
	if !data.Icon.IsNull() && !data.Icon.IsUnknown() {
		cfg["oidc_icon"] = data.Icon.ValueString()
	}
	if !data.GroupsClaimName.IsNull() && !data.GroupsClaimName.IsUnknown() {
		cfg["oidc_groups_claim_name"] = data.GroupsClaimName.ValueString()
	}
	if !data.DiscoveryURL.IsNull() && !data.DiscoveryURL.IsUnknown() {
		cfg["oidc_discovery_url"] = data.DiscoveryURL.ValueString()
	}
	if !data.SessionTTLSeconds.IsNull() && !data.SessionTTLSeconds.IsUnknown() {
		cfg["oidc_session_ttl_seconds"] = data.SessionTTLSeconds.ValueInt64()
	}
	if !data.ClientSecretJwt.IsNull() && !data.ClientSecretJwt.IsUnknown() {
		cfg["oidc_client_secret_jwt"] = data.ClientSecretJwt.ValueBool()
	}
	if !data.UseCaCert.IsNull() && !data.UseCaCert.IsUnknown() {
		cfg["oidc_use_ca_cert"] = data.UseCaCert.ValueBool()
	}
	if !data.SkipTLSVerify.IsNull() && !data.SkipTLSVerify.IsUnknown() {
		cfg["oidc_skip_tls_verify"] = data.SkipTLSVerify.ValueBool()
	}
	if !data.AllowInsecureDiscovery.IsNull() && !data.AllowInsecureDiscovery.IsUnknown() {
		cfg["oidc_allow_insecure_discovery"] = data.AllowInsecureDiscovery.ValueBool()
	}
	if !data.AdoptManualMembers.IsNull() && !data.AdoptManualMembers.IsUnknown() {
		cfg["oidc_adopt_manual_members"] = data.AdoptManualMembers.ValueBool()
	}

	// Secrets: only send when the plan carries a non-empty string. An absent or
	// empty value preserves the stored secret/cert on the server.
	if v := data.ClientSecret.ValueString(); v != "" {
		cfg["oidc_client_secret"] = v
	}
	if v := data.CaCert.ValueString(); v != "" {
		cfg["oidc_ca_cert"] = v
	}

	return cfg
}

// oidcIntegrationToData maps the API response to the Terraform state model.
//
// It sets the organization, all non-secret fields, and the computed presence
// flags (has_secret, has_ca_certificate).
//
// It deliberately does NOT touch data.ClientSecret or data.CaCert. The API
// never returns those values; overwriting them with an empty string would cause
// perpetual drift on every subsequent plan. Callers must preserve the prior
// state values for those two fields.
func oidcIntegrationToData(org string, i *apiclient.OIDCIntegration, data *oidcIntegrationResourceModel) {
	data.Organization = types.StringValue(org)
	data.Enabled = types.BoolValue(i.Enabled)

	if i.OidcDisplayName != "" {
		data.DisplayName = types.StringValue(i.OidcDisplayName)
	} else {
		data.DisplayName = types.StringNull()
	}
	if i.OidcClientID != "" {
		data.ClientID = types.StringValue(i.OidcClientID)
	} else {
		data.ClientID = types.StringNull()
	}
	if i.OidcIssuer != "" {
		data.Issuer = types.StringValue(i.OidcIssuer)
	} else {
		data.Issuer = types.StringNull()
	}
	if i.OidcIcon != "" {
		data.Icon = types.StringValue(i.OidcIcon)
	} else {
		data.Icon = types.StringNull()
	}
	if i.OidcGroupsClaimName != "" {
		data.GroupsClaimName = types.StringValue(i.OidcGroupsClaimName)
	} else {
		data.GroupsClaimName = types.StringNull()
	}

	if i.OidcDiscoveryURL != nil {
		data.DiscoveryURL = types.StringValue(*i.OidcDiscoveryURL)
	} else {
		data.DiscoveryURL = types.StringNull()
	}

	if i.OidcSessionTTLSeconds != nil {
		data.SessionTTLSeconds = types.Int64Value(*i.OidcSessionTTLSeconds)
	} else {
		data.SessionTTLSeconds = types.Int64Null()
	}

	data.ClientSecretJwt = types.BoolValue(i.OidcClientSecretJwt)
	data.UseCaCert = types.BoolValue(i.OidcUseCaCert)
	data.SkipTLSVerify = types.BoolValue(i.OidcSkipTLSVerify)

	if i.OidcAllowInsecureDiscovery != nil {
		data.AllowInsecureDiscovery = types.BoolValue(*i.OidcAllowInsecureDiscovery)
	} else {
		data.AllowInsecureDiscovery = types.BoolNull()
	}
	if i.OidcAdoptManualMembers != nil {
		data.AdoptManualMembers = types.BoolValue(*i.OidcAdoptManualMembers)
	} else {
		data.AdoptManualMembers = types.BoolNull()
	}

	// Presence flags — server reports whether a secret/cert is stored.
	if i.HasSecret != nil {
		data.HasSecret = types.BoolValue(*i.HasSecret)
	} else {
		data.HasSecret = types.BoolValue(false)
	}
	if i.HasCaCertificate != nil {
		data.HasCaCertificate = types.BoolValue(*i.HasCaCertificate)
	} else {
		data.HasCaCertificate = types.BoolValue(false)
	}

	// data.ClientSecret and data.CaCert are intentionally left untouched.
}
