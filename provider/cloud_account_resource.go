package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cycloidio/cycloid-cli/gen/models"
	"github.com/cycloidio/terraform-provider-cycloid/resource_cloud_account"
)

var (
	_ resource.Resource                = (*cloudAccountResource)(nil)
	_ resource.ResourceWithImportState = (*cloudAccountResource)(nil)
)

func NewCloudAccountResource() resource.Resource {
	return &cloudAccountResource{}
}

type cloudAccountResource struct {
	provider *CycloidProvider
}

type cloudAccountResourceModel resource_cloud_account.CloudAccountModel

func (r *cloudAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_account"
}

func (r *cloudAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_cloud_account.CloudAccountResourceSchema(ctx)
}

func (r *cloudAccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.provider = pv
}

func (r *cloudAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data cloudAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	name := data.Name.ValueString()
	canonical := data.Canonical.ValueString()

	var err error
	name, canonical, err = NameOrCanonical(name, canonical)
	if err != nil {
		resp.Diagnostics.AddError("failed to infer cloud account canonical", err.Error())
		return
	}

	body := &models.NewCloudAccount{
		Name:                &name,
		Canonical:           canonical,
		CloudProvider:       data.CloudProvider.ValueStringPointer(),
		CredentialCanonical: data.CredentialCanonical.ValueStringPointer(),
		Description:         data.Description.ValueString(),
		Owner:               data.Owner.ValueString(),
	}

	m := r.provider.Middleware
	_, _, err = m.CreateCloudAccount(org, body)
	if err != nil {
		resp.Diagnostics.AddError("failed to create cloud account", err.Error())
		return
	}

	// CreateCloudAccount returns *CloudAccount (no detail), re-read for full state.
	ca, _, err := m.GetCloudAccount(org, canonical)
	if err != nil {
		resp.Diagnostics.AddError("failed to read cloud account after creation", err.Error())
		return
	}

	cloudAccountCYModelToData(org, ca, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *cloudAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data cloudAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	canonical := data.Canonical.ValueString()

	ca, _, err := r.provider.Middleware.GetCloudAccount(org, canonical)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("failed to read cloud account", err.Error())
		return
	}

	cloudAccountCYModelToData(org, ca, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *cloudAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data cloudAccountResourceModel
	var stateData cloudAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	canonical := Coalesce(stateData.Canonical.ValueString(), data.Canonical.ValueString())
	name := data.Name.ValueString()

	body := &models.UpdateCloudAccount{
		Name:                &name,
		CredentialCanonical: data.CredentialCanonical.ValueStringPointer(),
		Description:         data.Description.ValueString(),
		Owner:               data.Owner.ValueString(),
	}

	m := r.provider.Middleware
	_, _, err := m.UpdateCloudAccount(org, canonical, body)
	if err != nil {
		resp.Diagnostics.AddError("failed to update cloud account", err.Error())
		return
	}

	ca, _, err := m.GetCloudAccount(org, canonical)
	if err != nil {
		resp.Diagnostics.AddError("failed to read cloud account after update", err.Error())
		return
	}

	cloudAccountCYModelToData(org, ca, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *cloudAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data cloudAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	canonical := data.Canonical.ValueString()

	_, err := r.provider.Middleware.DeleteCloudAccount(org, canonical)
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError("failed to delete cloud account", err.Error())
	}
}

func (r *cloudAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("canonical"), req, resp)
}

func cloudAccountCYModelToData(org string, ca *models.CloudAccountDetail, data *cloudAccountResourceModel) {
	data.Organization = types.StringValue(org)
	data.Canonical = types.StringPointerValue(ca.Canonical)
	data.Name = types.StringPointerValue(ca.Name)
	data.CloudProvider = types.StringPointerValue(ca.CloudProvider)
	data.Description = types.StringValue(ca.Description)
	data.ID = ptrUint32ToInt64(ca.ID)

	if ca.Credential != nil {
		data.CredentialCanonical = types.StringPointerValue(ca.Credential.Canonical)
	} else {
		data.CredentialCanonical = types.StringNull()
	}

	if ca.Owner != nil && ca.Owner.Username != nil {
		data.Owner = types.StringPointerValue(ca.Owner.Username)
	} else {
		data.Owner = types.StringValue("")
	}
}
