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
