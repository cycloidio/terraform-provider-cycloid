package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cycloidio/terraform-provider-cycloid/datasource_environment_type"
)

var _ datasource.DataSource = &environmentTypeDataSource{}

type environmentTypeDataSource struct {
	provider *CycloidProvider
}

func NewEnvironmentTypeDataSource() datasource.DataSource {
	return &environmentTypeDataSource{}
}

func (s *environmentTypeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_type"
}

func (s *environmentTypeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_environment_type.EnvironmentTypeDataSourceSchema(ctx)
}

func (s *environmentTypeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (s *environmentTypeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasource_environment_type.EnvironmentTypeModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*s.provider, data.Organization)
	canonical := data.Canonical.ValueString()

	et, _, err := s.provider.Middleware.GetEnvironmentType(org, canonical)
	if err != nil {
		resp.Diagnostics.AddError("failed to read environment type '"+canonical+"'", err.Error())
		return
	}

	data.Organization = types.StringValue(org)
	data.Canonical = types.StringPointerValue(et.Canonical)
	data.Name = types.StringPointerValue(et.Name)
	data.Color = types.StringPointerValue(et.Color)
	data.IsDefault = types.BoolPointerValue(et.IsDefault)
	data.EnvironmentsCount = ptrUint32ToInt64(et.EnvironmentsCount)
	data.ID = ptrUint32ToInt64(et.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
