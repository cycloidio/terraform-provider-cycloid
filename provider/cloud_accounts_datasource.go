package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/terraform-provider-cycloid/datasource_cloud_accounts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &cloudAccountsDataSource{}

type cloudAccountsDataSource struct {
	provider *CycloidProvider
}

func NewCloudAccountsDataSource() datasource.DataSource {
	return &cloudAccountsDataSource{}
}

func (s *cloudAccountsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_accounts"
}

func (s *cloudAccountsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_cloud_accounts.CloudAccountsDataSourceSchema(ctx)
}

func (s *cloudAccountsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pv, ok := req.ProviderData.(*CycloidProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider data at Configure()",
			fmt.Sprintf("Expected *CycloidProvider, got: %T. Please report this issue.", req.ProviderData),
		)
		return
	}
	s.provider = pv
}

var cloudAccountObjAttrTypes = map[string]attr.Type{
	"canonical":            types.StringType,
	"name":                 types.StringType,
	"cloud_provider":       types.StringType,
	"credential_canonical": types.StringType,
	"description":          types.StringType,
	"owner":                types.StringType,
	"id":                   types.Int64Type,
}

func (s *cloudAccountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasource_cloud_accounts.CloudAccountsModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*s.provider, data.Organization)
	cloudProviderFilter := data.CloudProvider.ValueString()

	cas, _, err := s.provider.Middleware.ListCloudAccounts(org)
	if err != nil {
		resp.Diagnostics.AddError("failed to list cloud accounts", err.Error())
		return
	}

	items := make([]attr.Value, 0, len(cas))
	for _, ca := range cas {
		if cloudProviderFilter != "" && (ca.CloudProvider == nil || *ca.CloudProvider != cloudProviderFilter) {
			continue
		}

		credCanonical := types.StringNull()
		if ca.Credential != nil {
			credCanonical = types.StringPointerValue(ca.Credential.Canonical)
		}

		ownerStr := ""
		if ca.Owner != nil && ca.Owner.Username != nil {
			ownerStr = *ca.Owner.Username
		}

		obj, objDiags := types.ObjectValue(cloudAccountObjAttrTypes, map[string]attr.Value{
			"canonical":            types.StringPointerValue(ca.Canonical),
			"name":                 types.StringPointerValue(ca.Name),
			"cloud_provider":       types.StringPointerValue(ca.CloudProvider),
			"credential_canonical": credCanonical,
			"description":          types.StringValue(ca.Description),
			"owner":                types.StringValue(ownerStr),
			"id":                   ptrUint32ToInt64(ca.ID),
		})
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		items = append(items, obj)
	}

	listVal, listDiags := types.ListValue(types.ObjectType{AttrTypes: cloudAccountObjAttrTypes}, items)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Organization = types.StringValue(org)
	data.CloudAccounts = listVal

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
