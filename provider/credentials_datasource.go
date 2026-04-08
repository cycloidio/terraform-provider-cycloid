package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/datasource_credentials"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/sanity-io/litter"
)

var _ datasource.DataSource = &credentialsDataSource{}

type credentialsDatasourceModel = datasource_credentials.CredentialsModel

type credentialsDataSource struct {
	provider *CycloidProvider
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

func (s *credentialsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data credentialsDatasourceModel

	// Read terraform configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organization := getOrganizationCanonical(*s.provider, data.Organization)

	query := url.Values{
		"page_index": []string{"1"},
		"page_size":  []string{"1000"},
	}

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

	for _, credentialType := range credentialTypes {
		query.Add("credential_types", credentialType)
	}

	var credentials []*CredentialSimple
	_, err := s.provider.Middleware.GenericRequest(middleware.Request{
		Method:       "GET",
		Organization: &organization,
		Route:        []string{"organizations", organization, "credentials"},
		Query:        query,
	}, &credentials)
	if err != nil {
		resp.Diagnostics.AddError("failed to list credentials", err.Error())
		return
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
