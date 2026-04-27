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
			"plugin_version_id": schema.Int64Attribute{
				Description:         "The ID of the plugin version to install. Triggers replacement when changed.",
				MarkdownDescription: "The ID of the plugin version to install. Triggers replacement when changed.",
				Optional:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"configuration": schema.MapAttribute{
				Description:         "Key-value configuration for the plugin (Stack Forms syntax). Triggers replacement when changed.",
				MarkdownDescription: "Key-value configuration for the plugin (Stack Forms syntax). Triggers replacement when changed.",
				Required:            true,
				Sensitive:           true,
				ElementType:         types.StringType,
				PlanModifiers:       []planmodifier.Map{mapplanmodifier.RequiresReplace()},
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
				Description:         "Installation status: `pending`, `installed`, or `failed`.",
				MarkdownDescription: "Installation status: `pending`, `installed`, or `failed`.",
				Computed:            true,
			},
			"pm_secret": schema.StringAttribute{
				Description:         "The plugin manager secret for webhook generation.",
				MarkdownDescription: "The plugin manager secret for webhook generation.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
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
	Organization    types.String `tfsdk:"organization"`
	PluginVersionID types.Int64  `tfsdk:"plugin_version_id"`
	Configuration   types.Map    `tfsdk:"configuration"`
	ID              types.Int64  `tfsdk:"id"`
	UUID            types.String `tfsdk:"uuid"`
	Status          types.String `tfsdk:"status"`
	PmSecret        types.String `tfsdk:"pm_secret"`
	CreatedAt       types.Int64  `tfsdk:"created_at"`
	UpdatedAt       types.Int64  `tfsdk:"updated_at"`
}
