package datasource_inventory_values

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func InventoryValuesDataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "This datasource allows you to fetch inventory values from the Cycloid inventory.",
		MarkdownDescription: "This datasource allows you to fetch inventory values from the Cycloid inventory.",
		Attributes: map[string]schema.Attribute{
			"filters": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"attribute": schema.StringAttribute{
							Required:            true,
							Description:         "The name of the attribute to filter.",
							MarkdownDescription: "The name of the attribute to filter.",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						"condition": schema.StringAttribute{
							Required:            true,
							Description:         `The condition to apply, one of "eq", "neq", "gt", "lt", "rlike" or "in".`,
							MarkdownDescription: `The condition to apply, one of "eq", "neq", "gt", "lt", "rlike" or "in".`,
							Validators: []validator.String{
								stringvalidator.OneOf("eq", "neq", "gt", "lt", "rlike", "in"),
							},
						},
						"value": schema.StringAttribute{
							Description:         "The value of the filter.",
							MarkdownDescription: "The value of the filter.",
							Required:            true,
						},
					},
				},
				Optional:            true,
				Description:         "List of LHS filters to apply to the inventory values. See the docs here: https://docs.cycloid.io/reference/api/LHS-filters",
				MarkdownDescription: "List of LHS filters to apply to the inventory values. See the docs [here](https://docs.cycloid.io/reference/api/LHS-filters)",
			},
			"organization": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The canonical of the organization where to fetch inventory values, default to the provider's organization.",
				MarkdownDescription: "The canonical of the organization where to fetch inventory values, default to the provider's organization.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
				},
			},
			"values": schema.DynamicAttribute{
				Computed:            true,
				Description:         "A list of inventory values matching the filters.",
				MarkdownDescription: "A list of inventory values matching the filters.",
			},
		},
	}
}

type Filter struct {
	Attribute string `tfsdk:"attribute"`
	Condition string `tfsdk:"condition"`
	Value     string `tfsdk:"value"`
}

type InventoryValuesModel struct {
	Filters      types.List    `tfsdk:"filters"`
	Organization types.String  `tfsdk:"organization"`
	Values       types.Dynamic `tfsdk:"values"`
}
