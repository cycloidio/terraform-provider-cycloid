package datasource_organization_plugin_widgets

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func OrganizationPluginWidgetsDataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "List organization-level plugin widgets for a placement, including the widget IDs used by organization nav ordering.",
		MarkdownDescription: "List organization-level plugin widgets for a placement, including the widget IDs used by `cycloid_organization_nav_order` items of type `plugin_widget`.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical, defaults to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical, defaults to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
			},
			"placement": schema.StringAttribute{
				Description:         "The plugin widget placement to list, such as `sideMenuPage` or `component`.",
				MarkdownDescription: "The plugin widget placement to list, such as `sideMenuPage` or `component`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"widgets": schema.ListNestedAttribute{
				Description:         "Plugin widgets matching the placement filter.",
				MarkdownDescription: "Plugin widgets matching the placement filter.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description:         "The numeric plugin widget ID. Use this value as the `key` for `plugin_widget` nav-order items.",
							MarkdownDescription: "The numeric plugin widget ID. Use this value as the `key` for `plugin_widget` nav-order items.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							Description:         "The widget type returned by the plugin manager.",
							MarkdownDescription: "The widget type returned by the plugin manager.",
							Computed:            true,
						},
						"placement": schema.StringAttribute{
							Description:         "The backend placement configuration for the widget, encoded as JSON.",
							MarkdownDescription: "The backend placement configuration for the widget, encoded as JSON.",
							Computed:            true,
						},
						"is_default": schema.BoolAttribute{
							Description:         "Whether this is a default widget.",
							MarkdownDescription: "Whether this is a default widget.",
							Computed:            true,
						},
						"widget": schema.StringAttribute{
							Description:         "The backend widget configuration, encoded as JSON.",
							MarkdownDescription: "The backend widget configuration, encoded as JSON.",
							Computed:            true,
						},
						"relation": schema.StringAttribute{
							Description:         "The backend relation information for the plugin widget, encoded as JSON.",
							MarkdownDescription: "The backend relation information for the plugin widget, encoded as JSON.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func PluginWidgetObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":         types.Int64Type,
			"type":       types.StringType,
			"placement":  types.StringType,
			"is_default": types.BoolType,
			"widget":     types.StringType,
			"relation":   types.StringType,
		},
	}
}

type PluginWidgetModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Type      types.String `tfsdk:"type"`
	Placement types.String `tfsdk:"placement"`
	IsDefault types.Bool   `tfsdk:"is_default"`
	Widget    types.String `tfsdk:"widget"`
	Relation  types.String `tfsdk:"relation"`
}

type OrganizationPluginWidgetsModel struct {
	Organization types.String `tfsdk:"organization"`
	Placement    types.String `tfsdk:"placement"`
	Widgets      types.List   `tfsdk:"widgets"`
}
