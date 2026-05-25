package datasource_cloud_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func CloudAccountDataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Look up a single Cycloid cloud account by canonical.",
		MarkdownDescription: "Look up a single Cycloid cloud account by canonical.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical to look up the cloud account in. Defaults to the provider's `default_organization`.",
				MarkdownDescription: "The organization canonical to look up the cloud account in. Defaults to the provider's `default_organization`.",
				Optional:            true,
				Computed:            true,
			},
			"canonical": schema.StringAttribute{
				Description:         "Canonical of the cloud account to look up.",
				MarkdownDescription: "Canonical of the cloud account to look up.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				Description:         "Display name of the cloud account.",
				MarkdownDescription: "Display name of the cloud account.",
				Computed:            true,
			},
			"cloud_provider": schema.StringAttribute{
				Description:         "Canonical of the associated cloud provider.",
				MarkdownDescription: "Canonical of the associated cloud provider.",
				Computed:            true,
			},
			"credential_canonical": schema.StringAttribute{
				Description:         "Canonical of the credential currently linked to this cloud account.",
				MarkdownDescription: "Canonical of the credential currently linked to this cloud account.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				Description:         "Free-form description of the cloud account.",
				MarkdownDescription: "Free-form description of the cloud account.",
				Computed:            true,
			},
			"owner": schema.StringAttribute{
				Description:         "Username of the cloud account owner.",
				MarkdownDescription: "Username of the cloud account owner.",
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

type CloudAccountModel struct {
	Organization        types.String `tfsdk:"organization"`
	Canonical           types.String `tfsdk:"canonical"`
	Name                types.String `tfsdk:"name"`
	CloudProvider       types.String `tfsdk:"cloud_provider"`
	CredentialCanonical types.String `tfsdk:"credential_canonical"`
	Description         types.String `tfsdk:"description"`
	Owner               types.String `tfsdk:"owner"`
	ID                  types.Int64  `tfsdk:"id"`
}
