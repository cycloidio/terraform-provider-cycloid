package resource_plugin_manager

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func PluginManagerResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manage a plugin manager in an organization. " +
			"Creating this resource registers the plugin manager as an invite and immediately accepts it.",
		MarkdownDescription: "Manage a plugin manager in an organization. " +
			"Creating this resource registers the plugin manager as an invite and immediately accepts it.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical, defaults to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical, defaults to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description:         "The display name of the plugin manager.",
				MarkdownDescription: "The display name of the plugin manager.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"url": schema.StringAttribute{
				Description:         "The URL of the plugin manager instance.",
				MarkdownDescription: "The URL of the plugin manager instance.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"id": schema.Int64Attribute{
				Description:         "The numeric ID of the plugin manager.",
				MarkdownDescription: "The numeric ID of the plugin manager.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				Description:         "Connection status of the plugin manager: `offline` or `connected`.",
				MarkdownDescription: "Connection status of the plugin manager: `offline` or `connected`.",
				Computed:            true,
			},
			"invite_status": schema.StringAttribute{
				Description:         "Invite status: always `accepted` after resource creation.",
				MarkdownDescription: "Invite status: always `accepted` after resource creation.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.Int64Attribute{
				Description:         "Unix timestamp of plugin manager creation.",
				MarkdownDescription: "Unix timestamp of plugin manager creation.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
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
	URL          types.String `tfsdk:"url"`
	ID           types.Int64  `tfsdk:"id"`
	Status       types.String `tfsdk:"status"`
	InviteStatus types.String `tfsdk:"invite_status"`
	CreatedAt    types.Int64  `tfsdk:"created_at"`
	UpdatedAt    types.Int64  `tfsdk:"updated_at"`
}
