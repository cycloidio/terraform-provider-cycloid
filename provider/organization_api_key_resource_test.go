package provider

import (
	"context"
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit tests — no TF_ACC required.

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
