package resource_stack

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func StackResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "The stack resource exists only to manage `visibility` and `team` parameters on a stack",
		MarkdownDescription: `
			The stack resource exists only to manage 'visibility' and 'team' parameters on a stack.

			On creation/update this will change those settings on the remote stack.

			On delete it will erase this resource on the state an keep the stack current state.
		`,
		Attributes: map[string]schema.Attribute{
			"organization_canonical": schema.StringAttribute{
				Required:            true,
				Description:         "The organization of the stack.",
				MarkdownDescription: "The organization of the stack.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9]+[a-z0-9\-_]+[a-z0-9]+$`), ""),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"canonical": schema.StringAttribute{
				Required:            true,
				Description:         "The canonical of a stack",
				MarkdownDescription: "The canonical of a stack",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9]+[a-z0-9\-_]+[a-z0-9]+$`), ""),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"visibility": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Change the visibility of a stack",
				MarkdownDescription: "Change the visibility of a stack",
				Validators: []validator.String{
					stringvalidator.OneOf("local", "shared", "hidden"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"team": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "Assign a team as maintainer of a stack",
				MarkdownDescription: "Assign a team as maintainer of a stack",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

type StackModel struct {
	OrganizationCanonical types.String `tfsdk:"organization_canonical"`
	Canonical             types.String `tfsdk:"canonical"`
	Visibility            types.String `tfsdk:"visibility"`
	Team                  types.String `tfsdk:"team"`
}
