package resource_plugin_version

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"plugin_id": schema.Int64Attribute{
				Description:         "The ID of the registry plugin this version belongs to.",
				MarkdownDescription: "The ID of the registry plugin this version belongs to.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
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
			"icon": schema.StringAttribute{
				Description:         "Icon URL of the plugin version.",
				MarkdownDescription: "Icon URL of the plugin version.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"images": schema.ListAttribute{
				Description:         "Image URLs for the plugin version.",
				MarkdownDescription: "Image URLs for the plugin version.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"scope": schema.ListAttribute{
				Description:         "Policies in which the plugin version is scoped.",
				MarkdownDescription: "Policies in which the plugin version is scoped.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"configuration_schema": schema.StringAttribute{
				Description:         "Stack Forms configuration schema as JSON.",
				MarkdownDescription: "Stack Forms configuration schema as JSON.",
				Computed:            true,
			},
			"schema": schema.StringAttribute{
				Description:         "SQLite schema definition as JSON.",
				MarkdownDescription: "SQLite schema definition as JSON.",
				Computed:            true,
			},
			"widgets": schema.StringAttribute{
				Description:         "Default widget configuration as JSON.",
				MarkdownDescription: "Default widget configuration as JSON.",
				Computed:            true,
			},
			"error": schema.StringAttribute{
				Description:         "Error message if version processing failed.",
				MarkdownDescription: "Error message if version processing failed.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true}),
		},
	}
}

type PluginVersionModel struct {
	Organization        types.String   `tfsdk:"organization"`
	RegistryID          types.Int64    `tfsdk:"registry_id"`
	PluginID            types.Int64    `tfsdk:"plugin_id"`
	URL                 types.String   `tfsdk:"url"`
	ID                  types.Int64    `tfsdk:"id"`
	Name                types.String   `tfsdk:"name"`
	Status              types.String   `tfsdk:"status"`
	Description         types.String   `tfsdk:"description"`
	Icon                types.String   `tfsdk:"icon"`
	Images              types.List     `tfsdk:"images"`
	Scope               types.List     `tfsdk:"scope"`
	ConfigurationSchema types.String   `tfsdk:"configuration_schema"`
	Schema              types.String   `tfsdk:"schema"`
	Widgets             types.String   `tfsdk:"widgets"`
	Error               types.String   `tfsdk:"error"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}
