package datasource_plugin_registry

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func PluginRegistryDataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Look up a plugin registry by name or URL within an organization.",
		MarkdownDescription: "Look up a plugin registry by name or URL within an organization.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical, defaults to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical, defaults to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The display name of the plugin registry to look up. Mutually exclusive with `url`.",
				MarkdownDescription: "The display name of the plugin registry to look up. Mutually exclusive with `url`.",
				Optional:            true,
			},
			"url": schema.StringAttribute{
				Description:         "The URL of the plugin registry to look up. Mutually exclusive with `name`.",
				MarkdownDescription: "The URL of the plugin registry to look up. Mutually exclusive with `name`.",
				Optional:            true,
				Computed:            true,
			},
			"id": schema.Int64Attribute{
				Description:         "The numeric ID of the plugin registry.",
				MarkdownDescription: "The numeric ID of the plugin registry.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				Description:         "Connection status of the registry: `offline` or `connected`.",
				MarkdownDescription: "Connection status of the registry: `offline` or `connected`.",
				Computed:            true,
			},
			"access": schema.BoolAttribute{
				Description:         "Whether you have access to create plugins in this registry.",
				MarkdownDescription: "Whether you have access to create plugins in this registry.",
				Computed:            true,
			},
			"created_at": schema.Int64Attribute{
				Description:         "Unix timestamp of registry creation.",
				MarkdownDescription: "Unix timestamp of registry creation.",
				Computed:            true,
			},
			"updated_at": schema.Int64Attribute{
				Description:         "Unix timestamp of last registry update.",
				MarkdownDescription: "Unix timestamp of last registry update.",
				Computed:            true,
			},
		},
	}
}

type PluginRegistryModel struct {
	Organization types.String `tfsdk:"organization"`
	Name         types.String `tfsdk:"name"`
	URL          types.String `tfsdk:"url"`
	ID           types.Int64  `tfsdk:"id"`
	Status       types.String `tfsdk:"status"`
	Access       types.Bool   `tfsdk:"access"`
	CreatedAt    types.Int64  `tfsdk:"created_at"`
	UpdatedAt    types.Int64  `tfsdk:"updated_at"`
}
