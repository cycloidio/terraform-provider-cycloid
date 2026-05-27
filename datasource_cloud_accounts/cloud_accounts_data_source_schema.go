package datasource_cloud_accounts

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func CloudAccountsDataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "List Cycloid cloud accounts in an organization, optionally filtered by cloud provider.",
		MarkdownDescription: "List Cycloid cloud accounts in an organization, optionally filtered by cloud provider.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical to list cloud accounts from. Defaults to the provider's `default_organization`.",
				MarkdownDescription: "The organization canonical to list cloud accounts from. Defaults to the provider's `default_organization`.",
				Optional:            true,
				Computed:            true,
			},
			"cloud_provider": schema.StringAttribute{
				Description:         "Optional cloud provider canonical to filter on (e.g. `aws`, `google`, `azurerm`, or a custom canonical).",
				MarkdownDescription: "Optional cloud provider canonical to filter on (e.g. `aws`, `google`, `azurerm`, or a custom canonical).",
				Optional:            true,
			},
			"cloud_accounts": schema.ListNestedAttribute{
				Description:         "Matching cloud accounts.",
				MarkdownDescription: "Matching cloud accounts.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"canonical":            schema.StringAttribute{Computed: true},
						"name":                 schema.StringAttribute{Computed: true},
						"cloud_provider":       schema.StringAttribute{Computed: true},
						"credential_canonical": schema.StringAttribute{Computed: true},
						"description":          schema.StringAttribute{Computed: true},
						"owner":                schema.StringAttribute{Computed: true},
						"id":                   schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

type CloudAccountsModel struct {
	Organization  types.String `tfsdk:"organization"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
	CloudAccounts types.List   `tfsdk:"cloud_accounts"`
}
