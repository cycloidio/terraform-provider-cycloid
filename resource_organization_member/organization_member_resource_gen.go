// Code generated by terraform-plugin-framework-generator DO NOT EDIT.

package resource_organization_member

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func OrganizationMemberResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"email": schema.StringAttribute{
				Required:            true,
				Description:         "Invite user by email",
				MarkdownDescription: "Invite user by email",
			},
			"member_id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "A member id",
				MarkdownDescription: "A member id",
			},
			"organization_canonical": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "A canonical of an organization.",
				MarkdownDescription: "A canonical of an organization.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9]+[a-z0-9\\-_]+[a-z0-9]+$"), ""),
				},
			},
			"role_canonical": schema.StringAttribute{
				Required:            true,
				Description:         "The canonical of an entity",
				MarkdownDescription: "The canonical of an entity",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9]+[a-z0-9\\-_]+[a-z0-9]+$"), ""),
				},
			},
		},
	}
}

type OrganizationMemberModel struct {
	Email                 types.String `tfsdk:"email"`
	MemberId              types.Int64  `tfsdk:"member_id"`
	OrganizationCanonical types.String `tfsdk:"organization_canonical"`
	RoleCanonical         types.String `tfsdk:"role_canonical"`
}
