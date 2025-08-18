package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/datasource_credentials"
	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/sanity-io/litter"
)

var _ datasource.DataSource = &credentialsDataSource{}

type credentialsDatasourceModel = datasource_credentials.CredentialsModel

type credentialsDataSource struct {
	provider provider_cycloid.CycloidModel
}

func NewCredentialsDataSource() datasource.DataSource {
	return &credentialsDataSource{}
}

func (s credentialsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_credentials"
}

func (s *credentialsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	schema := datasource_credentials.CredentialsDataSourceSchema(ctx)
	// adding schema description here is simpler than writing multi-line
	// description in a json file.
	description := fmt.Sprintln(
		"This datasource allows you to list the credentials of the designated cycloid organization.",
		"\nYou can define a specific organiztion with the `organization` attribute or it will default the the",
		"provider's organization settings.\n",
		"\nCredentials types can be filtered using the `credentials_types` attribute, you can fill more than one.",
		"\nThis datasource will only return the credentials metadata, if you need the credentials values, you will need to use",
		"the `datasource_credential` to retrieve them.",
	)
	schema.Description = description
	schema.MarkdownDescription = description
	resp.Schema = schema
}

func (s *credentialsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	pv, ok := req.ProviderData.(provider_cycloid.CycloidModel)
	if !ok {
		tflog.Error(ctx, "Unable to init client")
	}

	s.provider = pv
}

func (s *credentialsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data credentialsDatasourceModel

	// Read terraform configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if s.provider.Jwt.IsUnknown() || s.provider.Jwt.IsNull() {
		resp.Diagnostics.AddError("API token for cycloid is mising", "")
		return
	}

	var organization string
	if data.Organization.IsNull() || data.Organization.IsUnknown() {
		organization = s.provider.OrganizationCanonical.ValueString()
	} else {
		organization = data.Organization.ValueString()
	}

	// Fetch logic
	// We will not use the middleware as the current mid version does not support credential_types parameters
	apiUrl := s.provider.Url.ValueString() + "/organizations/" + organization + "/credentials"

	var credentialTypes []string = nil
	if !data.CredentialTypes.IsNull() && !data.CredentialTypes.IsUnknown() {
		elements, listDiags := data.CredentialTypes.ToListValue(ctx)
		resp.Diagnostics.Append(listDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		resp.Diagnostics.Append(
			elements.ElementsAs(ctx, &credentialTypes, false)...,
		)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if credentialTypes != nil {
		apiUrl = apiUrl + "?" + url.Values(map[string][]string{
			"credential_types": credentialTypes,
		}).Encode()
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiUrl, nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create http client, API url may be invalid", err.Error())
		return
	}
	request.Header.Add("Content-Type", "Application/json")
	request.Header.Add("Authorization", "Bearer "+s.provider.Jwt.ValueString())

	client := http.DefaultClient
	response, err := client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("failed to list credentials", err.Error())
		return
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		resp.Diagnostics.AddError("failed to read response body from API", err.Error())
		return
	}

	var credentials []*CredentialSimple
	var payloadData struct {
		Data   []*CredentialSimple `json:"data,omitempty"`
		Errors []struct {
			Message string
			Code    string
			Details []string
		} `json:"errors,omitempty"`
	}

	err = json.Unmarshal(payload, &payloadData)
	if err != nil {
		resp.Diagnostics.AddError("failed to parse JSON response from API", err.Error()+":\n"+litter.Sdump(payloadData.Errors))
		return
	}
	if payloadData.Data != nil {
		credentials = payloadData.Data
	}

	diags := dataModelFrom(ctx, &data, credentials)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Save data to terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func dataModelFrom(ctx context.Context, data *credentialsDatasourceModel, credentials []*CredentialSimple) diag.Diagnostics {
	var diags diag.Diagnostics

	credentialModelType := datasource_credentials.NewCredentialsValueNull().AttributeTypes(ctx)
	credentialsLen := len(credentials)
	credentialValues := make([]attr.Value, credentialsLen)
	for index, credential := range credentials {
		keys, keyDiags := types.ListValueFrom(ctx, types.StringType, credential.Keys)
		diags.Append(keyDiags...)

		value, valueDiags := datasource_credentials.NewCredentialsValue(
			credentialModelType,
			map[string]attr.Value{
				"canonical":   types.StringPointerValue(credential.Canonical),
				"description": types.StringValue(credential.Description),
				"keys":        keys,
				"name":        types.StringPointerValue(credential.Name),
				"owner":       types.StringValue(credential.GetOwnerCanonical()),
				"path":        types.StringPointerValue(credential.Path),
				"type":        types.StringPointerValue(credential.Type),
			},
		)
		if valueDiags.HasError() {
			diags.AddError("Hello there", litter.Sdump(credentialModelType))
			diags.Append(valueDiags...)
			return diags
		}

		credentialValues[index] = value
	}

	listValue, listDiags := types.ListValue(
		datasource_credentials.NewCredentialsValueNull().Type(ctx),
		credentialValues,
	)
	if listDiags.HasError() {
		diags.Append(listDiags...)
		return diags
	}

	data.Credentials = listValue

	return diags
}

type CredentialSimple struct {
	models.CredentialSimple
	Description string
}

func (c *CredentialSimple) GetOwnerCanonical() string {
	if c.Owner == nil {
		return ""
	}

	if c.Owner.Username != nil {
		return *c.Owner.Username
	}

	return ""
}
