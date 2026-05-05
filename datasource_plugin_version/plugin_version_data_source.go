package datasource_plugin_version

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func PluginVersionDataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Look up a plugin version by name or artifact URL within a registry plugin.",
		MarkdownDescription: "Look up a plugin version by name or artifact URL within a registry plugin.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical, defaults to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical, defaults to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
			},
			"registry_id": schema.Int64Attribute{
				Description:         "The ID of the plugin registry.",
				MarkdownDescription: "The ID of the plugin registry.",
				Required:            true,
			},
			"plugin_id": schema.Int64Attribute{
				Description:         "The ID of the registry plugin.",
				MarkdownDescription: "The ID of the registry plugin.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The version name to look up. Mutually exclusive with `url`.",
				MarkdownDescription: "The version name to look up. Mutually exclusive with `url`.",
				Optional:            true,
				Computed:            true,
			},
			"url": schema.StringAttribute{
				Description:         "The artifact URL to look up. Mutually exclusive with `name`.",
				MarkdownDescription: "The artifact URL to look up. Mutually exclusive with `name`.",
				Optional:            true,
				Computed:            true,
			},
			"id": schema.Int64Attribute{
				Description:         "The numeric ID of the plugin version.",
				MarkdownDescription: "The numeric ID of the plugin version.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				Description:         "Processing status: `pending`, `processing`, `success`, or `failed`.",
				MarkdownDescription: "Processing status: `pending`, `processing`, `success`, or `failed`.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				Description:         "Description of the plugin version.",
				MarkdownDescription: "Description of the plugin version.",
				Computed:            true,
			},
		},
	}
}

type PluginVersionModel struct {
	Organization types.String `tfsdk:"organization"`
	RegistryID   types.Int64  `tfsdk:"registry_id"`
	PluginID     types.Int64  `tfsdk:"plugin_id"`
	Name         types.String `tfsdk:"name"`
	URL          types.String `tfsdk:"url"`
	ID           types.Int64  `tfsdk:"id"`
	Status       types.String `tfsdk:"status"`
	Description  types.String `tfsdk:"description"`
}
