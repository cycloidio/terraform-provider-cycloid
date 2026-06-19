package resource_oidc_integration

import (
	"context"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func OidcIntegrationResourceSchema(ctx context.Context) schema.Schema {
	desc := strings.Join([]string{
		"Manages an organization's AuthenticationOIDC SSO integration.",
		"Singleton resource: one OIDC integration config exists per organization.",
		"The integration is created/updated via a PUT (merge semantics); absent keys keep their stored values.",
		"There is no delete API endpoint — removing this resource disables the integration (`enabled = false`) and drops it from Terraform state.",
	}, "\n")

	return schema.Schema{
		Description:         desc,
		MarkdownDescription: desc,
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "Organization canonical where to manage the OIDC integration. Defaults to provider `default_organization`.",
				MarkdownDescription: "Organization canonical where to manage the OIDC integration. Defaults to provider `default_organization`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9]+[a-z0-9\-_]+[a-z0-9]+$`), ""),
				},
			},

			// --- required ---
			"enabled": schema.BoolAttribute{
				Description:         "Whether the OIDC SSO integration is active for the organization.",
				MarkdownDescription: "Whether the OIDC SSO integration is active for the organization.",
				Required:            true,
			},

			// --- optional non-secret ---
			"display_name": schema.StringAttribute{
				Description:         "Human-readable label shown on the login page for this OIDC provider.",
				MarkdownDescription: "Human-readable label shown on the login page for this OIDC provider.",
				Optional:            true,
			},
			"client_id": schema.StringAttribute{
				Description:         "OIDC client ID registered with the identity provider.",
				MarkdownDescription: "OIDC client ID registered with the identity provider.",
				Optional:            true,
			},
			"issuer": schema.StringAttribute{
				Description:         "OIDC issuer URL of the identity provider.",
				MarkdownDescription: "OIDC issuer URL of the identity provider.",
				Optional:            true,
			},
			"icon": schema.StringAttribute{
				Description:         "URL or name of the icon displayed on the login button.",
				MarkdownDescription: "URL or name of the icon displayed on the login button.",
				Optional:            true,
			},
			"groups_claim_name": schema.StringAttribute{
				Description:         "Name of the claim in the OIDC token that carries the user's group memberships.",
				MarkdownDescription: "Name of the claim in the OIDC token that carries the user's group memberships.",
				Optional:            true,
			},
			"discovery_url": schema.StringAttribute{
				Description:         "Override URL for the OIDC discovery document (`.well-known/openid-configuration`). When set, `issuer` may be omitted.",
				MarkdownDescription: "Override URL for the OIDC discovery document (`.well-known/openid-configuration`). When set, `issuer` may be omitted.",
				Optional:            true,
			},
			"session_ttl_seconds": schema.Int64Attribute{
				Description:         "Session duration in seconds for OIDC-authenticated users. Leave unset to use the provider default.",
				MarkdownDescription: "Session duration in seconds for OIDC-authenticated users. Leave unset to use the provider default.",
				Optional:            true,
			},

			// --- optional bool flags ---
			"client_secret_jwt": schema.BoolAttribute{
				Description:         "When true, the client authenticates to the token endpoint using a JWT signed with the client secret (private_key_jwt).",
				MarkdownDescription: "When true, the client authenticates to the token endpoint using a JWT signed with the client secret (private_key_jwt).",
				Optional:            true,
				Computed:            true,
			},
			"use_ca_cert": schema.BoolAttribute{
				Description:         "When true, the custom CA certificate (see `ca_cert`) is used to verify the identity provider's TLS certificate.",
				MarkdownDescription: "When true, the custom CA certificate (see `ca_cert`) is used to verify the identity provider's TLS certificate.",
				Optional:            true,
				Computed:            true,
			},
			"skip_tls_verify": schema.BoolAttribute{
				Description:         "When true, TLS certificate verification is skipped for the identity provider endpoints. Use only in non-production environments.",
				MarkdownDescription: "When true, TLS certificate verification is skipped for the identity provider endpoints. Use only in non-production environments.",
				Optional:            true,
				Computed:            true,
			},
			"allow_insecure_discovery": schema.BoolAttribute{
				Description:         "When true, the OIDC discovery document may be fetched over HTTP (insecure). Use only in non-production environments.",
				MarkdownDescription: "When true, the OIDC discovery document may be fetched over HTTP (insecure). Use only in non-production environments.",
				Optional:            true,
				Computed:            true,
			},
			"adopt_manual_members": schema.BoolAttribute{
				Description:         "When true, existing manually-invited members who log in via this OIDC integration are adopted — their membership source is flipped to 'oidc' so group mapping manages them going forward.",
				MarkdownDescription: "When true, existing manually-invited members who log in via this OIDC integration are adopted — their membership source is flipped to `oidc` so group mapping manages them going forward.",
				Optional:            true,
				Computed:            true,
			},

			// --- write-only secrets ---
			"client_secret": schema.StringAttribute{
				Description:         "OIDC client secret. Write-only: the API never returns this value. Change the value here to rotate the secret. The `has_secret` attribute reflects whether a secret is currently stored on the server.",
				MarkdownDescription: "OIDC client secret. Write-only: the API never returns this value. Change the value here to rotate the secret. The `has_secret` attribute reflects whether a secret is currently stored on the server.",
				Optional:            true,
				Sensitive:           true,
			},
			"ca_cert": schema.StringAttribute{
				Description:         "PEM-encoded CA certificate used to verify the identity provider's TLS certificate. Write-only: the API never returns this value. Change the value here to rotate the certificate. The `has_ca_certificate` attribute reflects whether a certificate is currently stored on the server.",
				MarkdownDescription: "PEM-encoded CA certificate used to verify the identity provider's TLS certificate. Write-only: the API never returns this value. Change the value here to rotate the certificate. The `has_ca_certificate` attribute reflects whether a certificate is currently stored on the server.",
				Optional:            true,
				Sensitive:           true,
			},

			// --- computed presence flags ---
			"has_secret": schema.BoolAttribute{
				Description:         "True when the server has a stored client secret. Reflects server state; not settable.",
				MarkdownDescription: "True when the server has a stored client secret. Reflects server state; not settable.",
				Computed:            true,
			},
			"has_ca_certificate": schema.BoolAttribute{
				Description:         "True when the server has a stored CA certificate. Reflects server state; not settable.",
				MarkdownDescription: "True when the server has a stored CA certificate. Reflects server state; not settable.",
				Computed:            true,
			},
		},
	}
}

