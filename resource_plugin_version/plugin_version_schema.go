package resource_plugin_version

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func PluginVersionResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Publish a version for a plugin in a registry. " +
			"The URL typically references a Docker image (e.g. `docker.io/org/plugin:1.0.0`). " +
			"Terraform waits for processing to complete; the resource is tainted if processing fails.",
		MarkdownDescription: "Publish a version for a plugin in a registry. " +
			"The URL typically references a Docker image (e.g. `docker.io/org/plugin:1.0.0`). " +
			"Terraform waits for processing to complete; the resource is tainted if processing fails.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical, defaults to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical, defaults to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"registry_id": schema.Int64Attribute{
				Description:         "The ID of the plugin registry.",
				MarkdownDescription: "The ID of the plugin registry.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"plugin_id": schema.Int64Attribute{
				Description:         "The ID of the registry plugin this version belongs to.",
				MarkdownDescription: "The ID of the registry plugin this version belongs to.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"url": schema.StringAttribute{
				Description:         "The artifact URL for this plugin version (e.g. a Docker image reference).",
				MarkdownDescription: "The artifact URL for this plugin version (e.g. a Docker image reference).",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"id": schema.Int64Attribute{
				Description:         "The numeric ID of the plugin version.",
				MarkdownDescription: "The numeric ID of the plugin version.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description:         "The version name assigned by the registry.",
				MarkdownDescription: "The version name assigned by the registry.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
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
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

type PluginVersionModel struct {
	Organization types.String `tfsdk:"organization"`
	RegistryID   types.Int64  `tfsdk:"registry_id"`
	PluginID     types.Int64  `tfsdk:"plugin_id"`
	URL          types.String `tfsdk:"url"`
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Status       types.String `tfsdk:"status"`
	Description  types.String `tfsdk:"description"`
}
