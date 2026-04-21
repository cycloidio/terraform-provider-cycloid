package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/datasource_terraform_outputs"
	"github.com/cycloidio/terraform-provider-cycloid/internal/dynamic"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

var _ datasource.DataSource = &terraformOutputsDataSource{}

type terraformOutputsDatasourceModel = datasource_terraform_outputs.TerraformOutputsModel

type terraformOutputsDataSource struct {
	provider *CycloidProvider
}

func NewTerraformOutputsDataSource() datasource.DataSource {
	return &terraformOutputsDataSource{}
}

func (t terraformOutputsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_terraform_outputs"
}

func (t *terraformOutputsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	schema := datasource_terraform_outputs.TerraformOutputsDataSourceSchema(ctx)
	resp.Schema = schema
}

func (t *terraformOutputsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	t.provider = pv
}

func (t *terraformOutputsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data terraformOutputsDatasourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organization := getOrganizationCanonical(*t.provider, data.Organization)

	var filters []datasource_terraform_outputs.Filter = nil
	if !data.Filters.IsNull() && !data.Filters.IsUnknown() {
		elements, listDiags := data.Filters.ToListValue(ctx)
		resp.Diagnostics.Append(listDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		resp.Diagnostics.Append(
			elements.ElementsAs(ctx, &filters, false)...,
		)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	params := make(url.Values)
	for _, filter := range filters {
		params.Add(filter.Attribute+"["+filter.Condition+"]", filter.Value)
	}

	var terraformOutputs []datasource_terraform_outputs.TerraformOutput
	_, err := t.provider.Middleware.GenericRequest(middleware.Request{
		Method:       "GET",
		Organization: &organization,
		Route:        []string{"organizations", organization, "inventory", "outputs"},
		Query:        params,
	}, &terraformOutputs)
	if err != nil {
		resp.Diagnostics.AddError("failed to list terraform outputs", err.Error())
		return
	}
	terraformOutputsValue, diags := dynamic.AnyToDynamicValue(ctx, terraformOutputs)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	data.Outputs = terraformOutputsValue
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
