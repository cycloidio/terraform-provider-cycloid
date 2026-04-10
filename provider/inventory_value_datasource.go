package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/datasource_inventory_value"
	"github.com/cycloidio/terraform-provider-cycloid/internal/dynamic"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

var _ datasource.DataSource = &inventoryValueDataSource{}

type inventoryValueDatasourceModel = datasource_inventory_value.InventoryValueModel

type inventoryValueDataSource struct {
	provider *CycloidProvider
}

func NewInventoryValueDataSource() datasource.DataSource {
	return &inventoryValueDataSource{}
}

func (i inventoryValueDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inventory_value"
}

func (i *inventoryValueDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	schema := datasource_inventory_value.InventoryValueDataSourceSchema(ctx)
	resp.Schema = schema
}

func (i *inventoryValueDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (i *inventoryValueDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data inventoryValueDatasourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organization := getOrganizationCanonical(*i.provider, data.Organization)

	var filters []datasource_inventory_value.Filter
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
		params.Add(filter.Attribute+"["+filter.Condition+"]", url.QueryEscape(filter.Value))
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

	inventoryValuesLength := len(inventoryValues)
	if inventoryValuesLength > 1 && !data.SelectFirst.ValueBool() {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Inventory filter is not selective enough, we have %d values", inventoryValuesLength),
			"Add the `select_first` argument to select one or use finer filters.",
		)
		return
	}
	if inventoryValuesLength == 0 {
		resp.Diagnostics.AddError("Found no matching inventory value", "Adjust your filters to match your values.")
		return
	}

	valueValue, diags := dynamic.AnyToDynamicValue(ctx, inventoryValues[0])
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	data.Value = valueValue
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
