package provider

import (
	"context"

	"github.com/cycloidio/cycloid-cli/cmd/cycloid/common"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
	"github.com/cycloidio/terraform-provider-cycloid/resource_organization"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*organizationResource)(nil)

func NewOrganizationResource() resource.Resource {
	return &organizationResource{}
}

type organizationResource struct {
	provider provider_cycloid.CycloidModel
}

type organizationResourceModel resource_organization.OrganizationModel

func (r *organizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *organizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_organization.OrganizationResourceSchema(ctx)
}

func (r *organizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
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

func (r *organizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)
	co, err := mid.CreateOrganizationChild(data.Name.ValueString(), r.provider.OrganizationCanonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable create organization child",
			err.Error(),
		)
		return
	}
	data.Canonical = types.StringPointerValue(co.Canonical)
	data.Name = types.StringPointerValue(co.Name)
	data.Data = resource_organization.NewDataValueNull()
	data.OrganizationCanonical = r.provider.OrganizationCanonical

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)
	org, err := mid.GetOrganization(data.Canonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable get organization",
			err.Error(),
		)
		return
	}

	data.Name = types.StringPointerValue(org.Name)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data organizationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update API call logic

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)
	err := mid.DeleteOrganization(data.Canonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable delete organization",
			err.Error(),
		)
		return
	}
}
