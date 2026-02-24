package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/cycloidio/terraform-provider-cycloid/datasource_terraform_output"
	"github.com/cycloidio/terraform-provider-cycloid/internal/dynamic"
	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &terraformOutputDataSource{}

type terraformOutputDatasourceModel = datasource_terraform_output.TerraformOutputModel

type terraformOutputDataSource struct {
	provider provider_cycloid.CycloidModel
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

	pv, ok := req.ProviderData.(provider_cycloid.CycloidModel)
	if !ok {
		tflog.Error(ctx, "Unable to init client")
	}

	t.provider = pv
}

func (t *terraformOutputDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data terraformOutputDatasourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if t.provider.Jwt.IsUnknown() || t.provider.Jwt.IsNull() {
		resp.Diagnostics.AddError("API token for cycloid is mising", "")
		return
	}

	var organization string
	if data.Organization.IsNull() || data.Organization.IsUnknown() {
		organization = t.provider.OrganizationCanonical.ValueString()
	} else {
		organization = data.Organization.ValueString()
	}

	if organization == "" {
		resp.Diagnostics.AddAttributeError(path.Root("organization"), "org should not be empty", "fill it by the provider or the datasource settings")
		return
	}

	// We will not use the middleware because we need LHS filter that are undocumented
	apiUrl := fmt.Sprintf("%s/organizations/%s/inventory/outputs", t.provider.Url.ValueString(), organization)

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
		params.Add(filter.Attribute+"["+filter.Condition+"]", url.QueryEscape(filter.Value))
	}

	url := apiUrl + "?" + params.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create http client, API url may be invalid", err.Error())
		return
	}
	request.Header.Add("Content-Type", "Application/json")
	request.Header.Add("Authorization", "Bearer "+t.provider.Jwt.ValueString())

	client := http.DefaultClient
	response, err := client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("failed to list credentials", err.Error())
		return
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			tflog.Error(ctx, "failed to close body on http connection", map[string]any{
				"internal_error": err.Error(),
			})
		}
	}()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		resp.Diagnostics.AddError("failed to read response body from API", err.Error())
		return
	}

	payloadJSON := struct {
		Data []datasource_terraform_output.TerraformOutput `json:"data"`
	}{}
	err = json.Unmarshal(payload, &payloadJSON)
	if err != nil {
		resp.Diagnostics.AddError("failed to read JSON from API", err.Error())
		return
	}

	terraformOutputs := payloadJSON.Data
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
