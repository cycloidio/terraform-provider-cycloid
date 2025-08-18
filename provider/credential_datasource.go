package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/common"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/datasource_credential"
	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &credentialDataSource{}

type credentialDatasourceModel = datasource_credential.CredentialModel

type credentialDataSource struct {
	provider provider_cycloid.CycloidModel
}

func NewCredentialDataSource() datasource.DataSource {
	return &credentialDataSource{}
}

func (s credentialDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_credential"
}

func (s *credentialDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	schema := datasource_credential.CredentialDataSourceSchema(ctx)
	// adding schema description here is simpler than writing multi-line
	// description in a json file.
	description := fmt.Sprintln(
		"This datasource allows you to fetch a credential and its value.",
		"\nYou can define a specific organiztion with the `organization` attribute or it will default the the",
		"provider's organization settings.\n",
		"\nThe populated fields will depend on the credential types.",
	)
	schema.Description = description
	schema.MarkdownDescription = description
	resp.Schema = schema
}

func (s *credentialDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("path"),
			path.MatchRoot("canonical"),
		),
	}
}

func (s *credentialDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	pv, ok := req.ProviderData.(provider_cycloid.CycloidModel)
	if !ok {
		tflog.Error(ctx, "Unable to init client")
	}

	s.provider = pv
}

func (s *credentialDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data credentialDatasourceModel

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
	api := common.NewAPI(
		common.WithURL(s.provider.Url.ValueString()),
		common.WithToken(s.provider.Jwt.ValueString()),
	)
	m := middleware.NewMiddleware(api)

	canonical := data.Canonical.ValueString()

	credential, err := m.GetCredential(organization, canonical)
	if err != nil || credential == nil {
		resp.Diagnostics.AddError("failed to get credential with canonical '"+canonical+"'", err.Error())
		return
	}

	dataDiags := credentialDatasourceModelToData(ctx, organization, credential, &data)
	resp.Diagnostics.Append(dataDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data to terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// credentialCYModelToData converts the 'cred' into the 'credentialResourceModel'
func credentialDatasourceModelToData(ctx context.Context, org string, credential *models.Credential, data *credentialDatasourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if credential.Owner != nil {
		data.Owner = types.StringPointerValue(credential.Owner.Username)
	} else {
		data.Owner = types.StringValue("")
	}

	data.Name = types.StringPointerValue(credential.Name)
	data.Description = types.StringValue(credential.Description)
	data.Canonical = types.StringPointerValue(credential.Canonical)
	data.Type = types.StringPointerValue(credential.Type)
	data.Path = types.StringPointerValue(credential.Path)
	data.Organization = types.StringValue(org)
	keysValues, keysDiag := types.ListValueFrom(ctx, types.StringType, credential.Keys)
	diags.Append(keysDiag...)
	if diags.HasError() {
		return diags
	}
	data.Keys = keysValues

	var rawValue basetypes.MapValue
	if *credential.Type == "custom" {
		var rawDiags diag.Diagnostics
		rawValue, rawDiags = types.MapValueFrom(ctx, types.StringType, credential.Raw.Raw)
		if rawDiags.HasError() {
			diags.Append(rawDiags...)
			return diags
		}
	} else {
		rawValue = types.MapNull(types.StringType)
	}

	bodyValue, bodyDiags := datasource_credential.NewBodyValue(
		datasource_credential.NewBodyValueNull().AttributeTypes(ctx),
		map[string]attr.Value{
			"access_key":      types.StringValue(credential.Raw.AccessKey),
			"secret_key":      types.StringValue(credential.Raw.SecretKey),
			"account_name":    types.StringValue(credential.Raw.AccountName),
			"auth_url":        types.StringValue(credential.Raw.AuthURL),
			"ca_cert":         types.StringValue(credential.Raw.CaCert),
			"client_id":       types.StringValue(credential.Raw.ClientID),
			"client_secret":   types.StringValue(credential.Raw.ClientSecret),
			"domain_id":       types.StringValue(credential.Raw.DomainID),
			"json_key":        types.StringValue(credential.Raw.JSONKey),
			"password":        types.StringValue(credential.Raw.Password),
			"environment":     types.StringValue(credential.Raw.Environment),
			"ssh_key":         types.StringValue(credential.Raw.SSHKey),
			"subscription_id": types.StringValue(credential.Raw.SubscriptionID),
			"tenant_id":       types.StringValue(credential.Raw.TenantID),
			"username":        types.StringValue(credential.Raw.Username),
			"raw":             rawValue,
		},
	)
	diags.Append(bodyDiags...)
	if diags.HasError() {
		return diags
	}

	data.Body = bodyValue

	return diags
}
