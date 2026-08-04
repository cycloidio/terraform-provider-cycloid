package resource_plugin_widget_view

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func PluginWidgetViewResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Manage widget view overrides for an installed plugin.",
		MarkdownDescription: "Manage widget view overrides for an installed plugin.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical, defaults to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical, defaults to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"widget_view_id": schema.Int64Attribute{
				Description:         "The ID of the widget view to manage.",
				MarkdownDescription: "The ID of the widget view to manage.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"plugin_install_id": schema.Int64Attribute{
				Description:         "The ID of the plugin install that owns the widget view.",
				MarkdownDescription: "The ID of the plugin install that owns the widget view.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{
				Description:         "Whether the widget view is enabled.",
				MarkdownDescription: "Whether the widget view is enabled.",
				Required:            true,
			},
			"url_slug": schema.StringAttribute{
				Description:         "The URL slug for the widget view.",
				MarkdownDescription: "The URL slug for the widget view.",
				Required:            true,
			},
			"effective_enabled": schema.BoolAttribute{
				Description:         "The effective enabled state after inheritance resolution.",
				MarkdownDescription: "The effective enabled state after inheritance resolution.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"effective_slug": schema.StringAttribute{
				Description:         "The effective URL slug after inheritance resolution.",
				MarkdownDescription: "The effective URL slug after inheritance resolution.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"is_inherited": schema.BoolAttribute{
				Description:         "Whether the widget view configuration is inherited.",
				MarkdownDescription: "Whether the widget view configuration is inherited.",
				Computed:            true,
			},
			"has_override": schema.BoolAttribute{
				Description:         "Whether the widget view has an override.",
				MarkdownDescription: "Whether the widget view has an override.",
				Computed:            true,
			},
		},
	}
}

type PluginWidgetViewModel struct {
	Organization     types.String `tfsdk:"organization"`
	WidgetViewID     types.Int64  `tfsdk:"widget_view_id"`
	PluginInstallID  types.Int64  `tfsdk:"plugin_install_id"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	URLSlug          types.String `tfsdk:"url_slug"`
	EffectiveEnabled types.Bool   `tfsdk:"effective_enabled"`
	EffectiveSlug    types.String `tfsdk:"effective_slug"`
	IsInherited      types.Bool   `tfsdk:"is_inherited"`
	HasOverride      types.Bool   `tfsdk:"has_override"`
}
