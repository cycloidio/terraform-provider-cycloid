package provider

import (
	"math/big"
	"testing"

	"github.com/cycloidio/terraform-provider-cycloid/internal/dynamic"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestDynamicValueToVariablesPreservesAllGroupKeys(t *testing.T) {
	input := map[string]map[string]map[string]any{
		"Definition of the development project": {
			"Users settings": {
				"project_owners_raw":      "owner@example.com",
				"project_maintainers_raw": "maintainer@example.com",
				"project_developers_raw":  "developer@example.com",
			},
		},
	}

	inputDynamic, diags := dynamic.AnyToDynamicValue(t.Context(), input)
	if diags.HasError() {
		t.Fatalf("failed to build dynamic input: %v", diags)
	}

	output, diags := dynamicValueToVariables(t.Context(), inputDynamic)
	if diags.HasError() {
		t.Fatalf("failed to convert dynamic to variables: %v", diags)
	}

	assert.Equal(t, input, output, "all nested input_variables keys must be preserved")
}

func TestGetInputVariablesForReadAppliesAPIDriftOnOverlappingKeys(t *testing.T) {
	stateInputVariables := map[string]map[string]map[string]any{
		"Definition of the development project": {
			"Users settings": {
				"project_owners_raw":      "owner@example.com",
				"project_maintainers_raw": "maintainer@example.com",
				"project_developers_raw":  "developer@example.com",
			},
		},
	}

	stateInputDynamic, diags := dynamic.AnyToDynamicValue(t.Context(), stateInputVariables)
	if diags.HasError() {
		t.Fatalf("failed to build state dynamic input: %v", diags)
	}

	componentState := componentResourceModel{
		InputVariables: stateInputDynamic,
	}

	backendCurrentConfig := map[string]map[string]map[string]any{
		"Definition of the development project": {
			"Users settings": {
				"project_owners_raw": "owner-from-backend@example.com",
			},
		},
	}

	want := map[string]map[string]map[string]any{
		"Definition of the development project": {
			"Users settings": {
				"project_owners_raw":      "owner-from-backend@example.com",
				"project_maintainers_raw": "maintainer@example.com",
				"project_developers_raw":  "developer@example.com",
			},
		},
	}

	output, diags := getInputVariablesForRead(t.Context(), componentState, backendCurrentConfig)
	if diags.HasError() {
		t.Fatalf("failed to get input variables for read: %v", diags)
	}

	assert.Equal(t, want, output)
}

func TestGetInputVariablesForReadPreservesStateWhenOverlappingValuesMatchAPI(t *testing.T) {
	stateInputVariables := map[string]map[string]map[string]any{
		"s": {
			"g": {"k": "same"},
		},
	}
	stateInputDynamic, diags := dynamic.AnyToDynamicValue(t.Context(), stateInputVariables)
	if diags.HasError() {
		t.Fatalf("failed to build state dynamic: %v", diags)
	}
	api := map[string]map[string]map[string]any{
		"s": {
			"g": {"k": "same", "extra": "only-in-api"},
		},
	}
	output, diags := getInputVariablesForRead(t.Context(), componentResourceModel{InputVariables: stateInputDynamic}, api)
	if diags.HasError() {
		t.Fatalf("failed: %v", diags)
	}
	assert.Equal(t, stateInputVariables, output)
}

func TestDynamicValueToVariablesConvertsNumberValues(t *testing.T) {
	inputDynamic := types.DynamicValue(types.ObjectValueMust(
		map[string]attr.Type{
			"section": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"group": types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"integer_key": types.NumberType,
							"float_key":   types.NumberType,
						},
					},
				},
			},
		},
		map[string]attr.Value{
			"section": types.ObjectValueMust(
				map[string]attr.Type{
					"group": types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"integer_key": types.NumberType,
							"float_key":   types.NumberType,
						},
					},
				},
				map[string]attr.Value{
					"group": types.ObjectValueMust(
						map[string]attr.Type{
							"integer_key": types.NumberType,
							"float_key":   types.NumberType,
						},
						map[string]attr.Value{
							"integer_key": types.NumberValue(big.NewFloat(2)),
							"float_key":   types.NumberValue(big.NewFloat(2.5)),
						},
					),
				},
			),
		},
	))

	output, diags := dynamicValueToVariables(t.Context(), inputDynamic)
	if diags.HasError() {
		t.Fatalf("failed to convert dynamic to variables: %v", diags)
	}

	expected := map[string]map[string]map[string]any{
		"section": {
			"group": {
				"integer_key": int64(2),
				"float_key":   2.5,
			},
		},
	}

	assert.Equal(t, expected, output)
}
