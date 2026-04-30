package datasource_plugin_manager

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func PluginManagerDataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Look up a plugin manager by name within an organization.",
		MarkdownDescription: "Look up a plugin manager by name within an organization.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical, defaults to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical, defaults to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The display name of the plugin manager to look up.",
				MarkdownDescription: "The display name of the plugin manager to look up.",
				Required:            true,
			},
			"id": schema.Int64Attribute{
				Description:         "The numeric ID of the plugin manager.",
				MarkdownDescription: "The numeric ID of the plugin manager.",
				Computed:            true,
			},
			"url": schema.StringAttribute{
				Description:         "The URL of the plugin manager instance.",
				MarkdownDescription: "The URL of the plugin manager instance.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				Description:         "Connection status: `offline` or `connected`.",
				MarkdownDescription: "Connection status: `offline` or `connected`.",
				Computed:            true,
			},
			"created_at": schema.Int64Attribute{
				Description:         "Unix timestamp of plugin manager creation.",
				MarkdownDescription: "Unix timestamp of plugin manager creation.",
				Computed:            true,
			},
			"updated_at": schema.Int64Attribute{
				Description:         "Unix timestamp of last plugin manager update.",
				MarkdownDescription: "Unix timestamp of last plugin manager update.",
				Computed:            true,
			},
		},
	}
}

type PluginManagerModel struct {
	Organization types.String `tfsdk:"organization"`
	Name         types.String `tfsdk:"name"`
	ID           types.Int64  `tfsdk:"id"`
	URL          types.String `tfsdk:"url"`
	Status       types.String `tfsdk:"status"`
	CreatedAt    types.Int64  `tfsdk:"created_at"`
	UpdatedAt    types.Int64  `tfsdk:"updated_at"`
}
