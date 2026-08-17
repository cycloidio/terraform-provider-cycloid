package datasource_plugin_widget_views

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func PluginWidgetViewsDataSourceSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Description:         "List all widget views for an installed plugin.",
		MarkdownDescription: "List all widget views for an installed plugin.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical, defaults to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical, defaults to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
			},
			"plugin_install_id": schema.Int64Attribute{
				Description:         "The ID of the plugin install to list widget views for.",
				MarkdownDescription: "The ID of the plugin install to list widget views for.",
				Required:            true,
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"widget_views": schema.ListAttribute{
				Description:         "List of widget views for the plugin install.",
				MarkdownDescription: "List of widget views for the plugin install.",
				Computed:            true,
				ElementType: types.ObjectType{
					AttrTypes: WidgetViewAttrTypes(),
				},
			},
		},
	}
}

func WidgetViewAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                types.Int64Type,
		"enabled":           types.BoolType,
		"effective_enabled": types.BoolType,
		"url_slug":          types.StringType,
		"effective_slug":    types.StringType,
		"is_inherited":      types.BoolType,
		"has_override":      types.BoolType,
	}
}

type PluginWidgetViewsModel struct {
	Organization    types.String `tfsdk:"organization"`
	PluginInstallID types.Int64  `tfsdk:"plugin_install_id"`
	WidgetViews     types.List   `tfsdk:"widget_views"`
}

type WidgetViewItem struct {
	ID               types.Int64  `tfsdk:"id"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	EffectiveEnabled types.Bool   `tfsdk:"effective_enabled"`
	URLSlug          types.String `tfsdk:"url_slug"`
	EffectiveSlug    types.String `tfsdk:"effective_slug"`
	IsInherited      types.Bool   `tfsdk:"is_inherited"`
	HasOverride      types.Bool   `tfsdk:"has_override"`
}
