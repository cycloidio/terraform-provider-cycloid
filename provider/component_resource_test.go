package provider

import (
	"testing"

	"github.com/cycloidio/terraform-provider-cycloid/internal/dynamic"
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

func TestGetInputVariablesForReadReturnsStateInputsWhenVariableUpdatesEnabled(t *testing.T) {
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
		AllowVariableUpdate: types.BoolValue(true),
		InputVariables:      stateInputDynamic,
	}

	backendCurrentConfig := map[string]map[string]map[string]any{
		"Definition of the development project": {
			"Users settings": {
				"project_owners_raw": "owner-from-backend@example.com",
			},
		},
	}

	output, diags := getInputVariablesForRead(t.Context(), componentState, backendCurrentConfig)
	if diags.HasError() {
		t.Fatalf("failed to get input variables for read: %v", diags)
	}

	assert.Equal(t, stateInputVariables, output, "Read must preserve user-provided input_variables from state")
}
