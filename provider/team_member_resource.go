package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_team_member"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &teamMemberResource{}

type teamMemberResourceModel resource_team_member.TeamMemberModel

func NewTeamMemberResource() resource.Resource {
	return &teamMemberResource{}
}

type teamMemberResource struct {
	provider *CycloidProvider
}

func (r *teamMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_member"
}

func (r *teamMemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_team_member.TeamMemberResourcesSchema(ctx)
}

func (r *teamMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *teamMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var teamMemberState teamMemberResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &teamMemberState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Middleware

	org := getOrganizationCanonical(*r.provider, teamMemberState.Organization)
	team := teamMemberState.Team.ValueString()
	username := teamMemberState.Username.ValueString()
	email := teamMemberState.Email.ValueString()

	teamMembers, err := m.ListTeamMembers(org, team)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list current team_member in org %q", org), err.Error())
		return
	}

	var teamMember *models.MemberTeam
	for _, tm := range teamMembers {
		if tm.Username == username || ptr.Value(teamMember.Email).String() == email {
			teamMember = tm
		}
	}

	resp.Diagnostics.Append(
		TeamMemberToModel(ctx, org, team, teamMember, &teamMemberState)...,
	)
}

func (r *teamMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var teamMemberState teamMemberResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &teamMemberState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Middleware

	org := getOrganizationCanonical(*r.provider, teamMemberState.Organization)
	team := teamMemberState.Team.ValueString()
	username := teamMemberState.Username.ValueString()
	email := teamMemberState.Email.ValueString()

	// Resource is idempotent, so we check if current team_member exists to decide if we
	// create or update
	teamMembers, err := m.ListTeamMembers(org, team)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list current team_member in org %q", org), err.Error())
		return
	}

	var teamMember *models.MemberTeam
	for _, tm := range teamMembers {
		if tm.Username == username || ptr.Value(teamMember.Email).String() == email {
			teamMember = tm
		}
	}

	if teamMember == nil {
		teamMember, err = m.AssignMemberToTeam(org, team, &username, &email)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to assign team member %q to team %q in org %q", Coalesce(username, email), team, org), err.Error())
			return
		}
	}

	resp.Diagnostics.Append(
		TeamMemberToModel(ctx, org, team, teamMember, &teamMemberState)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &teamMemberState)...)
}

func (r *teamMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var teamMemberState teamMemberResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &teamMemberState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Middleware

	org := getOrganizationCanonical(*r.provider, teamMemberState.Organization)
	team := teamMemberState.Team.ValueString()
	username := teamMemberState.Username.ValueString()
	email := teamMemberState.Email.ValueString()

	// Resource is idempotent, so we check if current team_member exists to decide if we
	// create or update
	teamMembers, err := m.ListTeamMembers(org, team)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list current team_member in org %q", org), err.Error())
		return
	}

	var teamMember *models.MemberTeam
	for _, tm := range teamMembers {
		if tm.Username == username || ptr.Value(teamMember.Email).String() == email {
			teamMember = tm
		}
	}

	if teamMember == nil {
		teamMember, err = m.AssignMemberToTeam(org, team, &username, &email)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to assign team member %q to team %q in org %q", Coalesce(username, email), team, org), err.Error())
			return
		}
	}

	resp.Diagnostics.Append(
		TeamMemberToModel(ctx, org, team, teamMember, &teamMemberState)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &teamMemberState)...)
}

func (r *teamMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var teamMemberState teamMemberResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &teamMemberState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Middleware

	org := getOrganizationCanonical(*r.provider, teamMemberState.Organization)
	team := teamMemberState.Team.ValueString()
	username := teamMemberState.Username.ValueString()
	email := teamMemberState.Email.ValueString()

	// We need to check if the team_member exists before delete
	teamMembers, err := m.ListTeamMembers(org, team)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list current team_member in org %q", org), err.Error())
		return
	}

	var teamMember *models.MemberTeam
	for _, tm := range teamMembers {
		if tm.Username == username || ptr.Value(teamMember.Email).String() == email {
			teamMember = tm
		}
	}

	if teamMember != nil {
		err := m.UnAssignMemberFromTeam(org, team, ptr.Value(teamMember.ID))
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to unassign member %q from team %q in org %q", Coalesce(username, email), team, org), err.Error())
			return
		}
	}

	resp.Diagnostics.Append(
		TeamMemberToModel(ctx, org, team, &models.MemberTeam{}, &teamMemberState)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &teamMemberState)...)
}

func TeamMemberToModel(ctx context.Context, org, team string, teamMember *models.MemberTeam, teamMemberState *teamMemberResourceModel) diag.Diagnostics {
	if teamMember == nil {
		teamMemberState.Username = types.StringNull()
		teamMemberState.Email = types.StringNull()
		teamMemberState.Organization = types.StringNull()
		teamMemberState.Team = types.StringNull()
	} else {
		teamMemberState.Username = types.StringPointerValue(&teamMember.Username)
		teamMemberState.Email = types.StringValue(ptr.Value(teamMember.Email).String())
		teamMemberState.Organization = types.StringValue(org)
		teamMemberState.Team = types.StringValue(team)
	}
	return nil
}
