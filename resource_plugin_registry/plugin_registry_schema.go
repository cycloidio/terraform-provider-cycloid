package resource_plugin_registry

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func PluginRegistryResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Manage a plugin registry in an organization.",
		MarkdownDescription: "Manage a plugin registry in an organization.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical, defaults to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical, defaults to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description:         "The display name of the plugin registry.",
				MarkdownDescription: "The display name of the plugin registry.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"url": schema.StringAttribute{
				Description:         "The URL of the plugin registry.",
				MarkdownDescription: "The URL of the plugin registry.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"id": schema.Int64Attribute{
				Description:         "The numeric ID of the plugin registry.",
				MarkdownDescription: "The numeric ID of the plugin registry.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
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
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.Int64Attribute{
				Description:         "Unix timestamp of registry creation.",
				MarkdownDescription: "Unix timestamp of registry creation.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
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
