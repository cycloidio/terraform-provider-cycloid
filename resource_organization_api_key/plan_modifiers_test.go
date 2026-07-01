package resource_organization_api_key

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func ruleObjectType() attr.Type {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"action":    types.StringType,
			"effect":    types.StringType,
			"resources": types.ListType{ElemType: types.StringType},
		},
	}
}

func ruleListValue(t *testing.T, action string) types.List {
	t.Helper()

	v, diags := types.ListValueFrom(context.Background(), ruleObjectType(), []RuleModel{
		{
			Action:    types.StringValue(action),
			Effect:    types.StringValue("allow"),
			Resources: types.ListValueMust(types.StringType, nil),
		},
	})
	if diags.HasError() {
		t.Fatalf("failed to build test rules list: %v", diags)
	}
	return v
}

// TestListForceNewPlanModifier_RulesChange proves the claim in the
// TFPRO-50/51-adjacent review comment on organization_api_key_resource.go:
// a rules ("role") change must force replacement rather than flow through
// Update(), since Update() never sends rules to the API at all.
func TestListForceNewPlanModifier_RulesChange(t *testing.T) {
	m := listForceNewPlanModifier{}

	testCases := []struct {
		name            string
		state           types.List
		plan            types.List
		wantReplace     bool
		wantDescription string
	}{
		{
			name:        "create (no prior state) does not require replace",
			state:       types.ListNull(ruleObjectType()),
			plan:        ruleListValue(t, "organization:project:read"),
			wantReplace: false,
		},
		{
			name:        "unchanged rules does not require replace",
			state:       ruleListValue(t, "organization:project:read"),
			plan:        ruleListValue(t, "organization:project:read"),
			wantReplace: false,
		},
		{
			name:        "changed rules requires replace",
			state:       ruleListValue(t, "organization:project:read"),
			plan:        ruleListValue(t, "organization:project:write"),
			wantReplace: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.ListRequest{
				StateValue: tc.state,
				PlanValue:  tc.plan,
			}
			resp := &planmodifier.ListResponse{}

			m.PlanModifyList(context.Background(), req, resp)

			assert.Equal(t, tc.wantReplace, resp.RequiresReplace)
		})
	}
}
