package resource_team_member

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

func TeamMemberResourcesSchema(ctx context.Context) schema.Schema {
	teamMemberDesc := strings.Join([]string{
		"Assign an `orgnization_member` (user) to a team.",
	}, "\n")
	return schema.Schema{
		Description:         teamMemberDesc,
		MarkdownDescription: teamMemberDesc,
		Attributes: map[string]schema.Attribute{
			"team": schema.StringAttribute{
				Description:         "The canonical of the team.",
				MarkdownDescription: "The canonical of the team.",
				Required:            true,
			},
			"organization": schema.StringAttribute{
				Description:         "The organization canonical of the team, default to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical of the team, default to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
			},
			"username": schema.StringAttribute{
				Description:         "The username of the member to invite.",
				MarkdownDescription: "The username of the member to invite.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(
						path.MatchRoot("username"),
						path.MatchRoot("email"),
					),
				},
			},
			"email": schema.StringAttribute{
				Description:         "The email of the member to invite.",
				MarkdownDescription: "The email of the member to invite.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(
						path.MatchRoot("username"),
						path.MatchRoot("email"),
					),
				},
			},
		},
	}
}

var TeamMemberTypes = map[string]attr.Type{
	"team":         types.StringType,
	"organization": types.StringType,
	"username":     types.StringType,
	"email":        types.StringType,
}

type TeamMemberModel struct {
	Team         types.String `tfsdk:"team"`
	Organization types.String `tfsdk:"organization"`
	Username     types.String `tfsdk:"username"`
	Email        types.String `tfsdk:"email"`
}
