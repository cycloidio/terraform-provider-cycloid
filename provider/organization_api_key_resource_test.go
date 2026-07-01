package provider

import (
	"context"
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/resource_organization_api_key"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit tests — no TF_ACC required.

func mkRulesList(t *testing.T, ctx context.Context, rules []resource_organization_api_key.RuleModel) types.List {
	t.Helper()
	vals := make([]attr.Value, 0, len(rules))
	for _, r := range rules {
		obj, d := types.ObjectValueFrom(ctx, apiKeyRuleObjectType().(types.ObjectType).AttrTypes, r)
		require.False(t, d.HasError(), "ObjectValueFrom: %v", d)
		vals = append(vals, obj)
	}
	list, d := types.ListValue(apiKeyRuleObjectType(), vals)
	require.False(t, d.HasError(), "ListValue: %v", d)
	return list
}

func apiKeyStrListNull() types.List { return types.ListNull(types.StringType) }

func apiKeyStrList(t *testing.T, ctx context.Context, elems ...string) types.List {
	t.Helper()
	if elems == nil {
		elems = []string{}
	}
	l, d := types.ListValueFrom(ctx, types.StringType, elems)
	require.False(t, d.HasError(), "ListValueFrom: %v", d)
	return l
}

func TestAPIKeyCYModelToData_WithToken(t *testing.T) {
	canonical := "my-api-key"
	name := "My API Key"
	lastSeven := "abc1234"
	id := uint32(42)
	username := "john-doe"
	action := "organization:read"
	effect := "allow"
	ruleID := uint32(1)

	apiKey := &models.APIKey{
		Canonical: &canonical,
		Name:      &name,
		LastSeven: &lastSeven,
		ID:        &id,
		Token:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
		Owner: &models.User{
			Username: &username,
		},
		Rules: []*models.Rule{
			{Action: &action, Effect: &effect, ID: &ruleID},
		},
	}

	var data organizationAPIKeyResourceModel
	ctx := context.Background()

	diags := apiKeyCYModelToData(ctx, "my-org", apiKey, &data)

	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	assert.Equal(t, "my-org", data.OrganizationCanonical.ValueString())
	assert.Equal(t, "my-api-key", data.Canonical.ValueString())
	assert.Equal(t, "My API Key", data.Name.ValueString())
	assert.Equal(t, "abc1234", data.LastSeven.ValueString())
	assert.Equal(t, int64(42), data.ID.ValueInt64())
	assert.Equal(t, "john-doe", data.Owner.ValueString())
	assert.Equal(t, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test", data.Token.ValueString())
	assert.False(t, data.Rules.IsNull())
}

func TestAPIKeyCYModelToData_NoToken_PreservesState(t *testing.T) {
	canonical := "my-api-key"
	name := "My API Key"
	lastSeven := "abc1234"
	id := uint32(42)

	apiKey := &models.APIKey{
		Canonical: &canonical,
		Name:      &name,
		LastSeven: &lastSeven,
		ID:        &id,
		Token:     "", // empty — as returned on GET (not create)
		Rules:     []*models.Rule{},
	}

	// Simulate prior state having a token already stored.
	var data organizationAPIKeyResourceModel
	data.Token = types.StringValue("previously-stored-token")

	ctx := context.Background()

	diags := apiKeyCYModelToData(ctx, "my-org", apiKey, &data)

	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	// Token must not be overwritten when the API returns empty.
	assert.Equal(t, "previously-stored-token", data.Token.ValueString())
}

func TestAPIKeyCYModelToData_NoOwner(t *testing.T) {
	canonical := "key"
	name := "Key"
	lastSeven := "xyz7890"
	id := uint32(1)

	apiKey := &models.APIKey{
		Canonical: &canonical,
		Name:      &name,
		LastSeven: &lastSeven,
		ID:        &id,
		Owner:     nil,
		Rules:     []*models.Rule{},
	}

	var data organizationAPIKeyResourceModel
	data.Owner = types.StringNull()

	ctx := context.Background()

	diags := apiKeyCYModelToData(ctx, "org", apiKey, &data)

	require.False(t, diags.HasError())
	assert.Equal(t, "", data.Owner.ValueString())
}

func TestDataToNewRules_Empty(t *testing.T) {
	ctx := context.Background()

	ruleObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"action":    types.StringType,
			"effect":    types.StringType,
			"resources": types.ListType{ElemType: types.StringType},
		},
	}

	rulesVal, diags := types.ListValueFrom(ctx, ruleObjType, []interface{}{})
	require.False(t, diags.HasError())

	rules, rDiags := dataToNewRules(ctx, rulesVal)
	require.False(t, rDiags.HasError())
	assert.Empty(t, rules)
}

