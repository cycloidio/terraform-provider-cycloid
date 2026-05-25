package resource_environment_link

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func EnvironmentLinkResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Attach an existing organization-scoped environment to an additional project. Use this resource for the second, third, ... project link of a shared environment; the first link is owned by the `cycloid_environment` resource via its `project` attribute. Deleting this resource only unlinks the environment from the project, the environment itself is preserved.",
		MarkdownDescription: "Attach an existing organization-scoped environment to an additional project. Use this resource for the second, third, ... project link of a shared environment; the first link is owned by the [`cycloid_environment`](./environment.md) resource via its `project` attribute. Deleting this resource only unlinks the environment from the project, the environment itself is preserved.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical that owns both the project and the environment. Defaults to the provider's `default_organization`.",
				MarkdownDescription: "The organization canonical that owns both the project and the environment. Defaults to the provider's `default_organization`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"project": schema.StringAttribute{
				Description:         "Canonical of the project to link the environment to. Changing this attribute forces replacement.",
				MarkdownDescription: "Canonical of the project to link the environment to. Changing this attribute forces replacement.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
				},
			},
			"environment": schema.StringAttribute{
				Description:         "Canonical of the organization-scoped environment to link. Changing this attribute forces replacement.",
				MarkdownDescription: "Canonical of the organization-scoped environment to link. Changing this attribute forces replacement.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
				},
			},
			"id": schema.StringAttribute{
				Description:         "Composite identifier `<organization>/<project>/<environment>` used internally by Terraform.",
				MarkdownDescription: "Composite identifier `<organization>/<project>/<environment>` used internally by Terraform.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

type EnvironmentLinkModel struct {
	Organization types.String `tfsdk:"organization"`
	Project      types.String `tfsdk:"project"`
	Environment  types.String `tfsdk:"environment"`
	ID           types.String `tfsdk:"id"`
}
