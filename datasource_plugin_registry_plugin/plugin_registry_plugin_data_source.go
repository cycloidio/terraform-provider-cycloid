package datasource_plugin_registry_plugin

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func PluginRegistryPluginDataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Look up a plugin definition within a registry by name.",
		MarkdownDescription: "Look up a plugin definition within a registry by name.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical, defaults to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical, defaults to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
			},
			"registry_id": schema.Int64Attribute{
				Description:         "The ID of the plugin registry to search within.",
				MarkdownDescription: "The ID of the plugin registry to search within.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The display name of the plugin to look up.",
				MarkdownDescription: "The display name of the plugin to look up.",
				Required:            true,
			},
			"id": schema.Int64Attribute{
				Description:         "The numeric ID of the plugin.",
				MarkdownDescription: "The numeric ID of the plugin.",
				Computed:            true,
			},
			"owned": schema.BoolAttribute{
				Description:         "Whether this plugin is owned by the organization.",
				MarkdownDescription: "Whether this plugin is owned by the organization.",
				Computed:            true,
			},
		},
	}
}

type PluginRegistryPluginModel struct {
	Organization types.String `tfsdk:"organization"`
	RegistryID   types.Int64  `tfsdk:"registry_id"`
	Name         types.String `tfsdk:"name"`
	ID           types.Int64  `tfsdk:"id"`
	Owned        types.Bool   `tfsdk:"owned"`
}
