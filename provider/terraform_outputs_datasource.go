package provider

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/cycloidio/terraform-provider-cycloid/datasource_terraform_outputs"
	"github.com/cycloidio/terraform-provider-cycloid/internal/dynamic"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &terraformOutputsDataSource{}

type terraformOutputsDatasourceModel = datasource_terraform_outputs.TerraformOutputsModel

type terraformOutputsDataSource struct {
	provider CycloidProvider
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

	pv, ok := req.ProviderData.(CycloidProvider)
	if !ok {
		tflog.Error(ctx, "Unable to init client")
	}

	t.provider = pv
}

func (t *terraformOutputsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data terraformOutputsDatasourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organization := getOrganizationCanonical(t.provider, data.Organization)

	// Fetch logic
	// We will not use the middleware because we need LHS filter that are undocumented
	apiUrl := fmt.Sprintf("%s/organizations/%s/inventory/outputs", t.provider.APIUrl, organization)

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
		params.Add(filter.Attribute+"["+filter.Condition+"]", url.QueryEscape(filter.Value))
	}

	url := apiUrl + "?" + params.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create http client, API url may be invalid", err.Error())
		return
	}
	request.Header.Add("Content-Type", "Application/json")
	request.Header.Add("Authorization", "Bearer "+t.provider.APIKey)

	client := http.DefaultClient
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: t.provider.Insecure,
		},
	}

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
		Data []datasource_terraform_outputs.TerraformOutput `json:"data"`
	}{}
	err = json.Unmarshal(payload, &payloadJSON)
	if err != nil {
		resp.Diagnostics.AddError("failed to read JSON from API", err.Error())
		return
	}
	terraformOutputs := payloadJSON.Data
	terraformOutputsValue, diags := dynamic.AnyToDynamicValue(ctx, terraformOutputs)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	data.Outputs = terraformOutputsValue
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
