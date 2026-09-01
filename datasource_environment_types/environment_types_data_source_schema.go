package datasource_environment_types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func EnvironmentTypesDataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "List all Cycloid environment types in an organization.",
		MarkdownDescription: "List all Cycloid environment types in an organization.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical to list environment types from. Defaults to the provider's `default_organization`.",
				MarkdownDescription: "The organization canonical to list environment types from. Defaults to the provider's `default_organization`.",
				Optional:            true,
				Computed:            true,
			},
			"environment_types": schema.ListNestedAttribute{
				Description:         "Environment types in the organization.",
				MarkdownDescription: "Environment types in the organization.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"canonical":          schema.StringAttribute{Computed: true},
						"name":               schema.StringAttribute{Computed: true},
						"color":              schema.StringAttribute{Computed: true},
						"is_default":         schema.BoolAttribute{Computed: true},
						"environments_count": schema.Int64Attribute{Computed: true},
						"id":                 schema.Int64Attribute{Computed: true},
						"label_selector": schema.SingleNestedAttribute{
							Computed:    true,
							Description: "Label selector constraining which stacks are offered in environments of this type.",
							Attributes: map[string]schema.Attribute{
								"enforcement": schema.StringAttribute{Computed: true},
								"requirements": schema.ListNestedAttribute{
									Computed: true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"key":      schema.StringAttribute{Computed: true},
											"operator": schema.StringAttribute{Computed: true},
											"values":   schema.ListAttribute{Computed: true, ElementType: types.StringType},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

type EnvironmentTypesModel struct {
	Organization     types.String `tfsdk:"organization"`
	EnvironmentTypes types.List   `tfsdk:"environment_types"`
}
