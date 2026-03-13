package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/cycloid-cli/client/client/organization_roles"
	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_organization_role"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &organizationRoleResource{}

type organizationRoleResourceModel resource_organization_role.OrganizationRoleModel
type organizationRoleRuleResourceModel resource_organization_role.OrganizationRoleRuleModel

func NewOrganizationRoleResource() resource.Resource {
	return &organizationRoleResource{}
}

type organizationRoleResource struct {
	provider *CycloidProvider
}

func (r *organizationRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_role"
}

func (r *organizationRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_organization_role.OrganizationRoleResourceSchema(ctx)
}

func (r *organizationRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *organizationRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var rolePlan organizationRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &rolePlan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Middleware
	org := getOrganizationCanonical(*r.provider, rolePlan.Organization)
	name, canonical, err := NameOrCanonical(rolePlan.Name.ValueString(), rolePlan.Canonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to infer role canonical", err.Error())
		return
	}

	rules, diags := organizationRolePlanRulesToCYModel(ctx, rolePlan.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := m.ListRoles(org)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list roles in org %q", org), err.Error())
		return
	}

	var existingRole *models.Role
	for _, role := range roles {
		if ptr.Value(role.Canonical) == canonical {
			existingRole = role
			break
		}
	}

	description := rolePlan.Description.ValueStringPointer()
	if existingRole == nil {
		_, err = m.CreateRole(org, &name, &canonical, description, rules)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to create role %q in org %q", canonical, org), err.Error())
			return
		}
	} else {
		_, err = r.updateRole(ctx, org, canonical, &name, &canonical, description, rules)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to update existing role %q in org %q", canonical, org), err.Error())
			return
		}
	}

	role, err := m.GetRole(org, canonical)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to read created role %q in org %q", canonical, org), err.Error())
		return
	}

	resp.Diagnostics.Append(organizationRoleCYModelToData(ctx, org, role, &rolePlan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &rolePlan)...)
}

func (r *organizationRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var roleState organizationRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &roleState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Middleware
	org := getOrganizationCanonical(*r.provider, roleState.Organization)
	_, canonical, err := NameOrCanonical(roleState.Name.ValueString(), roleState.Canonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to infer role canonical", err.Error())
		return
	}

	role, err := m.GetRole(org, canonical)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to read role %q in org %q", canonical, org), err.Error())
		return
	}

	resp.Diagnostics.Append(organizationRoleCYModelToData(ctx, org, role, &roleState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &roleState)...)
}

func (r *organizationRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var rolePlan organizationRoleResourceModel
	var roleState organizationRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &roleState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &rolePlan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, rolePlan.Organization)
	name, canonical, err := NameOrCanonical(rolePlan.Name.ValueString(), rolePlan.Canonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to infer role canonical", err.Error())
		return
	}

	_, currentCanonical, err := NameOrCanonical(roleState.Name.ValueString(), roleState.Canonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to infer current role canonical", err.Error())
		return
	}

	rules, diags := organizationRolePlanRulesToCYModel(ctx, rolePlan.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.updateRole(
		ctx,
		org,
		currentCanonical,
		&name,
		&canonical,
		rolePlan.Description.ValueStringPointer(),
		rules,
	)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to update role %q in org %q", currentCanonical, org), err.Error())
		return
	}

	resp.Diagnostics.Append(organizationRoleCYModelToData(ctx, org, role, &rolePlan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &rolePlan)...)
}

func (r *organizationRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var roleState organizationRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &roleState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Middleware
	org := getOrganizationCanonical(*r.provider, roleState.Organization)
	_, canonical, err := NameOrCanonical(roleState.Name.ValueString(), roleState.Canonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to infer role canonical", err.Error())
		return
	}

	err = m.DeleteRole(org, canonical)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to delete role %q in org %q", canonical, org), err.Error())
		return
	}

	resp.Diagnostics.Append(organizationRoleCYModelToData(ctx, org, nil, &roleState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &roleState)...)
}