// OidcIntegrationModel is the Terraform state model for the oidc_integration resource.
type OidcIntegrationModel struct {
	Organization           types.String `tfsdk:"organization"`
	Enabled                types.Bool   `tfsdk:"enabled"`
	DisplayName            types.String `tfsdk:"display_name"`
	ClientID               types.String `tfsdk:"client_id"`
	Issuer                 types.String `tfsdk:"issuer"`
	Icon                   types.String `tfsdk:"icon"`
	GroupsClaimName        types.String `tfsdk:"groups_claim_name"`
	DiscoveryURL           types.String `tfsdk:"discovery_url"`
	SessionTTLSeconds      types.Int64  `tfsdk:"session_ttl_seconds"`
	ClientSecretJwt        types.Bool   `tfsdk:"client_secret_jwt"`
	UseCaCert              types.Bool   `tfsdk:"use_ca_cert"`
	SkipTLSVerify          types.Bool   `tfsdk:"skip_tls_verify"`
	AllowInsecureDiscovery types.Bool   `tfsdk:"allow_insecure_discovery"`
	AdoptManualMembers     types.Bool   `tfsdk:"adopt_manual_members"`
	ClientSecret           types.String `tfsdk:"client_secret"`
	CaCert                 types.String `tfsdk:"ca_cert"`
	HasSecret              types.Bool   `tfsdk:"has_secret"`
	HasCaCertificate       types.Bool   `tfsdk:"has_ca_certificate"`
}
