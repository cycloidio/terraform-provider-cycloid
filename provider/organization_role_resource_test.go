package provider

import (
	"context"
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/resource_organization_role"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }

// mkRulesSet builds a rules Set from a slice of rule models, mirroring how the
// Terraform framework would populate the plan/state.
func mkRulesSet(t *testing.T, ctx context.Context, rules []organizationRoleRuleResourceModel) types.Set {
	t.Helper()
	ruleObjectType := types.ObjectType{AttrTypes: resource_organization_role.OrganizationRoleRuleTypes}
	vals := make([]attr.Value, 0, len(rules))
	for _, r := range rules {
		obj, d := types.ObjectValueFrom(ctx, resource_organization_role.OrganizationRoleRuleTypes, r)
		require.False(t, d.HasError(), "ObjectValueFrom: %v", d)
		vals = append(vals, obj)
	}
	set, d := types.SetValue(ruleObjectType, vals)
	require.False(t, d.HasError(), "SetValue: %v", d)
	return set
}

func strListNull() types.List { return types.ListNull(types.StringType) }

func strList(t *testing.T, ctx context.Context, elems ...string) types.List {
	t.Helper()
	if elems == nil {
		// A zero-arg variadic call yields a nil slice, which reflects to a null
		// list. `resources = []` in HCL is a non-null empty list, so force that.
		elems = []string{}
	}
	l, d := types.ListValueFrom(ctx, types.StringType, elems)
	require.False(t, d.HasError(), "ListValueFrom: %v", d)
	return l
}

// TestOrganizationRolePlanRulesToCYModel_ResourcesAlwaysExplicit is the core
// regression for TFPRO-42: whether the user omits `resources` (null) or sets it
// to `[]`, the request body sent to the API must carry an explicit empty list so
// the API clears any previously scoped resources rather than treating the field
// as "no change".
func TestOrganizationRolePlanRulesToCYModel_ResourcesAlwaysExplicit(t *testing.T) {
	ctx := context.Background()

	rules := mkRulesSet(t, ctx, []organizationRoleRuleResourceModel{
		{Action: types.StringValue("organization:project:*"), Effect: types.StringNull(), Resources: strListNull()},
		{Action: types.StringValue("organization:team:*"), Effect: types.StringValue("allow"), Resources: strList(t, ctx)},
		{Action: types.StringValue("organization:credential:read"), Effect: types.StringNull(), Resources: strList(t, ctx, "organization:myorg:project:myproject")},
	})

	cyRules, diags := organizationRolePlanRulesToCYModel(ctx, rules)
	require.False(t, diags.HasError(), "diags: %v", diags)
	require.Len(t, cyRules, 3)

	byAction := map[string]*models.NewRule{}
	for _, r := range cyRules {
		require.NotNil(t, r.Resources, "Resources must never be nil — an explicit [] is required to clear API state")
		byAction[*r.Action] = r
	}

	// Omitted (null) resources -> explicit empty list, effect defaulted to allow.
	omitted := byAction["organization:project:*"]
	require.NotNil(t, omitted)
	require.Equal(t, []string{}, omitted.Resources)
	require.Equal(t, "allow", *omitted.Effect)

	// Explicit [] resources -> explicit empty list.
	empty := byAction["organization:team:*"]
	require.NotNil(t, empty)
	require.Equal(t, []string{}, empty.Resources)

	// Populated resources -> passed through unchanged.
	scoped := byAction["organization:credential:read"]
	require.NotNil(t, scoped)
	require.Equal(t, []string{"organization:myorg:project:myproject"}, scoped.Resources)
}

// TestOrganizationRoleCYModelToData_PreservesEmptyRepresentation verifies that
// because `resources` is Optional (not Computed), the state written back keeps
// the exact null-vs-[] form the user configured when the API reports the rule
// has no scoped resources. Otherwise Terraform raises "provider produced
// inconsistent result after apply".
func TestOrganizationRoleCYModelToData_PreservesEmptyRepresentation(t *testing.T) {
	ctx := context.Background()

	plan := organizationRoleResourceModel{
		Canonical: types.StringValue("my-role"),
		Rules: mkRulesSet(t, ctx, []organizationRoleRuleResourceModel{
			{Action: types.StringValue("act:omitted"), Effect: types.StringValue("allow"), Resources: strListNull()},
			{Action: types.StringValue("act:empty"), Effect: types.StringValue("allow"), Resources: strList(t, ctx)},
			{Action: types.StringValue("act:scoped"), Effect: types.StringValue("allow"), Resources: strList(t, ctx, "organization:o:project:p")},
		}),
	}

	role := &models.Role{
		Name:      strPtr("My Role"),
		Canonical: strPtr("my-role"),
		Rules: []*models.Rule{
			{Action: strPtr("act:omitted"), Effect: strPtr("allow"), Resources: nil},
			{Action: strPtr("act:empty"), Effect: strPtr("allow"), Resources: []string{}},
			{Action: strPtr("act:scoped"), Effect: strPtr("allow"), Resources: []string{"organization:o:project:p"}},
		},
	}

	diags := organizationRoleCYModelToData(ctx, "my-org", role, &plan)
	require.False(t, diags.HasError(), "diags: %v", diags)

	var stateRules []organizationRoleRuleResourceModel
	diags = plan.Rules.ElementsAs(ctx, &stateRules, false)
	require.False(t, diags.HasError(), "ElementsAs: %v", diags)

	byAction := map[string]organizationRoleRuleResourceModel{}
	for _, r := range stateRules {
		byAction[r.Action.ValueString()] = r
	}

	// Omitted in config + empty on API -> stays null.
	require.True(t, byAction["act:omitted"].Resources.IsNull(),
		"omitted resources must remain null to match config")

	// Explicit [] in config + empty on API -> stays an empty (non-null) list.
	emptyRes := byAction["act:empty"].Resources
	require.False(t, emptyRes.IsNull(), "explicit [] must remain a non-null empty list")
	require.Len(t, emptyRes.Elements(), 0)

	// Scoped resources on API -> surfaced regardless of prior representation.
	require.Len(t, byAction["act:scoped"].Resources.Elements(), 1)
}
