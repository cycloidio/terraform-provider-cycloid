package resource_plugin_registry_plugin

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func PluginRegistryPluginResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Manage a plugin definition within a plugin registry.",
		MarkdownDescription: "Manage a plugin definition within a plugin registry.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical, defaults to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical, defaults to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"registry_id": schema.Int64Attribute{
				Description:         "The ID of the plugin registry this plugin belongs to.",
				MarkdownDescription: "The ID of the plugin registry this plugin belongs to.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description:         "The display name of the plugin.",
				MarkdownDescription: "The display name of the plugin.",
				Required:            true,
			},
			"id": schema.Int64Attribute{
				Description:         "The numeric ID of the plugin.",
				MarkdownDescription: "The numeric ID of the plugin.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"owned": schema.BoolAttribute{
				Description:         "Whether this plugin is owned by the organization.",
				MarkdownDescription: "Whether this plugin is owned by the organization.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
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
