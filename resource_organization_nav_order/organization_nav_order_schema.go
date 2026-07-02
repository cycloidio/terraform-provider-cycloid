package resource_organization_nav_order

import (
	"context"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func OrganizationNavOrderResourceSchema(ctx context.Context) schema.Schema {
	desc := strings.Join([]string{
		"Manage the per-organization sidebar navigation ordering.",
		"Singleton resource: a single nav ordering config exists per organization.",
		"`items` is required, but an empty list (`items = []`) resets the sidebar to its default ordering.",
	}, "\n")

	return schema.Schema{
		Description:         desc,
		MarkdownDescription: desc,
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "Organization canonical where to manage the nav ordering. Defaults to provider `default_organization`.",
				MarkdownDescription: "Organization canonical where to manage the nav ordering. Defaults to provider `default_organization`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9]+[a-z0-9\-_]+[a-z0-9]+$`), ""),
				},
			},
			"items": schema.ListNestedAttribute{
				Description:         "Sidebar entries in the desired order. Required — pass an empty list to reset to the default ordering.",
				MarkdownDescription: "Sidebar entries in the desired order. Required — pass an empty list to reset to the default ordering.",
				Required:            true,
				Validators: []validator.List{
					listvalidator.SizeAtMost(200),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description:         "Either `native` (a built-in section, identified by name) or `plugin_widget` (identified by the widget's ID as a string).",
							MarkdownDescription: "Either `native` (a built-in section, identified by name) or `plugin_widget` (identified by the widget's ID as a string).",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("native", "plugin_widget"),
							},
						},
						"key": schema.StringAttribute{
							Description:         "Native section name (e.g. `dashboard`) or plugin widget ID (as a string).",
							MarkdownDescription: "Native section name (e.g. `dashboard`) or plugin widget ID (as a string).",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 255),
							},
						},
						"position": schema.Int64Attribute{
							Description:         "1-indexed position. Must be unique across all items.",
							MarkdownDescription: "1-indexed position. Must be unique across all items.",
							Required:            true,
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
							},
						},
					},
				},
			},
		},
	}
}

type NavItemModel struct {
	Type     types.String `tfsdk:"type"`
	Key      types.String `tfsdk:"key"`
	Position types.Int64  `tfsdk:"position"`
}

type OrganizationNavOrderModel struct {
	Organization types.String `tfsdk:"organization"`
	Items        types.List   `tfsdk:"items"`
}
