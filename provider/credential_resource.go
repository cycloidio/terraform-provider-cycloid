package provider

import (
	"context"
	"fmt"

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
	// TODO: https://github.com/cycloidio/terraform-provider-cycloid/issues/3
	if ct == "custom" {
		resp.Diagnostics.AddError(
			"'custom' type is not yet supported on credentials, for more information check https://github.com/cycloidio/terraform-provider-cycloid/issues/3",
			fmt.Errorf("attribute 'type=\"custom\"' is not supported").Error(),
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

	cred, err := mid.CreateCredential(r.provider.OrganizationCanonical.ValueString(), name, ct, rawCred, path, can, des)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable create credential",
			err.Error(),
		)
		return
	}

	credentialCYModelToData(ctx, r.provider.OrganizationCanonical.ValueString(), cred, &data)

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

	cred, err := mid.GetCredential(r.provider.OrganizationCanonical.ValueString(), can)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable read credential",
			err.Error(),
		)
		return
	}

	credentialCYModelToData(ctx, r.provider.OrganizationCanonical.ValueString(), cred, &data)

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

	cred, err := mid.UpdateCredential(r.provider.OrganizationCanonical.ValueString(), name, ct, rawCred, path, can, des)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable create credential",
			err.Error(),
		)
		return
	}

	credentialCYModelToData(ctx, r.provider.OrganizationCanonical.ValueString(), cred, &data)

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

	// Delete API call logic
	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	err := mid.DeleteCredential(r.provider.OrganizationCanonical.ValueString(), can)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable delete credential",
			err.Error(),
		)
		return
	}

}

// credentialCYModelToData converts the 'cred' into the 'credentialResourceModel'
func credentialCYModelToData(ctx context.Context, org string, cred *models.Credential, data *credentialResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	data.Name = types.StringPointerValue(cred.Name)
	data.Description = types.StringValue(cred.Description)
	data.Owner = types.StringPointerValue(cred.Owner.Username)
	data.Canonical = types.StringPointerValue(cred.Canonical)
	data.OrganizationCanonical = types.StringValue(org)
	data.Body.AccessKey = types.StringValue(cred.Raw.AccessKey)
	data.Body.SecretKey = types.StringValue(cred.Raw.SecretKey)
	data.Body.AccountName = types.StringValue(cred.Raw.AccountName)
	data.Body.AuthUrl = types.StringValue(cred.Raw.AuthURL)
	data.Body.CaCert = types.StringValue(cred.Raw.CaCert)
	data.Body.ClientId = types.StringValue(cred.Raw.ClientID)
	data.Body.ClientSecret = types.StringValue(cred.Raw.ClientSecret)
	data.Body.DomainId = types.StringValue(cred.Raw.DomainID)
	data.Body.JsonKey = types.StringValue(cred.Raw.JSONKey)
	data.Body.Password = types.StringValue(cred.Raw.Password)
	var ov types.Object
	// TODO: https://github.com/cycloidio/terraform-provider-cycloid/issues/3
	//if cred.Raw.Raw != nil {
	//at := make(map[string]attr.Type)
	//av := make(map[string]attr.Value)
	//for k, v := range cred.Raw.Raw.(map[string]interface{}) {
	//switch v.(type) {
	//case string:
	//at[k] = types.StringType
	//av[k] = types.StringValue(v.(string))
	//case int64:
	//at[k] = types.Int64Type
	//av[k] = types.Int64Value(v.(int64))
	//case bool:
	//at[k] = types.BoolType
	//av[k] = types.BoolValue(v.(bool))
	//case float64:
	//at[k] = types.Float64Type
	//av[k] = types.Float64Value(v.(float64))
	//}
	//}
	//ov, diags = types.ObjectValue(at, av)
	//} else {
	//ov, diags = resource_credential.NewRawValueNull().ToObjectValue(ctx)
	//}
	data.Body.Raw = ov
	data.Body.SshKey = types.StringValue(cred.Raw.SSHKey)
	data.Body.SubscriptionId = types.StringValue(cred.Raw.SubscriptionID)
	data.Body.TenantId = types.StringValue(cred.Raw.TenantID)
	data.Body.Username = types.StringValue(cred.Raw.Username)

	return diags
}

func dataRawToCredentialRawCYModel(ctx context.Context, data credentialResourceModel) (*models.CredentialRaw, diag.Diagnostics) {
	var (
		raw map[string]interface{}
	)
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
		Raw:            raw,
		SSHKey:         data.Body.SshKey.ValueString(),
		SubscriptionID: data.Body.SubscriptionId.ValueString(),
		TenantID:       data.Body.TenantId.ValueString(),
		Username:       data.Body.Username.ValueString(),
	}

	return rawCred, nil
}
