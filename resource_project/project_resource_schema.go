package resource_project

import (
	"context"
	"regexp"

	"github.com/cycloidio/terraform-provider-cycloid/internal/icons"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func ProjectResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Display name of the project, for the UI, either name or canonical must be filled to create a project",
				MarkdownDescription: "Display name of the project, for the UI, either name or canonical must be filled to create a project",
			},
			"canonical": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Canonical of the project, serve as the unique identifier, either name or canonical must be filled to create a project",
				MarkdownDescription: "Canonical of the project, serve as the unique identifier, either name or canonical must be filled to create a project",
			},
			"organization": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The organization where to create the project, default to the `default_organization` of the provider",
				MarkdownDescription: "The organization where to create the project, default to the `default_organization` of the provider",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9]+[a-z0-9\-_]+[a-z0-9]+$`), ""),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Description:         "Description of the project, displayed in the UI",
				MarkdownDescription: "Description of the project, displayed in the UI",
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
			"icon": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The icon of the project displayed in the UI.",
				MarkdownDescription: "The icon of the project displayed in the UI.",
				Validators: []validator.String{
					stringvalidator.OneOf(icons.ValidIcons...),
				},
			},
			"owner": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Attribute a team or a member as owner of this project, affect teams by canonical and members by username. Will default to the owner of the current API Key.",
				MarkdownDescription: "Attribute a team or a member as owner of this project, affect teams by canonical and members by username. Will default to the owner of the current API Key.",
			},
			"config_repository": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Affect a config repository by its canonical to this project, default to the default config repository of the org.",
				MarkdownDescription: "Affect a config repository by its canonical to this project, default to the default config repository of the org.",
			},
		},
	}
}

type ProjectModel struct {
	Canonical        types.String `tfsdk:"canonical"`
	Color            types.String `tfsdk:"color"`
	ConfigRepository types.String `tfsdk:"config_repository"`
	Description      types.String `tfsdk:"description"`
	Icon             types.String `tfsdk:"icon"`
	Name             types.String `tfsdk:"name"`
	Organization     types.String `tfsdk:"organization"`
	Owner            types.String `tfsdk:"owner"`
}
