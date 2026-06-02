package resource_oidc_organization_settings

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

func OidcOrganizationSettingsResourceSchema(ctx context.Context) schema.Schema {
	desc := strings.Join([]string{
		"Manage the per-organization OIDC reconciliation settings.",
		"Singleton resource: a single settings row exists per organization.",
		"`oidc_no_match_policy = \"eject\"` requires `oidc_managed = true`; the API rejects the combination otherwise.",
	}, "\n")

	return schema.Schema{
		Description:         desc,
		MarkdownDescription: desc,
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "Organization canonical where to manage the OIDC settings. Defaults to provider `default_organization`.",
				MarkdownDescription: "Organization canonical where to manage the OIDC settings. Defaults to provider `default_organization`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9]+[a-z0-9\\-_]+[a-z0-9]+$"), ""),
				},
			},
			"default_role_canonical": schema.StringAttribute{
				Description:         "Canonical of the org-level role granted to OIDC-managed users on provisioning. Leave unset to configure no default role.",
				MarkdownDescription: "Canonical of the org-level role granted to OIDC-managed users on provisioning. Leave unset to configure no default role.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9]+[a-z0-9\\-_]+[a-z0-9]+$"), ""),
				},
			},
			"oidc_managed": schema.BoolAttribute{
				Description:         "When true, local member/team/invite edits are disabled for the organization (strict mode).",
				MarkdownDescription: "When true, local member/team/invite edits are disabled for the organization (strict mode).",
				Required:            true,
			},
			"oidc_no_match_policy": schema.StringAttribute{
				Description:         "Policy applied when no group mapping matches on login. `eject` requires `oidc_managed = true`.",
				MarkdownDescription: "Policy applied when no group mapping matches on login. `eject` requires `oidc_managed = true`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("keep_membership", "eject"),
				},
			},
		},
	}
}

type OidcOrganizationSettingsModel struct {
	Organization         types.String `tfsdk:"organization"`
	DefaultRoleCanonical types.String `tfsdk:"default_role_canonical"`
	OidcManaged          types.Bool   `tfsdk:"oidc_managed"`
	OidcNoMatchPolicy    types.String `tfsdk:"oidc_no_match_policy"`
}
