package provider

import (
	"context"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/common"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/provider/resource_credential"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*credentialResource)(nil)

func NewCredentialResource(p provider.Provider) func() resource.Resource {
	return func() resource.Resource {
		return &credentialResource{
			provider: p,
		}
	}
}

type credentialResource struct {
	provider provider.Provider
}

func (r *credentialResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_credential"
}

func (r *credentialResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_credential.CredentialResourceSchema(ctx)
}

func (r *credentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_credential.CredentialModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(createCredential(ctx, r.provider, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *credentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_credential.CredentialModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *credentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_credential.CredentialModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(updateCredential(ctx, r.provider, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *credentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_credential.CredentialModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Typically this method would contain logic that makes an HTTP call to a remote API, and then stores
// computed results back to the data model. For example purposes, this function just sets all unknown
// Pet values to null to avoid data consistency errors.
func createCredential(ctx context.Context, p provider.Provider, c *resource_credential.CredentialModel) diag.Diagnostics {
	api := common.NewAPI("https://api.staging.cycloid.io/")
	m := middleware.NewMiddleware(api)

	org := c.OrganizationCanonical.ValueString()
	name := c.Name.ValueString()
	ct := c.Type.ValueString()

	rawCred := &models.CredentialRaw{
		AccessKey: c.Body.AccessKey.ValueString(),
		SecretKey: c.Body.SecretKey.ValueString(),
	}

	path := c.Path.ValueString()
	can := c.Canonical.ValueString()
	des := c.Description.ValueString()

	err := m.CreateCredential(org, name, ct, rawCred, path, can, des)
	if err != nil {
		//panic(fmt.Sprintf("%T", err))
		return diag.Diagnostics{diag.NewErrorDiagnostic("failed to create credential", err.Error())}
	}

	// If not an error like
	/*
		│ Error: Provider returned invalid result object after apply
		│
		│ After the apply operation, the provider still indicated an unknown value for cycloid_credential.test_from_tf.body.username. All values must be known after apply, so this is always a bug in the provider and
		│ should be reported in the provider's own repository. Terraform will still save the other known object values in the state.
	*/
	// I'm guessing there should be something better
	//c.Description = types.StringNull()
	c.Owner = types.StringNull()
	c.Data = resource_credential.NewDataValueNull()
	c.CredentialCanonical = types.StringNull()
	c.Canonical = types.StringNull()
	c.Body.AccountName = types.StringNull()
	c.Body.AuthUrl = types.StringNull()
	c.Body.CaCert = types.StringNull()
	c.Body.ClientId = types.StringNull()
	c.Body.ClientSecret = types.StringNull()
	c.Body.DomainId = types.StringNull()
	c.Body.JsonKey = types.StringNull()
	c.Body.Password = types.StringNull()
	c.Body.Raw = types.ObjectNull(resource_credential.RawValue{}.AttributeTypes(ctx))
	c.Body.SshKey = types.StringNull()
	c.Body.SubscriptionId = types.StringNull()
	c.Body.TenantId = types.StringNull()
	c.Body.Username = types.StringNull()

	return nil
}

// Typically this method would contain logic that makes an HTTP call to a remote API, and then stores
// computed results back to the data model. For example purposes, this function just sets all unknown
// Pet values to null to avoid data consistency errors.
func updateCredential(ctx context.Context, p provider.Provider, c *resource_credential.CredentialModel) diag.Diagnostics {
	//api := common.NewAPI()
	//m := middleware.NewMiddleware(api)

	//org := c.OrganizationCanonical.ValueString()
	//name := c.Name.ValueString()
	//ct := c.Type.ValueString()

	//rawCre := &models.CredentialRaw{
	//AccessKey: c.RawValue.AccessKey.ValueString(),
	//SecretKey: c.RawValue.SecretKey.ValueString(),
	//}

	//path := c.Path.ValueString()
	//can := c.Canonical.ValueString()
	//des := c.Description.ValueString()

	//err = m.CreateCredential(org, name, credT, rawCred, path, can, des)
	//if err != nil {
	//return diag.FromErr(err)
	//}

	return nil
}
