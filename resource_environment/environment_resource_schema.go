package resource_environment

import (
	"context"
	"regexp"

	"github.com/cycloidio/terraform-provider-cycloid/internal/icons"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func EnvironmentResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Version:             1,
		Description:         "This resource manage Cycloid environments. Environments are part of a project, see `cycloid_project` resource. Docs: https://docs.cycloid.io/reference/core-concepts/",
		MarkdownDescription: "This resource manage Cycloid environments. Environments are part of a project, see `cycloid_project` resource. [Docs](https://docs.cycloid.io/reference/core-concepts/).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				Description:         "The ID of the environment",
				MarkdownDescription: "The ID of the environment",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Display name of the environment, for the UI, either name or canonical must be filled to create a environment",
				MarkdownDescription: "Display name of the environment, for the UI, either name or canonical must be filled to create a environment",
			},
			"canonical": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Canonical of the environment, serve as the unique identifier, either name or canonical must be filled to create a environment",
				MarkdownDescription: "Canonical of the environment, serve as the unique identifier, either name or canonical must be filled to create a environment",
			},
			"project": schema.StringAttribute{
				Required:            true,
				Description:         "The project canonical where this environent resides",
				MarkdownDescription: "The project canonical where this environent resides",
			},
			"organization": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The organization where to create the environment, default to the `default_organization` of the provider",
				MarkdownDescription: "The organization where to create the environment, default to the `default_organization` of the provider",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9]+[a-z0-9\-_]+[a-z0-9]+$`), ""),
				},
			},
			"color": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The color for the icon displayed in the UI.",
				MarkdownDescription: "The color for the icon displayed in the UI.",
				Validators: []validator.String{
					stringvalidator.OneOf(icons.ValidColors...),
				},
			},
		},
	}
}

type EnvironmentModel struct {
	Project      types.String `tfsdk:"project"`
	Canonical    types.String `tfsdk:"canonical"`
	Color        types.String `tfsdk:"color"`
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Organization types.String `tfsdk:"organization"`
}
