package resource_plugin

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func PluginResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Install a plugin in an organization. " +
			"All fields trigger a replacement on change because plugin upgrades are not supported via the API yet.",
		MarkdownDescription: "Install a plugin in an organization. " +
			"All fields trigger a replacement on change because plugin upgrades are not supported via the API yet.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical, defaults to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical, defaults to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"registry_id": schema.Int64Attribute{
				Description:         "The ID of the plugin registry containing the plugin to install.",
				MarkdownDescription: "The ID of the plugin registry containing the plugin to install.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"plugin_id": schema.Int64Attribute{
				Description:         "The ID of the plugin within the registry.",
				MarkdownDescription: "The ID of the plugin within the registry.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"plugin_version_id": schema.Int64Attribute{
				Description:         "The ID of the plugin version to install. Triggers replacement when changed.",
				MarkdownDescription: "The ID of the plugin version to install. Triggers replacement when changed.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"configuration": schema.MapAttribute{
				Description: "Visible key-value configuration for the plugin (Stack Forms syntax). " +
					"Values appear in plan output. Triggers replacement when changed.",
				MarkdownDescription: "Visible key-value configuration for the plugin (Stack Forms syntax). " +
					"Values appear in plan output. Triggers replacement when changed.",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()},
			},
			"configuration_sensitive": schema.MapAttribute{
				Description: "Sensitive key-value configuration for the plugin (Stack Forms syntax). " +
					"Values are hidden in plan output. Triggers replacement when changed. " +
					"Keys must not overlap with `configuration`.",
				MarkdownDescription: "Sensitive key-value configuration for the plugin (Stack Forms syntax). " +
					"Values are hidden in plan output. Triggers replacement when changed. " +
					"Keys must not overlap with `configuration`.",
				Optional:      true,
				Sensitive:     true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()},
			},
			"id": schema.Int64Attribute{
				Description:         "The numeric ID of the installed plugin.",
				MarkdownDescription: "The numeric ID of the installed plugin.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"uuid": schema.StringAttribute{
				Description:         "The UUID of the installed plugin.",
				MarkdownDescription: "The UUID of the installed plugin.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
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
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.Int64Attribute{
				Description:         "Unix timestamp of last install update.",
				MarkdownDescription: "Unix timestamp of last install update.",
				Computed:            true,
			},
		},
	}
}

type PluginModel struct {
	Organization           types.String `tfsdk:"organization"`
	RegistryID             types.Int64  `tfsdk:"registry_id"`
	PluginID               types.Int64  `tfsdk:"plugin_id"`
	PluginVersionID        types.Int64  `tfsdk:"plugin_version_id"`
	Configuration          types.Map    `tfsdk:"configuration"`
	ConfigurationSensitive types.Map    `tfsdk:"configuration_sensitive"`
	ID                     types.Int64  `tfsdk:"id"`
	UUID                   types.String `tfsdk:"uuid"`
	Status                 types.String `tfsdk:"status"`
	CreatedAt              types.Int64  `tfsdk:"created_at"`
	UpdatedAt              types.Int64  `tfsdk:"updated_at"`
}
