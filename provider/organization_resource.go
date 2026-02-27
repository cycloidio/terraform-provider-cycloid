package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/resource_organization"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*organizationResource)(nil)

func NewOrganizationResource() resource.Resource {
	return &organizationResource{}
}

type organizationResource struct {
	provider *CycloidProvider
}

type organizationResourceModel resource_organization.OrganizationModel

func (r *organizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *organizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_organization.OrganizationResourceSchema(ctx)
}

func (r *organizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *organizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mid := r.provider.Middleware
	orgCan := getOrganizationCanonical(*r.provider, data.OrganizationCanonical)

	co, err := mid.CreateOrganizationChild(orgCan, data.Canonical.ValueString(), data.Name.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable create organization child",
			err.Error(),
		)
		return
	}

	organizationCYModelToData(orgCan, co, &data)

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
	mid := r.provider.Middleware

	can := data.Canonical.ValueString()
	if can == "" {
		var plandata organizationResourceModel
		// Read Terraform prior state data into the model
		resp.Diagnostics.Append(req.State.Get(ctx, &plandata)...)
		if resp.Diagnostics.HasError() {
			return
		}
		can = plandata.Canonical.ValueString()
	}

	org, err := mid.GetOrganization(can)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable get organization",
			err.Error(),
		)
		return
	}

	orgCan := getOrganizationCanonical(*r.provider, data.OrganizationCanonical)

	organizationCYModelToData(orgCan, org, &data)

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
	mid := r.provider.Middleware
	can := data.Canonical.ValueString()

	// As the canonical is not required to be set we read it from the
	// state as we set it on creation and we need it to update the
	// credential to the API
	if can == "" {
		var plandata organizationResourceModel
		// Read Terraform prior state data into the model
		resp.Diagnostics.Append(req.State.Get(ctx, &plandata)...)
		if resp.Diagnostics.HasError() {
			return
		}
		can = plandata.Canonical.ValueString()
	}

	uo, err := mid.UpdateOrganization(can, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable create organization child",
			err.Error(),
		)
		return
	}

	orgCan := getOrganizationCanonical(*r.provider, data.OrganizationCanonical)

	organizationCYModelToData(orgCan, uo, &data)

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
	mid := r.provider.Middleware
	err := mid.DeleteOrganization(data.Canonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable delete organization",
			err.Error(),
		)
		return
	}
}

func organizationCYModelToData(org string, o *models.Organization, data *organizationResourceModel) {
	data.Canonical = types.StringPointerValue(o.Canonical)
	data.Name = types.StringPointerValue(o.Name)
	data.Data = resource_organization.NewDataValueNull()
	data.OrganizationCanonical = types.StringValue(org)
}
