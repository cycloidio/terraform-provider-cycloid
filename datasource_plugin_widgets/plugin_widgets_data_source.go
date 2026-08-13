package datasource_plugin_widgets

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func PluginWidgetsDataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Query organization-level plugin widgets by placement.",
		MarkdownDescription: "Query organization-level plugin widgets by placement.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical, defaults to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical, defaults to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
			},
			"placement": schema.StringAttribute{
				Description:         `The widget placement type to filter by. Valid values: "component", "sideMenuPage".`,
				MarkdownDescription: "The widget placement type to filter by. Valid values: `component`, `sideMenuPage`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("component", "sideMenuPage"),
				},
			},
			"widgets_json": schema.StringAttribute{
				Description:         "JSON-encoded array of widget objects.",
				MarkdownDescription: "JSON-encoded array of widget objects.",
				Computed:            true,
			},
		},
	}
}

type PluginWidgetsModel struct {
	Organization types.String `tfsdk:"organization"`
	Placement    types.String `tfsdk:"placement"`
	WidgetsJSON  types.String `tfsdk:"widgets_json"`
}
