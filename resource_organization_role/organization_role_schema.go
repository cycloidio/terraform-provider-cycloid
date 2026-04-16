package resource_organization_role

import (
	"context"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func OrganizationRoleResourceSchema(ctx context.Context) schema.Schema {
	organizationRoleDesc := strings.Join([]string{
		"Manage custom organization roles.",
		"Roles define a list of authorization rules that can be assigned to organization members and teams.",
	}, "\n")

	return schema.Schema{
		Description:         organizationRoleDesc,
		MarkdownDescription: organizationRoleDesc,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description:         "Display name of the role. Either `name` or `canonical` must be set.",
				MarkdownDescription: "Display name of the role. Either `name` or `canonical` must be set.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(
						path.MatchRoot("name"),
						path.MatchRoot("canonical"),
					),
					stringvalidator.LengthBetween(3, 100),
				},
			},
			"canonical": schema.StringAttribute{
				Description:         "Canonical unique identifier of the role. Inferred from `name` when omitted.",
				MarkdownDescription: "Canonical unique identifier of the role. Inferred from `name` when omitted.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(
						path.MatchRoot("name"),
						path.MatchRoot("canonical"),
					),
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9]+[a-z0-9\\-_]+[a-z0-9]+$"), ""),
				},
			},
			"organization": schema.StringAttribute{
				Description:         "Organization canonical where to manage the role. Defaults to provider `default_organization`.",
				MarkdownDescription: "Organization canonical where to manage the role. Defaults to provider `default_organization`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9]+[a-z0-9\\-_]+[a-z0-9]+$"), ""),
				},
			},
			"description": schema.StringAttribute{
				Description:         "Role description shown in Cycloid.",
				MarkdownDescription: "Role description shown in Cycloid.",
				Optional:            true,
				Computed:            true,
			},
			"rules": schema.SetNestedAttribute{
				Description:         "Authorization rules attached to this role.",
				MarkdownDescription: "Authorization rules attached to this role.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							Description:         "Policy action code, supports globs.",
							MarkdownDescription: "Policy action code, supports globs.",
							Required:            true,
						},
						"effect": schema.StringAttribute{
							Description:         "Rule effect. Only `allow` is supported.",
							MarkdownDescription: "Rule effect. Only `allow` is supported.",
							Optional:            true,
							Computed:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("allow"),
							},
						},
						"resources": schema.ListAttribute{
							Description:         "Resources where this action applies.",
							MarkdownDescription: "Resources where this action applies.",
							ElementType:         types.StringType,
							Optional:            true,
							Computed:            true,
						},
					},
				},
			},
			"id": schema.Int64Attribute{
				Description:         "Role identifier.",
				MarkdownDescription: "Role identifier.",
				Computed:            true,
			},
			"default": schema.BoolAttribute{
				Description:         "Whether this role is a Cycloid default role.",
				MarkdownDescription: "Whether this role is a Cycloid default role.",
				Computed:            true,
			},
		},
	}
}

var OrganizationRoleRuleTypes = map[string]attr.Type{
	"action":    types.StringType,
	"effect":    types.StringType,
	"resources": types.ListType{ElemType: types.StringType},
}

type OrganizationRoleModel struct {
	Name         types.String `tfsdk:"name"`
	Canonical    types.String `tfsdk:"canonical"`
	Organization types.String `tfsdk:"organization"`
	Description  types.String `tfsdk:"description"`
	Rules        types.Set    `tfsdk:"rules"`
	ID           types.Int64  `tfsdk:"id"`
	Default      types.Bool   `tfsdk:"default"`
}

type OrganizationRoleRuleModel struct {
	Action    types.String `tfsdk:"action"`
	Effect    types.String `tfsdk:"effect"`
	Resources types.List   `tfsdk:"resources"`
}
