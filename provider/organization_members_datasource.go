package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &organizationMembersDataSource{}

type organizationMembersDataSource struct {
	provider *CycloidProvider
}

func NewOrganizationMembersDataSource() datasource.DataSource {
	return &organizationMembersDataSource{}
}

type organizationMembersDatasourceModel struct {
	Organization types.String `tfsdk:"organization"`
	Members      types.List   `tfsdk:"members"`
}

type orgMemberDatasourceItem struct {
	MemberId        types.Int64  `tfsdk:"member_id"`
	Username        types.String `tfsdk:"username"`
	Email           types.String `tfsdk:"email"`
	FullName        types.String `tfsdk:"full_name"`
	Role            types.String `tfsdk:"role"`
	InvitationState types.String `tfsdk:"invitation_state"`
}

var orgMemberDatasourceItemAttrTypes = map[string]attr.Type{
	"member_id":        types.Int64Type,
	"username":         types.StringType,
	"email":            types.StringType,
	"full_name":        types.StringType,
	"role":             types.StringType,
	"invitation_state": types.StringType,
}

func (d *organizationMembersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_members"
}

func (d *organizationMembersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Lists all members of a Cycloid organization. Member emails are stored in Terraform state — protect your state backend accordingly.",
		MarkdownDescription: "Lists all members of a Cycloid organization. Member emails are stored in Terraform state — protect your state backend accordingly.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Canonical of the organization. Defaults to the provider organization setting.",
				MarkdownDescription: "Canonical of the organization. Defaults to the provider organization setting.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9]+[a-z0-9\\-_]+[a-z0-9]+$"), ""),
				},
			},
			"members": schema.ListNestedAttribute{
				Computed:            true,
				Description:         "List of organization members.",
				MarkdownDescription: "List of organization members.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"member_id": schema.Int64Attribute{
							Computed:            true,
							Description:         "Unique identifier of the member.",
							MarkdownDescription: "Unique identifier of the member.",
						},
						"username": schema.StringAttribute{
							Computed:            true,
							Description:         "Username (canonical) of the member.",
							MarkdownDescription: "Username (canonical) of the member.",
						},
						"email": schema.StringAttribute{
							Computed:            true,
							Description:         "Email address of the member. For pending invitations this is the address the invitation was sent to.",
							MarkdownDescription: "Email address of the member. For pending invitations this is the address the invitation was sent to.",
						},
						"full_name": schema.StringAttribute{
							Computed:            true,
							Description:         "Full name of the member.",
							MarkdownDescription: "Full name of the member.",
						},
						"role": schema.StringAttribute{
							Computed:            true,
							Description:         "Canonical of the role assigned to the member.",
							MarkdownDescription: "Canonical of the role assigned to the member.",
						},
						"invitation_state": schema.StringAttribute{
							Computed:            true,
							Description:         "State of the member invitation: pending, accepted, or declined.",
							MarkdownDescription: "State of the member invitation: `pending`, `accepted`, or `declined`.",
						},
					},
				},
			},
		},
	}
}

func (d *organizationMembersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.provider = pv
}

func (d *organizationMembersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationMembersDatasourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*d.provider, data.Organization)

	// TODO(TFPRO-39): migrate to paginated middleware helper once available.
	members, _, err := d.provider.Middleware.ListMembers(org)
	if err != nil {
		resp.Diagnostics.AddError("failed to list organization members", err.Error())
		return
	}

	listVal, diags := orgMembersToListValue(ctx, members)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Organization = types.StringValue(org)
	data.Members = listVal

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func orgMembersToListValue(ctx context.Context, members []*models.MemberOrg) (types.List, diag.Diagnostics) {
	elemType := types.ObjectType{AttrTypes: orgMemberDatasourceItemAttrTypes}

	if len(members) == 0 {
		return types.ListValueMust(elemType, []attr.Value{}), nil
	}

	items := make([]orgMemberDatasourceItem, 0, len(members))
	for _, m := range members {
		if m == nil {
			continue
		}

		var memberID int64
		if m.ID != nil {
			memberID = int64(*m.ID)
		}

		email := memberEmailForDatasource(m.Email, m.InvitationEmail, m.InvitationState)

		var role string
		if m.Role != nil && m.Role.Canonical != nil {
			role = *m.Role.Canonical
		}

		items = append(items, orgMemberDatasourceItem{
			MemberId:        types.Int64Value(memberID),
			Username:        types.StringValue(m.Username),
			Email:           types.StringValue(email),
			FullName:        types.StringValue(m.FullName),
			Role:            types.StringValue(role),
			InvitationState: types.StringValue(m.InvitationState),
		})
	}

	return types.ListValueFrom(ctx, elemType, items)
}

// memberEmailForDatasource mirrors orgMemberCYModelToData: for pending invites use InvitationEmail.
func memberEmailForDatasource(email, invitationEmail strfmt.Email, invitationState string) string {
	if invitationState == "pending" {
		return string(invitationEmail)
	}
	return string(email)
}
