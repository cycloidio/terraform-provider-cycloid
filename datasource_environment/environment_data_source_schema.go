package datasource_environment

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func EnvironmentDataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Look up a single Cycloid organization-scoped environment by canonical.",
		MarkdownDescription: "Look up a single Cycloid organization-scoped environment by canonical.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical to look up the environment in. Defaults to the provider's `default_organization`.",
				MarkdownDescription: "The organization canonical to look up the environment in. Defaults to the provider's `default_organization`.",
				Optional:            true,
				Computed:            true,
			},
			"canonical": schema.StringAttribute{
				Description:         "Canonical of the environment to look up.",
				MarkdownDescription: "Canonical of the environment to look up.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				Description:         "Display name of the environment.",
				MarkdownDescription: "Display name of the environment.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				Description:         "Free-form description of the environment.",
				MarkdownDescription: "Free-form description of the environment.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				Description:         "Canonical of the environment type associated with this environment.",
				MarkdownDescription: "Canonical of the environment type associated with this environment.",
				Computed:            true,
			},
			"owner": schema.StringAttribute{
				Description:         "Username of the environment owner.",
				MarkdownDescription: "Username of the environment owner.",
				Computed:            true,
			},
			"cloud_account_canonicals": schema.ListAttribute{
				Description:         "Canonicals of the cloud accounts linked to this environment.",
				MarkdownDescription: "Canonicals of the cloud accounts linked to this environment.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"variables": schema.ListNestedAttribute{
				Description:         "Environment variables attached to this environment.",
				MarkdownDescription: "Environment variables attached to this environment.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key":         schema.StringAttribute{Computed: true},
						"type":        schema.StringAttribute{Computed: true},
						"value":       schema.DynamicAttribute{Computed: true, Sensitive: true},
						"description": schema.StringAttribute{Computed: true},
						"sensitive":   schema.BoolAttribute{Computed: true},
					},
				},
			},
			"resources_count": schema.Int64Attribute{
				Description:         "Total count of infrastructure resources attributed to this environment.",
				MarkdownDescription: "Total count of infrastructure resources attributed to this environment.",
				Computed:            true,
			},
			"id": schema.Int64Attribute{
				Description:         "Internal numeric ID assigned by the Cycloid API.",
				MarkdownDescription: "Internal numeric ID assigned by the Cycloid API.",
				Computed:            true,
			},
			"created_at": schema.Int64Attribute{
				Description:         "Unix timestamp at which the environment was created.",
				MarkdownDescription: "Unix timestamp at which the environment was created.",
				Computed:            true,
			},
			"updated_at": schema.Int64Attribute{
				Description:         "Unix timestamp at which the environment was last updated.",
				MarkdownDescription: "Unix timestamp at which the environment was last updated.",
				Computed:            true,
			},
		},
	}
}

type EnvironmentModel struct {
	Organization           types.String `tfsdk:"organization"`
	Canonical              types.String `tfsdk:"canonical"`
	Name                   types.String `tfsdk:"name"`
	Description            types.String `tfsdk:"description"`
	Type                   types.String `tfsdk:"type"`
	Owner                  types.String `tfsdk:"owner"`
	CloudAccountCanonicals types.List   `tfsdk:"cloud_account_canonicals"`
	Variables              types.List   `tfsdk:"variables"`
	ResourcesCount         types.Int64  `tfsdk:"resources_count"`
	ID                     types.Int64  `tfsdk:"id"`
	CreatedAt              types.Int64  `tfsdk:"created_at"`
	UpdatedAt              types.Int64  `tfsdk:"updated_at"`
}
