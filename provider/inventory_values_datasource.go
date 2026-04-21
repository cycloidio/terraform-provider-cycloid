package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/datasource_inventory_values"
	"github.com/cycloidio/terraform-provider-cycloid/internal/dynamic"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

var _ datasource.DataSource = &inventoryValuesDataSource{}

type inventoryValuesDatasourceModel = datasource_inventory_values.InventoryValuesModel

type inventoryValuesDataSource struct {
	provider *CycloidProvider
}

func NewInventoryValuesDataSource() datasource.DataSource {
	return &inventoryValuesDataSource{}
}

func (i inventoryValuesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inventory_values"
}

func (i *inventoryValuesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	schema := datasource_inventory_values.InventoryValuesDataSourceSchema(ctx)
	resp.Schema = schema
}

func (i *inventoryValuesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	i.provider = pv
}

func (i *inventoryValuesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data inventoryValuesDatasourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organization := getOrganizationCanonical(*i.provider, data.Organization)

	var filters []datasource_inventory_values.Filter
	if !data.Filters.IsNull() && !data.Filters.IsUnknown() {
		elements, listDiags := data.Filters.ToListValue(ctx)
		resp.Diagnostics.Append(listDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		resp.Diagnostics.Append(elements.ElementsAs(ctx, &filters, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	params := make(url.Values)
	for _, filter := range filters {
		params.Add(filter.Attribute+"["+filter.Condition+"]", filter.Value)
	}

	var inventoryValues []map[string]any
	_, err := i.provider.Middleware.GenericRequest(middleware.Request{
		Method:       "GET",
		Organization: &organization,
		Route:        []string{"organizations", organization, "inventory"},
		Query:        params,
	}, &inventoryValues)
	if err != nil {
		resp.Diagnostics.AddError("failed to list inventory values", err.Error())
		return
	}

	inventoryValuesValue, diags := dynamic.AnyToDynamicValue(ctx, inventoryValues)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	data.Values = inventoryValuesValue
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
