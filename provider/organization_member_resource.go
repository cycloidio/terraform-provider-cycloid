package provider

import (
	"context"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/common"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
	"github.com/cycloidio/terraform-provider-cycloid/resource_organization_member"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*organizationMemberResource)(nil)

func NewOrganizationMemberResource() resource.Resource {
	return &organizationMemberResource{}
}

type organizationMemberResource struct {
	provider provider_cycloid.CycloidModel
}

type organizationMemberResourceModel resource_organization_member.OrganizationMemberModel

func (r *organizationMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_member"
}

func (r *organizationMemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_organization_member.OrganizationMemberResourceSchema(ctx)
}

func (r *organizationMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
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

func (r *organizationMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationMemberResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	email := data.Email.ValueString()
	role := data.RoleCanonical.ValueString()

	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)

	m, err := mid.InviteMember(orgCan, email, role)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable create member",
			err.Error(),
		)
		return
	}

	orgMemberCYModelToData(orgCan, m, &data)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func orgMemberCYModelToData(org string, m *models.MemberOrg, data *organizationMemberResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	email := m.Email
	if m.InvitationState == "pending" {
		email = m.InvitationEmail
	}

	data.OrganizationCanonical = types.StringValue(org)
	data.MemberId = types.Int64Value(int64(*m.ID))
	data.Email = types.StringValue(string(email))
	data.RoleCanonical = types.StringValue(*m.Role.Canonical)

	return diags
}

func (r *organizationMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationMemberResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	memberID := data.MemberId
	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)

	m, err := mid.GetMember(orgCan, uint32(memberID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable read member",
			err.Error(),
		)
		return
	}

	orgMemberCYModelToData(orgCan, m, &data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data organizationMemberResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)
	memberID := data.MemberId.ValueInt64()
	roleCan := data.RoleCanonical.ValueString()

	// If there's no memberID, try to get it from the prior state
	if memberID == 0 {
		var plandata organizationMemberResourceModel
		// Read Terraform prior state data into the model
		resp.Diagnostics.Append(req.State.Get(ctx, &plandata)...)
		if resp.Diagnostics.HasError() {
			return
		}
		memberID = plandata.MemberId.ValueInt64()
	}

	m, err := mid.UpdateMember(orgCan, uint32(memberID), roleCan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable update member",
			err.Error(),
		)
		return
	}

	orgMemberCYModelToData(orgCan, m, &data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationMemberResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	memberID := data.MemberId.ValueInt64()
	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)

	err := mid.DeleteMember(orgCan, uint32(memberID))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable delete member",
			err.Error(),
		)
		return
	}
}
