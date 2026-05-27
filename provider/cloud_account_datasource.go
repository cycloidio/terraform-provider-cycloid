package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/terraform-provider-cycloid/datasource_cloud_account"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &cloudAccountDataSource{}

type cloudAccountDataSource struct {
	provider *CycloidProvider
}

func NewCloudAccountDataSource() datasource.DataSource {
	return &cloudAccountDataSource{}
}

func (s *cloudAccountDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_account"
}

func (s *cloudAccountDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_cloud_account.CloudAccountDataSourceSchema(ctx)
}

func (s *cloudAccountDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (s *cloudAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasource_cloud_account.CloudAccountModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*s.provider, data.Organization)
	canonical := data.Canonical.ValueString()

	ca, _, err := s.provider.Middleware.GetCloudAccount(org, canonical)
	if err != nil {
		resp.Diagnostics.AddError("failed to read cloud account '"+canonical+"'", err.Error())
		return
	}

	data.Organization = types.StringValue(org)
	data.Canonical = types.StringPointerValue(ca.Canonical)
	data.Name = types.StringPointerValue(ca.Name)
	data.CloudProvider = types.StringPointerValue(ca.CloudProvider)
	data.Description = types.StringValue(ca.Description)
	data.ID = ptrUint32ToInt64(ca.ID)

	if ca.Credential != nil {
		data.CredentialCanonical = types.StringPointerValue(ca.Credential.Canonical)
	} else {
		data.CredentialCanonical = types.StringNull()
	}

	if ca.Owner != nil && ca.Owner.Username != nil {
		data.Owner = types.StringPointerValue(ca.Owner.Username)
	} else {
		data.Owner = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
