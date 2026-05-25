package datasource_environments

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func EnvironmentsDataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "List Cycloid environments. Without a `project` filter, returns the organization-scoped catalog. With a `project` filter, returns only environments linked to that project.",
		MarkdownDescription: "List Cycloid environments. Without a `project` filter, returns the organization-scoped catalog. With a `project` filter, returns only environments linked to that project.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical to list environments from. Defaults to the provider's `default_organization`.",
				MarkdownDescription: "The organization canonical to list environments from. Defaults to the provider's `default_organization`.",
				Optional:            true,
				Computed:            true,
			},
			"project": schema.StringAttribute{
				Description:         "Optional project canonical to filter the listing. When set, only environments linked to that project are returned.",
				MarkdownDescription: "Optional project canonical to filter the listing. When set, only environments linked to that project are returned.",
				Optional:            true,
			},
			"environments": schema.ListNestedAttribute{
				Description:         "Matching environments.",
				MarkdownDescription: "Matching environments.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"canonical":       schema.StringAttribute{Computed: true},
						"name":            schema.StringAttribute{Computed: true},
						"description":     schema.StringAttribute{Computed: true},
						"type":            schema.StringAttribute{Computed: true},
						"owner":           schema.StringAttribute{Computed: true},
						"resources_count": schema.Int64Attribute{Computed: true},
						"id":              schema.Int64Attribute{Computed: true},
						"created_at":      schema.Int64Attribute{Computed: true},
						"updated_at":      schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

type EnvironmentsModel struct {
	Organization types.String `tfsdk:"organization"`
	Project      types.String `tfsdk:"project"`
	Environments types.List   `tfsdk:"environments"`
}
