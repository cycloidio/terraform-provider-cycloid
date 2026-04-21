package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/datasource_terraform_output"
	"github.com/cycloidio/terraform-provider-cycloid/internal/dynamic"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &terraformOutputDataSource{}

type terraformOutputDatasourceModel = datasource_terraform_output.TerraformOutputModel

type terraformOutputDataSource struct {
	provider *CycloidProvider
}

func NewTerraformOutputDataSource() datasource.DataSource {
	return &terraformOutputDataSource{}
}

func (t terraformOutputDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_terraform_output"
}

func (t *terraformOutputDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	schema := datasource_terraform_output.TerraformOutputDataSourceSchema(ctx)
	resp.Schema = schema
}

func (t *terraformOutputDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (t *terraformOutputDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data terraformOutputDatasourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organization := getOrganizationCanonical(*t.provider, data.Organization)

	var filters []datasource_terraform_output.Filter = nil
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

	var terraformOutputs []datasource_terraform_output.TerraformOutput
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
	terraformOutputsLength := len(terraformOutputs)
	var terraformOutput datasource_terraform_output.TerraformOutput
	if terraformOutputsLength > 1 && !data.SelectFirst.ValueBool() {
		resp.Diagnostics.AddError(fmt.Sprintf("Output filter is not selective enough, we have %d outputs", terraformOutputsLength), "Add the `select_first` argument to select one or use finer filters.")
		return
	} else if terraformOutputsLength == 0 {
		resp.Diagnostics.AddError("Found no matching terraform output", "Ajust your filters to match your outputs.")
		return
	} else if terraformOutputsLength > 1 && data.SelectFirst.ValueBool() || terraformOutputsLength == 1 {
		terraformOutput = terraformOutputs[0]
	}

	typeValue, diags := dynamic.AnyToDynamicValue(ctx, terraformOutput.Type)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	valueValue, diags := dynamic.AnyToDynamicValue(ctx, terraformOutput.Value)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	outObject, diags := types.ObjectValue(map[string]attr.Type{
		"id":          types.Int64Type,
		"key":         types.StringType,
		"type":        types.DynamicType,
		"value":       types.DynamicType,
		"sensitive":   types.BoolType,
		"description": types.StringType,
	}, map[string]attr.Value{
		"id":          types.Int64Value(int64(terraformOutput.ID)),
		"key":         types.StringValue(terraformOutput.Key),
		"type":        typeValue,
		"value":       valueValue,
		"sensitive":   types.BoolValue(terraformOutput.Sensitive),
		"description": types.StringPointerValue(terraformOutput.Description),
	})
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	data.Output = outObject
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
