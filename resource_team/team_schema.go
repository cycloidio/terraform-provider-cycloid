package resource_team

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TeamResourcesSchema(ctx context.Context) schema.Schema {
	teamDesc := strings.Join([]string{
		"Manage team in an organization.",
		"Teams allows you to groupe people and assign permissions to it.",
		"",
		"A team can get multiple roles assigned for custom permissions.",
	}, "\n")
	return schema.Schema{
		Description:         teamDesc,
		MarkdownDescription: teamDesc,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description:         "The name of the team, displayed in the UI. Either `name` or `canonical` must be filled.",
				MarkdownDescription: "The name of the team, displayed in the UI. Either `name` or `canonical` must be filled.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(
						path.MatchRoot("name"),
						path.MatchRoot("canonical"),
					),
				},
			},
			"canonical": schema.StringAttribute{
				Description:         "The canonical of the team will be inferred from the name if not specified.",
				MarkdownDescription: "The canonical of the team will be inferred from the name if not specified.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(
						path.MatchRoot("name"),
						path.MatchRoot("canonical"),
					),
				},
			},
			"organization": schema.StringAttribute{
				Description:         "The organization canonical where to create the team, default to the provider `default_organization`",
				MarkdownDescription: "The organization canonical where to create the team, default to the provider `default_organization`",
				Optional:            true,
				Computed:            true,
			},
			"owner": schema.StringAttribute{
				Description:         "The username of the team's owner, will default to the owner of the current API key at creation.",
				MarkdownDescription: "The username of the team's owner, will default to the owner of the current API key at creation.",
				Optional:            true,
				Computed:            true,
			},
			"roles": schema.ListAttribute{
				Description:         "List of roles canonicals to attribute to this team, you can list available roles using the Cycloid CLI following command: `cy list roles`",
				MarkdownDescription: "List of roles canonicals to attribute to this team, you can list available roles using the Cycloid CLI following command: `cy list roles`",
				ElementType:         types.StringType,
				Required:            true,
			},
		},
	}
}

var TeamTypes = map[string]attr.Type{
	"name":         types.StringType,
	"canonical":    types.StringType,
	"organization": types.StringType,
	"owner":        types.StringType,
	"roles": types.ListType{
		ElemType: types.StringType,
	},
}

type TeamModel struct {
	Name         types.String `tfsdk:"name"`
	Canonical    types.String `tfsdk:"canonical"`
	Organization types.String `tfsdk:"organization"`
	Owner        types.String `tfsdk:"owner"`
	Roles        types.List   `tfsdk:"roles"`
}
