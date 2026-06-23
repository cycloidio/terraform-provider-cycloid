package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cycloidio/terraform-provider-cycloid/datasource_environment_types"
)

var _ datasource.DataSource = &environmentTypesDataSource{}

type environmentTypesDataSource struct {
	provider *CycloidProvider
}

func NewEnvironmentTypesDataSource() datasource.DataSource {
	return &environmentTypesDataSource{}
}

func (s *environmentTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_types"
}

func (s *environmentTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_environment_types.EnvironmentTypesDataSourceSchema(ctx)
}

func (s *environmentTypesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

var envTypeObjAttrTypes = map[string]attr.Type{
	"canonical":          types.StringType,
	"name":               types.StringType,
	"color":              types.StringType,
	"is_default":         types.BoolType,
	"environments_count": types.Int64Type,
	"id":                 types.Int64Type,
}

func (s *environmentTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasource_environment_types.EnvironmentTypesModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*s.provider, data.Organization)

	ets, _, err := s.provider.Middleware.ListEnvironmentTypes(org)
	if err != nil {
		resp.Diagnostics.AddError("failed to list environment types", err.Error())
		return
	}

	items := make([]attr.Value, len(ets))
	for i, et := range ets {
		obj, objDiags := types.ObjectValue(envTypeObjAttrTypes, map[string]attr.Value{
			"canonical":          types.StringPointerValue(et.Canonical),
			"name":               types.StringPointerValue(et.Name),
			"color":              types.StringPointerValue(et.Color),
			"is_default":         types.BoolPointerValue(et.IsDefault),
			"environments_count": ptrUint32ToInt64(et.EnvironmentsCount),
			"id":                 ptrUint32ToInt64(et.ID),
		})
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		items[i] = obj
	}

	listVal, listDiags := types.ListValue(types.ObjectType{AttrTypes: envTypeObjAttrTypes}, items)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Organization = types.StringValue(org)
	data.EnvironmentTypes = listVal

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
