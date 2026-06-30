package resource_oidc_group_mapping

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

func OidcGroupMappingResourceSchema(ctx context.Context) schema.Schema {
	desc := strings.Join([]string{
		"Maps an OIDC group claim to a team within an organization.",
		"The org-level role granted to OIDC-managed users is driven by the per-organization OIDC settings (`cycloid_oidc_organization_settings.default_role_canonical`), not by the mapping.",
		"Assign a group to several teams by declaring multiple mappings.",
	}, "\n")

	return schema.Schema{
		Description:         desc,
		MarkdownDescription: desc,
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "Organization canonical where to manage the mapping. Defaults to provider `default_organization`.",
				MarkdownDescription: "Organization canonical where to manage the mapping. Defaults to provider `default_organization`.",
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
			"group_name": schema.StringAttribute{
				Description:         "The OIDC group claim value to match.",
				MarkdownDescription: "The OIDC group claim value to match.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"team_canonical": schema.StringAttribute{
				Description:         "The canonical of the team the user is added to when this mapping matches.",
				MarkdownDescription: "The canonical of the team the user is added to when this mapping matches.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9]+[a-z0-9\-_]+[a-z0-9]+$`), ""),
				},
			},
			"id": schema.Int64Attribute{
				Description:         "OIDC group mapping identifier.",
				MarkdownDescription: "OIDC group mapping identifier.",
				Computed:            true,
			},
		},
	}
}

type OidcGroupMappingModel struct {
	Organization  types.String `tfsdk:"organization"`
	GroupName     types.String `tfsdk:"group_name"`
	TeamCanonical types.String `tfsdk:"team_canonical"`
	ID            types.Int64  `tfsdk:"id"`
}
