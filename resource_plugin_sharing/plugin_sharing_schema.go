package resource_plugin_sharing

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func PluginSharingResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Manage sharing configuration for an installed plugin.",
		MarkdownDescription: "Manage sharing configuration for an installed plugin.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical, defaults to the provider `default_organization`.",
				MarkdownDescription: "The organization canonical, defaults to the provider `default_organization`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"plugin_install_id": schema.Int64Attribute{
				Description:         "The ID of the plugin install to configure sharing for.",
				MarkdownDescription: "The ID of the plugin install to configure sharing for.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"visibility": schema.StringAttribute{
				Description:         "Sharing visibility: `local` or `shared`.",
				MarkdownDescription: "Sharing visibility: `local` or `shared`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("local", "shared"),
				},
			},
			"mode": schema.StringAttribute{
				Description:         "Sharing mode: `include` or `exclude`. Defaults to `include`.",
				MarkdownDescription: "Sharing mode: `include` or `exclude`. Defaults to `include`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("include"),
				Validators: []validator.String{
					stringvalidator.OneOf("include", "exclude"),
				},
			},
			"organizations": schema.ListAttribute{
				Description:         "List of organization canonicals to include or exclude from sharing.",
				MarkdownDescription: "List of organization canonicals to include or exclude from sharing.",
				Optional:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

type PluginSharingModel struct {
	Organization    types.String `tfsdk:"organization"`
	PluginInstallID types.Int64  `tfsdk:"plugin_install_id"`
	Visibility      types.String `tfsdk:"visibility"`
	Mode            types.String `tfsdk:"mode"`
	Organizations   types.List   `tfsdk:"organizations"`
}