// TestDataToNewRules_ResourcesAlwaysExplicit is the organization_api_key
// counterpart of TestOrganizationRolePlanRulesToCYModel_ResourcesAlwaysExplicit
// (TFPRO-42): omitted or explicit-[] resources must both send an explicit []
// to the API, never a nil/JSON-null Resources field.
func TestDataToNewRules_ResourcesAlwaysExplicit(t *testing.T) {
	ctx := context.Background()

	rulesVal := mkRulesList(t, ctx, []resource_organization_api_key.RuleModel{
		{Action: types.StringValue("organization:project:read"), Effect: types.StringValue("allow"), Resources: apiKeyStrListNull()},
		{Action: types.StringValue("organization:team:read"), Effect: types.StringValue("allow"), Resources: apiKeyStrList(t, ctx)},
		{Action: types.StringValue("organization:credential:read"), Effect: types.StringValue("allow"), Resources: apiKeyStrList(t, ctx, "organization:myorg:project:myproject")},
	})

	cyRules, diags := dataToNewRules(ctx, rulesVal)
	require.False(t, diags.HasError(), "diags: %v", diags)
	require.Len(t, cyRules, 3)

	byAction := map[string]*models.NewRule{}
	for _, r := range cyRules {
		require.NotNil(t, r.Resources, "Resources must never be nil — an explicit [] is required to clear API state")
		byAction[*r.Action] = r
	}

	omitted := byAction["organization:project:read"]
	require.NotNil(t, omitted)
	assert.Equal(t, []string{}, omitted.Resources)

	empty := byAction["organization:team:read"]
	require.NotNil(t, empty)
	assert.Equal(t, []string{}, empty.Resources)

	scoped := byAction["organization:credential:read"]
	require.NotNil(t, scoped)
	assert.Equal(t, []string{"organization:myorg:project:myproject"}, scoped.Resources)
}

// TestAPIKeyCYModelToData_PreservesEmptyRepresentation is the
// organization_api_key counterpart of
// TestOrganizationRoleCYModelToData_PreservesEmptyRepresentation (TFPRO-42):
// when the API reports a rule has no resources, the state must keep whichever
// null-vs-[] form the user configured, not collapse both to the same thing.
func TestAPIKeyCYModelToData_PreservesEmptyRepresentation(t *testing.T) {
	ctx := context.Background()

	data := organizationAPIKeyResourceModel{
		Canonical: types.StringValue("my-api-key"),
		Rules: mkRulesList(t, ctx, []resource_organization_api_key.RuleModel{
			{Action: types.StringValue("act:omitted"), Effect: types.StringValue("allow"), Resources: apiKeyStrListNull()},
			{Action: types.StringValue("act:empty"), Effect: types.StringValue("allow"), Resources: apiKeyStrList(t, ctx)},
			{Action: types.StringValue("act:scoped"), Effect: types.StringValue("allow"), Resources: apiKeyStrList(t, ctx, "organization:o:project:p")},
		}),
	}

	apiKey := &models.APIKey{
		Canonical: strPtr("my-api-key"),
		Name:      strPtr("My API Key"),
		Rules: []*models.Rule{
			{Action: strPtr("act:omitted"), Effect: strPtr("allow"), Resources: nil},
			{Action: strPtr("act:empty"), Effect: strPtr("allow"), Resources: []string{}},
			{Action: strPtr("act:scoped"), Effect: strPtr("allow"), Resources: []string{"organization:o:project:p"}},
		},
	}

	diags := apiKeyCYModelToData(ctx, "my-org", apiKey, &data)
	require.False(t, diags.HasError(), "diags: %v", diags)

	var stateRules []resource_organization_api_key.RuleModel
	diags = data.Rules.ElementsAs(ctx, &stateRules, false)
	require.False(t, diags.HasError(), "ElementsAs: %v", diags)

	byAction := map[string]resource_organization_api_key.RuleModel{}
	for _, r := range stateRules {
		byAction[r.Action.ValueString()] = r
	}

	assert.True(t, byAction["act:omitted"].Resources.IsNull(),
		"omitted resources must remain null to match config")

	emptyRes := byAction["act:empty"].Resources
	assert.False(t, emptyRes.IsNull(), "explicit [] must remain a non-null empty list")
	assert.Len(t, emptyRes.Elements(), 0)

	assert.Len(t, byAction["act:scoped"].Resources.Elements(), 1)
}
