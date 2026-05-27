package datasource_environment_type

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func EnvironmentTypeDataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Look up a single Cycloid environment type by canonical.",
		MarkdownDescription: "Look up a single Cycloid environment type by canonical.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical to look up the environment type in. Defaults to the provider's `default_organization`.",
				MarkdownDescription: "The organization canonical to look up the environment type in. Defaults to the provider's `default_organization`.",
				Optional:            true,
				Computed:            true,
			},
			"canonical": schema.StringAttribute{
				Description:         "Canonical of the environment type to look up.",
				MarkdownDescription: "Canonical of the environment type to look up.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				Description:         "Display name of the environment type.",
				MarkdownDescription: "Display name of the environment type.",
				Computed:            true,
			},
			"color": schema.StringAttribute{
				Description:         "Color associated with this environment type.",
				MarkdownDescription: "Color associated with this environment type.",
				Computed:            true,
			},
			"is_default": schema.BoolAttribute{
				Description:         "True for built-in environment types that cannot be renamed or deleted.",
				MarkdownDescription: "True for built-in environment types that cannot be renamed or deleted.",
				Computed:            true,
			},
			"environments_count": schema.Int64Attribute{
				Description:         "Number of environments currently using this type.",
				MarkdownDescription: "Number of environments currently using this type.",
				Computed:            true,
			},
			"id": schema.Int64Attribute{
				Description:         "Internal numeric ID assigned by the Cycloid API.",
				MarkdownDescription: "Internal numeric ID assigned by the Cycloid API.",
				Computed:            true,
			},
		},
	}
}

type EnvironmentTypeModel struct {
	Organization      types.String `tfsdk:"organization"`
	Canonical         types.String `tfsdk:"canonical"`
	Name              types.String `tfsdk:"name"`
	Color             types.String `tfsdk:"color"`
	IsDefault         types.Bool   `tfsdk:"is_default"`
	EnvironmentsCount types.Int64  `tfsdk:"environments_count"`
	ID                types.Int64  `tfsdk:"id"`
}
