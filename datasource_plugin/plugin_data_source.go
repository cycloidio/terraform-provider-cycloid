package datasource_plugin

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func PluginDataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Look up an installed plugin by name within an organization.",
		MarkdownDescription: "Look up an installed plugin by name within an organization.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical, defaults to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical, defaults to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The display name of the installed plugin to look up.",
				MarkdownDescription: "The display name of the installed plugin to look up.",
				Required:            true,
			},
			"id": schema.Int64Attribute{
				Description:         "The numeric ID of the plugin install.",
				MarkdownDescription: "The numeric ID of the plugin install.",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				Description:         "The UUID of the plugin install.",
				MarkdownDescription: "The UUID of the plugin install.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				Description:         "Installation status: `pending`, `running`, or `failed`.",
				MarkdownDescription: "Installation status: `pending`, `running`, or `failed`.",
				Computed:            true,
			},
			"created_at": schema.Int64Attribute{
				Description:         "Unix timestamp of install creation.",
				MarkdownDescription: "Unix timestamp of install creation.",
				Computed:            true,
			},
			"updated_at": schema.Int64Attribute{
				Description:         "Unix timestamp of last install update.",
				MarkdownDescription: "Unix timestamp of last install update.",
				Computed:            true,
			},
			"pm_secret": schema.StringAttribute{
				Description:         "Webhook secret for the plugin install.",
				MarkdownDescription: "Webhook secret for the plugin install.",
				Computed:            true,
				Sensitive:           true,
			},
			"version_name": schema.StringAttribute{
				Description:         "Name of the installed plugin version.",
				MarkdownDescription: "Name of the installed plugin version.",
				Computed:            true,
			},
			"version_status": schema.StringAttribute{
				Description:         "Processing status of the installed plugin version.",
				MarkdownDescription: "Processing status of the installed plugin version.",
				Computed:            true,
			},
			"registry_id": schema.Int64Attribute{
				Description:         "The ID of the plugin registry.",
				MarkdownDescription: "The ID of the plugin registry.",
				Computed:            true,
			},
			"plugin_id": schema.Int64Attribute{
				Description:         "The ID of the plugin within the registry.",
				MarkdownDescription: "The ID of the plugin within the registry.",
				Computed:            true,
			},
			"plugin_version_id": schema.Int64Attribute{
				Description:         "The ID of the installed plugin version.",
				MarkdownDescription: "The ID of the installed plugin version.",
				Computed:            true,
			},
		},
	}
}

type PluginModel struct {
	Organization    types.String `tfsdk:"organization"`
	Name            types.String `tfsdk:"name"`
	ID              types.Int64  `tfsdk:"id"`
	UUID            types.String `tfsdk:"uuid"`
	Status          types.String `tfsdk:"status"`
	CreatedAt       types.Int64  `tfsdk:"created_at"`
	UpdatedAt       types.Int64  `tfsdk:"updated_at"`
	PmSecret        types.String `tfsdk:"pm_secret"`
	VersionName     types.String `tfsdk:"version_name"`
	VersionStatus   types.String `tfsdk:"version_status"`
	RegistryID      types.Int64  `tfsdk:"registry_id"`
	PluginID        types.Int64  `tfsdk:"plugin_id"`
	PluginVersionID types.Int64  `tfsdk:"plugin_version_id"`
}