func (r *organizationRoleResource) updateRole(
	ctx context.Context,
	org string,
	currentCanonical string,
	name *string,
	canonical *string,
	description *string,
	rules []*models.NewRule,
) (*models.Role, error) {
	updateParams := organization_roles.NewUpdateRoleParamsWithContext(ctx)
	updateParams.SetOrganizationCanonical(org)
	updateParams.SetRoleCanonical(currentCanonical)

	descriptionValue := ptr.Value(description)
	canonicalValue := ptr.Value(canonical)
	nameValue := ptr.Value(name)

	updateParams.SetBody(&models.NewRole{
		Name:        &nameValue,
		Canonical:   canonicalValue,
		Description: descriptionValue,
		Rules:       rules,
	})

	updateResp, err := r.provider.APIClient.OrganizationRoles.UpdateRole(updateParams, r.provider.APIClient.Credentials(&org))
	if err != nil {
		return nil, err
	}

	if updateResp == nil || updateResp.Payload == nil || updateResp.Payload.Data == nil {
		return nil, fmt.Errorf("organization role update returned an empty response")
	}

	return updateResp.Payload.Data, nil
}

func organizationRolePlanRulesToCYModel(ctx context.Context, rulesState types.List) ([]*models.NewRule, diag.Diagnostics) {
	var diags diag.Diagnostics

	if rulesState.IsNull() || rulesState.IsUnknown() {
		diags.AddError("invalid role rules", "rules must be provided")
		return nil, diags
	}

	var stateRules []organizationRoleRuleResourceModel
	diags.Append(rulesState.ElementsAs(ctx, &stateRules, false)...)
	if diags.HasError() {
		return nil, diags
	}

	cyRules := make([]*models.NewRule, 0, len(stateRules))
	for _, stateRule := range stateRules {
		action := stateRule.Action.ValueString()
		effect := stateRule.Effect.ValueString()
		if effect == "" {
			effect = "allow"
		}

		resources := []string{}
		if !stateRule.Resources.IsNull() && !stateRule.Resources.IsUnknown() {
			diags.Append(stateRule.Resources.ElementsAs(ctx, &resources, false)...)
			if diags.HasError() {
				return nil, diags
			}
		}

		cyRules = append(cyRules, &models.NewRule{
			Action:    &action,
			Effect:    &effect,
			Resources: resources,
		})
	}

	return cyRules, diags
}

func organizationRoleCYModelToData(ctx context.Context, org string, role *models.Role, roleState *organizationRoleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	ruleObjectType := types.ObjectType{AttrTypes: resource_organization_role.OrganizationRoleRuleTypes}
	if role == nil {
		roleState.Name = types.StringNull()
		roleState.Canonical = types.StringNull()
		roleState.Organization = types.StringValue(org)
		roleState.Description = types.StringNull()
		roleState.ID = types.Int64Null()
		roleState.Default = types.BoolNull()
		roleState.Rules = types.ListNull(ruleObjectType)
		return diags
	}

	ruleValues := make([]attr.Value, 0, len(role.Rules))
	for _, roleRule := range role.Rules {
		if roleRule == nil {
			continue
		}

		resourcesList, resourcesDiags := types.ListValueFrom(ctx, types.StringType, roleRule.Resources)
		diags.Append(resourcesDiags...)
		if diags.HasError() {
			return diags
		}

		effect := ptr.Value(roleRule.Effect)
		if effect == "" {
			effect = "allow"
		}

		ruleValue, ruleDiags := types.ObjectValue(
			resource_organization_role.OrganizationRoleRuleTypes,
			map[string]attr.Value{
				"action":    types.StringValue(ptr.Value(roleRule.Action)),
				"effect":    types.StringValue(effect),
				"resources": resourcesList,
			},
		)
		diags.Append(ruleDiags...)
		if diags.HasError() {
			return diags
		}

		ruleValues = append(ruleValues, ruleValue)
	}

	rulesList, rulesDiags := types.ListValue(ruleObjectType, ruleValues)
	diags.Append(rulesDiags...)
	if diags.HasError() {
		return diags
	}

	roleState.Name = types.StringPointerValue(role.Name)
	roleState.Canonical = types.StringPointerValue(role.Canonical)
	roleState.Organization = types.StringValue(org)
	roleState.Description = types.StringPointerValue(role.Description)
	roleState.ID = types.Int64Value(int64(ptr.Value(role.ID)))
	roleState.Default = types.BoolValue(ptr.Value(role.Default))
	roleState.Rules = rulesList

	return diags
}
