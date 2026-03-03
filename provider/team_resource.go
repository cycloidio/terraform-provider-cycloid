package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_team"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &teamResource{}

type teamResourceModel resource_team.TeamModel

func NewTeamResource() resource.Resource {
	return &teamResource{}
}

type teamResource struct {
	provider *CycloidProvider
}

func (r *teamResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (r *teamResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_team.TeamResourcesSchema(ctx)
}

func (r *teamResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *teamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var teamState teamResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &teamState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Middleware

	org := getOrganizationCanonical(*r.provider, teamState.Organization)
	name, canonical, err := NameOrCanonical(teamState.Name.ValueString(), teamState.Canonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to infer canonical", err.Error())
		return
	}

	teams, err := m.ListTeams(org, &name, nil, nil, nil)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list current team in org %q", org), err.Error())
		return
	}

	var team *models.Team
	for _, t := range teams {
		if ptr.Value(t.Canonical) == canonical {
			team = t
		}
	}

	resp.Diagnostics.Append(
		TeamToModel(ctx, org, team, &teamState)...,
	)
}

func (r *teamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var teamState teamResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &teamState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Middleware

	org := getOrganizationCanonical(*r.provider, teamState.Organization)
	name, canonical, err := NameOrCanonical(teamState.Name.ValueString(), teamState.Canonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to infer canonical", err.Error())
		return
	}

	// Resource is idempotent, so we check if current team exists to decide if we
	// create or update
	teams, err := m.ListTeams(org, &name, nil, nil, nil)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list current team in org %q", org), err.Error())
		return
	}

	var team *models.Team
	for _, t := range teams {
		if ptr.Value(t.Canonical) == canonical {
			team = t
		}
	}

	var roles = []string{}
	resp.Diagnostics.Append(teamState.Roles.ElementsAs(ctx, &roles, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if team == nil {
		team, err = m.CreateTeam(org, &name, &canonical, teamState.Owner.ValueStringPointer(), roles)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to create team %q in org %q", canonical, org), err.Error())
			return
		}
	} else {
		team, err = m.UpdateTeam(org, &name, &canonical, teamState.Owner.ValueStringPointer(), roles)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to update existing team %q in org %q", canonical, org), err.Error())
			return
		}
	}

	resp.Diagnostics.Append(
		TeamToModel(ctx, org, team, &teamState)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, teamState)...)
}

func (r *teamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var teamState teamResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &teamState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Middleware

	org := getOrganizationCanonical(*r.provider, teamState.Organization)
	name, canonical, err := NameOrCanonical(teamState.Name.ValueString(), teamState.Canonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to infer canonical", err.Error())
		return
	}

	// Resource is idempotent, so we check if current team exists to decide if we
	// create or update
	teams, err := m.ListTeams(org, &name, nil, nil, nil)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list current team in org %q", org), err.Error())
		return
	}

	var team *models.Team
	for _, t := range teams {
		if ptr.Value(t.Canonical) == canonical {
			team = t
		}
	}

	var roles = []string{}
	resp.Diagnostics.Append(teamState.Roles.ElementsAs(ctx, &roles, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if team == nil {
		team, err = m.CreateTeam(org, &name, &canonical, teamState.Owner.ValueStringPointer(), roles)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to create team %q in org %q", canonical, org), err.Error())
			return
		}
	} else {
		team, err = m.UpdateTeam(org, &name, &canonical, teamState.Owner.ValueStringPointer(), roles)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to update existing team %q in org %q", canonical, org), err.Error())
			return
		}
	}

	resp.Diagnostics.Append(
		TeamToModel(ctx, org, team, &teamState)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, teamState)...)
}

func (r *teamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var teamState teamResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &teamState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Middleware

	org := getOrganizationCanonical(*r.provider, teamState.Organization)
	_, canonical, err := NameOrCanonical(teamState.Name.ValueString(), teamState.Canonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to infer canonical", err.Error())
		return
	}

	// We need to check if the team exists before delete
	teams, err := m.ListTeams(org, nil, nil, nil, nil)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list current team in org %q", org), err.Error())
		return
	}

	var team *models.Team
	for _, t := range teams {
		if ptr.Value(t.Canonical) == canonical {
			team = t
		}
	}

	if team != nil {
		err = m.DeleteTeam(org, canonical)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("failed to delete team %q in org %q", canonical, org), err.Error(),
			)
			return
		}
	}

	resp.Diagnostics.Append(
		TeamToModel(ctx, org, &models.Team{}, &teamState)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, teamState)...)
}

func TeamToModel(ctx context.Context, org string, team *models.Team, teamState *teamResourceModel) diag.Diagnostics {
	if team == nil {
		teamState.Name = types.StringNull()
		teamState.Canonical = types.StringNull()
		teamState.Owner = types.StringNull()
		teamState.Organization = types.StringValue(org)
		teamState.Roles = types.ListNull(types.StringType)
	} else {
		teamState.Name = types.StringPointerValue(team.Name)
		teamState.Canonical = types.StringPointerValue(team.Canonical)
		teamState.Owner = types.StringPointerValue(ptr.Value(team.Owner).Username)
		teamState.Organization = types.StringValue(org)

		var roles = make([]string, len(team.Roles))
		for i, role := range team.Roles {
			roles[i] = ptr.Value(role.Canonical)
		}
		roleValues, diags := types.ListValueFrom(ctx, types.StringType, roles)
		if diags.HasError() {
			return diags
		}

		teamState.Roles = roleValues
	}

	return nil
}
