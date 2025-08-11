package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/common"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
	"github.com/cycloidio/terraform-provider-cycloid/resource_credential"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*credentialResource)(nil)

func NewCredentialResource() resource.Resource {
	return &credentialResource{}
}

type credentialResource struct {
	provider provider_cycloid.CycloidModel
}

type credentialResourceModel resource_credential.CredentialModel

func (r *credentialResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_credential"
}

func (r *credentialResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_credential.CredentialResourceSchema(ctx)
}

func (r *credentialResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	pv, ok := req.ProviderData.(provider_cycloid.CycloidModel)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}
	r.provider = pv
}

func (r *credentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data credentialResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	name := data.Name.ValueString()
	ct := data.Type.ValueString()

	err := validateCredential(data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create credential",
			err.Error(),
		)
		return
	}

	rawCred, diags := dataRawToCredentialRawCYModel(ctx, data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	path := data.Path.ValueString()
	can := data.Canonical.ValueString()
	des := data.Description.ValueString()

	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)

	cred, err := mid.CreateCredential(orgCan, name, ct, rawCred, path, can, des)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create credential",
			err.Error(),
		)
		return
	}

	credentialCYModelToData(ctx, orgCan, cred, &data)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *credentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data credentialResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	can := data.Canonical.ValueString()

	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)

	cred, err := mid.GetCredential(orgCan, can)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read credential",
			err.Error(),
		)
		return
	}

	credentialCYModelToData(ctx, orgCan, cred, &data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *credentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data credentialResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := validateCredential(data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update credential",
			err.Error(),
		)
		return
	}
	// Update API call logic
	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	name := data.Name.ValueString()
	ct := data.Type.ValueString()
	rawCred, diags := dataRawToCredentialRawCYModel(ctx, data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	path := data.Path.ValueString()
	can := data.Canonical.ValueString()
	des := data.Description.ValueString()

	// As the canonical is not required to be set we read it from the
	// state as we set it on creation and we need it to update the
	// credential to the API
	if can == "" {
		var plandata credentialResourceModel
		// Read Terraform prior state data into the model
		resp.Diagnostics.Append(req.State.Get(ctx, &plandata)...)
		if resp.Diagnostics.HasError() {
			return
		}
		can = plandata.Canonical.ValueString()
	}

	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)

	cred, err := mid.UpdateCredential(orgCan, name, ct, rawCred, path, can, des)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update credential",
			err.Error(),
		)
		return
	}

	credentialCYModelToData(ctx, orgCan, cred, &data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *credentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data credentialResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	can := data.Canonical.ValueString()

	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)

	// Delete API call logic
	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	err := mid.DeleteCredential(orgCan, can)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete credential",
			err.Error(),
		)
		return
	}
}

// credentialCYModelToData converts the 'cred' into the 'credentialResourceModel'
func credentialCYModelToData(ctx context.Context, org string, credential *models.Credential, data *credentialResourceModel) diag.Diagnostics {
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
	data.OrganizationCanonical = types.StringValue(org)
	data.Body.AccessKey = types.StringValue(credential.Raw.AccessKey)
	data.Body.SecretKey = types.StringValue(credential.Raw.SecretKey)
	data.Body.AccountName = types.StringValue(credential.Raw.AccountName)
	data.Body.AuthUrl = types.StringValue(credential.Raw.AuthURL)
	data.Body.CaCert = types.StringValue(credential.Raw.CaCert)
	data.Body.ClientId = types.StringValue(credential.Raw.ClientID)
	data.Body.ClientSecret = types.StringValue(credential.Raw.ClientSecret)
	data.Body.DomainId = types.StringValue(credential.Raw.DomainID)
	data.Body.JsonKey = types.StringValue(credential.Raw.JSONKey)
	data.Body.Password = types.StringValue(credential.Raw.Password)
	data.Body.Environment = types.StringValue(credential.Raw.Environment)
	data.Body.SshKey = types.StringValue(credential.Raw.SSHKey)
	data.Body.SubscriptionId = types.StringValue(credential.Raw.SubscriptionID)
	data.Body.TenantId = types.StringValue(credential.Raw.TenantID)
	data.Body.Username = types.StringValue(credential.Raw.Username)
	if data.Type.ValueString() == "custom" {
		var rawDiags diag.Diagnostics
		data.Body.Raw, rawDiags = types.MapValueFrom(ctx, data.Body.Raw.ElementType(ctx), credential.Raw.Raw)
		if rawDiags.HasError() {
			diags.Append(rawDiags...)
			return diags
		}
	} else {
		data.Body.Raw = types.MapNull(data.Body.Raw.ElementType(ctx))
	}

	return diags
}

func validateCredential(data credentialResourceModel) error {
	if strings.HasPrefix(data.Body.SshKey.ValueString(), "\n") || strings.HasSuffix(data.Body.SshKey.ValueString(), "\n") {
		return fmt.Errorf("Expected 'body.ssh_key' to not have \\n at the beginning or end of it, use 'chomp()' Terraform function to fix this")
	}

	return nil
}

func dataRawToCredentialRawCYModel(ctx context.Context, data credentialResourceModel) (*models.CredentialRaw, diag.Diagnostics) {
	rawCred := &models.CredentialRaw{
		AccessKey:      data.Body.AccessKey.ValueString(),
		SecretKey:      data.Body.SecretKey.ValueString(),
		AccountName:    data.Body.AccountName.ValueString(),
		AuthURL:        data.Body.AuthUrl.ValueString(),
		CaCert:         data.Body.CaCert.ValueString(),
		ClientID:       data.Body.ClientId.ValueString(),
		ClientSecret:   data.Body.ClientSecret.ValueString(),
		DomainID:       data.Body.DomainId.ValueString(),
		JSONKey:        data.Body.JsonKey.ValueString(),
		Password:       data.Body.Password.ValueString(),
		SSHKey:         data.Body.SshKey.ValueString(),
		SubscriptionID: data.Body.SubscriptionId.ValueString(),
		TenantID:       data.Body.TenantId.ValueString(),
		Username:       data.Body.Username.ValueString(),
	}

	if data.Type.ValueString() != "custom" || data.Body.Raw.IsNull() || data.Body.Raw.IsUnknown() {
		rawCred.Raw = nil
		return rawCred, nil
	}

	if data.Type.ValueString() == "custom" {
		elements := make(map[string]types.String, len(data.Body.Raw.Elements()))
		diags := data.Body.Raw.ElementsAs(ctx, &elements, false)
		if diags.HasError() {
			return rawCred, diags
		}

		customMapString := make(map[string]string, len(elements))
		for k, v := range elements {
			customMapString[k] = v.ValueString()
		}

		rawCred.Raw = customMapString
	}

	return rawCred, nil
}
