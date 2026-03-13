package provider

import (
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/dynamic"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
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

func TestGetInputVariablesForReadReturnsFilteredBackendInputsWhenVariableUpdatesEnabled(t *testing.T) {
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
				"project_owners_raw":      "owner-from-backend@example.com",
				"project_maintainers_raw": "maintainer-from-backend@example.com",
				"project_extra_raw":       "extra-from-backend@example.com",
			},
		},
	}

	output, diags := getInputVariablesForRead(t.Context(), componentState, backendCurrentConfig)
	if diags.HasError() {
		t.Fatalf("failed to get input variables for read: %v", diags)
	}

	expected := map[string]map[string]map[string]any{
		"Definition of the development project": {
			"Users settings": {
				"project_owners_raw":      "owner-from-backend@example.com",
				"project_maintainers_raw": "maintainer-from-backend@example.com",
			},
		},
	}

	assert.Equal(t, expected, output, "Read must detect backend drift for user-managed input_variables")
}

func TestGetInputVariablesForReadReturnsStateInputsWhenVariableUpdatesDisabled(t *testing.T) {
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
		AllowVariableUpdate: types.BoolValue(false),
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

	assert.Equal(t, stateInputVariables, output, "Read must preserve state when variable updates are disabled")
}

func TestComponentToModelSetsInputVariables(t *testing.T) {
	inputVariables := map[string]map[string]map[string]any{
		"section": {
			"group": {
				"key": "value",
			},
		},
	}
	currentConfig := map[string]map[string]map[string]any{
		"section": {
			"group": {
				"key": "value",
			},
		},
	}

	componentState := componentResourceModel{}
	component := &models.Component{
		ServiceCatalog: &models.ServiceCatalog{
			Ref: ptr.Ptr("org:stack"),
		},
	}

	diags := ComponentToModel(t.Context(), "org", component, inputVariables, currentConfig, &componentState)
	if diags.HasError() {
		t.Fatalf("failed to map component to model: %v", diags)
	}

	output, diags := dynamicValueToVariables(t.Context(), componentState.InputVariables)
	if diags.HasError() {
		t.Fatalf("failed to convert model input_variables: %v", diags)
	}

	assert.Equal(t, inputVariables, output, "ComponentToModel must refresh input_variables in state")
}
