package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/dynamic"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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

	assert.Equal(t, expected, output, "when variable updates are enabled, Read returns backend values for user-defined keys that exist in the API config")
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

	diags := ComponentToModel(t.Context(), "org", component, inputVariables, currentConfig, &componentState, true)
	if diags.HasError() {
		t.Fatalf("failed to map component to model: %v", diags)
	}

	output, diags := dynamicValueToVariables(t.Context(), componentState.InputVariables)
	if diags.HasError() {
		t.Fatalf("failed to convert model input_variables: %v", diags)
	}

	assert.Equal(t, inputVariables, output, "ComponentToModel must refresh input_variables in state when refreshInputVariables is true")
}

// defaultComponentInputVarsJSON is used when test config component.input_variables is empty.
const defaultComponentInputVarsJSON = `{"application":{"config":{"replicas":2,"cpu_limit":"500m"}}}`

func componentInputVars(cfg *TestConfig) string {
	if cfg != nil && cfg.Component != nil && cfg.Component.InputVariables != nil && len(cfg.Component.InputVariables) > 0 {
		b, err := json.Marshal(cfg.Component.InputVariables)
		if err != nil {
			return defaultComponentInputVarsJSON
		}
		return string(b)
	}
	return defaultComponentInputVarsJSON
}

func componentUseCase(cfg *TestConfig) string {
	if cfg != nil && cfg.Component != nil && cfg.Component.UseCase != "" {
		return cfg.Component.UseCase
	}
	return "default"
}

func TestAccComponentResource(t *testing.T) {
	t.Parallel()

	const (
		componentName = "test-component"
		componentDesc = "Test component for acceptance testing"
	)
	projectName := RandomCanonical("test-project")
	envName := RandomCanonical("test-env")

	orgCanonical := testAccGetOrganizationCanonical()
	cfg := testAccGetTestConfig(t)
	if cfg.Component == nil || cfg.Component.StackCanonical == "" {
		t.Skip("component.stack_canonical must be set in test_config.yaml for component acceptance tests")
	}
	stackRef, stackVersion := orgCanonical+":"+cfg.Component.StackCanonical, cfg.Component.StackVersion
	useCase := componentUseCase(cfg)
	if stackVersion == "" {
		stackVersion = "main"
	}
	ctx := context.Background()
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	// Pre-create project and environment (nested dependency); cleanup runs in reverse order after all steps.
	testProject, err := depManager.EnsureTestProject(ctx, t, orgCanonical, projectName, "")
	if err != nil {
		t.Fatalf("failed to ensure test project: %v", err)
	}
	projectCanonical := ptr.Value(testProject.Canonical)

	testEnv, err := depManager.EnsureTestEnvironment(ctx, t, orgCanonical, projectCanonical, envName)
	if err != nil {
		t.Fatalf("failed to ensure test environment: %v", err)
	}
	envCanonical := ptr.Value(testEnv.Canonical)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create component with full control - tests CreateAndConfigureComponent middleware
			{
				Config:             testAccComponentConfig_fullControl(orgCanonical, projectCanonical, envCanonical, componentName, componentDesc, stackRef, useCase, stackVersion, componentInputVars(cfg)),
				ExpectNonEmptyPlan: true, // API may return input_variables/canonical in a different shape than config after create
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_component.test", "project", projectCanonical),
					resource.TestCheckResourceAttr("cycloid_component.test", "environment", envCanonical),
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "canonical", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "description", componentDesc),
					resource.TestCheckResourceAttr("cycloid_component.test", "stack_ref", stackRef),
					resource.TestCheckResourceAttr("cycloid_component.test", "use_case", useCase),
					resource.TestCheckResourceAttr("cycloid_component.test", "stack_version", stackVersion),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_version_update", "true"),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_variable_update", "true"),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_destroy", "true"),
				),
			},
			// Update component
			{
				Config: testAccComponentConfig_fullControl(orgCanonical, projectCanonical, envCanonical, componentName+"-updated", componentDesc+" updated", stackRef, "staging", stackVersion, componentInputVars(cfg)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName+"-updated"),
					resource.TestCheckResourceAttr("cycloid_component.test", "description", componentDesc+" updated"),
					resource.TestCheckResourceAttr("cycloid_component.test", "use_case", "staging"),
				),
			},
			// Set allow_destroy=false so that destroy is rejected
			{
				Config: testAccComponentConfig_protectedComponent(orgCanonical, projectCanonical, envCanonical, componentName+"-updated", stackRef, "staging"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_destroy", "false"),
				),
			},
			// Assert that destroy must fail when allow_destroy=false
			{
				Config:      testAccComponentConfig_protectedComponent(orgCanonical, projectCanonical, envCanonical, componentName+"-updated", stackRef, "staging"),
				Destroy:     true,
				ExpectError: regexp.MustCompile(`Component deletion not allowed`),
			},
			// Re-enable destroy so post-test cleanup can delete the component
			{
				Config: testAccComponentConfig_fullControl(orgCanonical, projectCanonical, envCanonical, componentName+"-updated", componentDesc+" updated", stackRef, "staging", stackVersion, componentInputVars(cfg)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_destroy", "true"),
				),
			},
		},
	})
}

func TestAccComponentResource_WithVariableUpdateFalse(t *testing.T) {
	t.Parallel()

	const componentName = "no-var-update-component"
	projectName := RandomCanonical("test-project")
	envName := RandomCanonical("test-env")

	orgCanonical := testAccGetOrganizationCanonical()
	cfg := testAccGetTestConfig(t)
	if cfg.Component == nil || cfg.Component.StackCanonical == "" {
		t.Skip("component.stack_canonical must be set in test_config.yaml for component acceptance tests")
	}
	stackRef, stackVersion := orgCanonical+":"+cfg.Component.StackCanonical, cfg.Component.StackVersion
	useCase := componentUseCase(cfg)
	if stackVersion == "" {
		stackVersion = "v1.0.0"
	}
	ctx := context.Background()
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	testProject, err := depManager.EnsureTestProject(ctx, t, orgCanonical, projectName, "")
	if err != nil {
		t.Fatalf("failed to ensure test project: %v", err)
	}
	projectCanonical := ptr.Value(testProject.Canonical)
	testEnv, err := depManager.EnsureTestEnvironment(ctx, t, orgCanonical, projectCanonical, envName)
	if err != nil {
		t.Fatalf("failed to ensure test environment: %v", err)
	}
	envCanonical := ptr.Value(testEnv.Canonical)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccComponentConfig_noVariableUpdate(orgCanonical, projectCanonical, envCanonical, componentName, stackRef, useCase, stackVersion, componentInputVars(cfg)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_variable_update", "false"),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_version_update", "true"),
				),
			},
			{
				Config: testAccComponentConfig_noVariableUpdate(orgCanonical, projectCanonical, envCanonical, componentName, stackRef, useCase, stackVersion, componentInputVars(cfg)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_variable_update", "false"),
				),
			},
			{
				Config:  testAccComponentConfig_noVariableUpdate(orgCanonical, projectCanonical, envCanonical, componentName, stackRef, useCase, stackVersion, componentInputVars(cfg)),
				Destroy: true,
			},
		},
	})
}

func TestAccComponentResource_DriftDetection(t *testing.T) {
	t.Parallel()

	const componentName = "drift-test-component"
	projectName := RandomCanonical("test-project")
	envName := RandomCanonical("test-env")

	orgCanonical := testAccGetOrganizationCanonical()
	cfg := testAccGetTestConfig(t)
	if cfg.Component == nil || cfg.Component.StackCanonical == "" {
		t.Skip("component.stack_canonical must be set in test_config.yaml for component acceptance tests")
	}
	stackRef, stackVersion := orgCanonical+":"+cfg.Component.StackCanonical, cfg.Component.StackVersion
	useCase := componentUseCase(cfg)
	if stackVersion == "" {
		stackVersion = "v1.0.0"
	}
	ctx := context.Background()
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	testProject, err := depManager.EnsureTestProject(ctx, t, orgCanonical, projectName, "")
	if err != nil {
		t.Fatalf("failed to ensure test project: %v", err)
	}
	projectCanonical := ptr.Value(testProject.Canonical)
	testEnv, err := depManager.EnsureTestEnvironment(ctx, t, orgCanonical, projectCanonical, envName)
	if err != nil {
		t.Fatalf("failed to ensure test environment: %v", err)
	}
	envCanonical := ptr.Value(testEnv.Canonical)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config:             testAccComponentConfig_fullControl(orgCanonical, projectCanonical, envCanonical, componentName, "Drift test component", stackRef, useCase, stackVersion, componentInputVars(cfg)),
				ExpectNonEmptyPlan: true, // API may return input_variables/canonical in a different shape after create
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "description", "Drift test component"),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_variable_update", "true"),
				),
			},
			{
				Config: testAccComponentConfig_fullControl(orgCanonical, projectCanonical, envCanonical, componentName, "Drift test component updated", stackRef, useCase, stackVersion, componentInputVars(cfg)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "description", "Drift test component updated"),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_variable_update", "true"),
				),
			},
			{
				Config:  testAccComponentConfig_fullControl(orgCanonical, projectCanonical, envCanonical, componentName, "Drift test component updated", stackRef, useCase, stackVersion, componentInputVars(cfg)),
				Destroy: true,
			},
		},
	})
}

func TestAccComponentResource_CreateAndUpdateOnly(t *testing.T) {
	t.Parallel()

	const componentName = "create-update-component"
	projectName := RandomCanonical("test-project")
	envName := RandomCanonical("test-env")

	orgCanonical := testAccGetOrganizationCanonical()
	cfg := testAccGetTestConfig(t)
	if cfg.Component == nil || cfg.Component.StackCanonical == "" {
		t.Skip("component.stack_canonical must be set in test_config.yaml for component acceptance tests")
	}
	stackRef, stackVersion := orgCanonical+":"+cfg.Component.StackCanonical, cfg.Component.StackVersion
	useCase := componentUseCase(cfg)
	if stackVersion == "" {
		stackVersion = "v1.0.0"
	}
	ctx := context.Background()
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	testProject, err := depManager.EnsureTestProject(ctx, t, orgCanonical, projectName, "")
	if err != nil {
		t.Fatalf("failed to ensure test project: %v", err)
	}
	projectCanonical := ptr.Value(testProject.Canonical)
	testEnv, err := depManager.EnsureTestEnvironment(ctx, t, orgCanonical, projectCanonical, envName)
	if err != nil {
		t.Fatalf("failed to ensure test environment: %v", err)
	}
	envCanonical := ptr.Value(testEnv.Canonical)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccComponentConfig_createAndUpdateOnly(orgCanonical, projectCanonical, envCanonical, componentName, stackRef, useCase, stackVersion, componentInputVars(cfg)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "stack_ref", stackRef),
					resource.TestCheckResourceAttr("cycloid_component.test", "use_case", useCase),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_version_update", "false"),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_variable_update", "false"),
				),
			},
			// Update description only (do not rename component; API identifies by canonical and rename can cause 404 on read)
			{
				Config: testAccComponentConfig_createAndUpdateOnlyUpdated(orgCanonical, projectCanonical, envCanonical, componentName, stackRef, useCase, stackVersion, componentInputVars(cfg)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "description", "Updated description"),
				),
			},
			{
				Config:  testAccComponentConfig_createAndUpdateOnlyUpdated(orgCanonical, projectCanonical, envCanonical, componentName, stackRef, useCase, stackVersion, componentInputVars(cfg)),
				Destroy: true,
			},
		},
	})
}

func TestAccComponentResource_WithPreventDestroy(t *testing.T) {
	t.Parallel()

	const componentName = "protected-component"
	projectName := RandomCanonical("test-project")
	envName := RandomCanonical("test-env")

	orgCanonical := testAccGetOrganizationCanonical()
	cfg := testAccGetTestConfig(t)
	if cfg.Component == nil || cfg.Component.StackCanonical == "" {
		t.Skip("component.stack_canonical must be set in test_config.yaml for component acceptance tests")
	}
	stackRef := orgCanonical + ":" + cfg.Component.StackCanonical
	useCase := componentUseCase(cfg)
	ctx := context.Background()
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	testProject, err := depManager.EnsureTestProject(ctx, t, orgCanonical, projectName, "")
	if err != nil {
		t.Fatalf("failed to ensure test project: %v", err)
	}
	projectCanonical := ptr.Value(testProject.Canonical)
	testEnv, err := depManager.EnsureTestEnvironment(ctx, t, orgCanonical, projectCanonical, envName)
	if err != nil {
		t.Fatalf("failed to ensure test environment: %v", err)
	}
	envCanonical := ptr.Value(testEnv.Canonical)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccComponentConfig_protectedComponent(orgCanonical, projectCanonical, envCanonical, componentName, stackRef, useCase),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_destroy", "false"),
				),
			},
			{
				Config:      testAccComponentConfig_protectedComponent(orgCanonical, projectCanonical, envCanonical, componentName, stackRef, useCase),
				Destroy:     true,
				ExpectError: regexp.MustCompile(`Component deletion not allowed`),
			},
		},
	})
}

func testAccComponentConfig_fullControl(org, project, env, name, desc, stackRef, useCase, stackVersion, inputVarsJSON string) string {
	return fmt.Sprintf(`
resource "cycloid_component" "test" {
  organization        = "%s"
  project           = "%s"
  environment        = "%s"
  name              = "%s"
  description       = "%s"
  stack_ref         = "%s"
  use_case          = "%s"
  stack_version      = "%s"

  input_variables = jsondecode(%q)

  allow_version_update  = true
  allow_variable_update = true
  allow_destroy       = true
}
`, org, project, env, name, desc, stackRef, useCase, stackVersion, inputVarsJSON)
}

func testAccComponentConfig_noVariableUpdate(org, project, env, name, stackRef, useCase, stackVersion, inputVarsJSON string) string {
	return fmt.Sprintf(`
resource "cycloid_component" "test" {
  organization        = "%s"
  project           = "%s"
  environment        = "%s"
  name              = "%s"
  stack_ref         = "%s"
  use_case          = "%s"
  stack_version      = "%s"

  input_variables = jsondecode(%q)

  allow_version_update  = true
  allow_variable_update = false
  allow_destroy       = true
}
`, org, project, env, name, stackRef, useCase, stackVersion, inputVarsJSON)
}

func testAccComponentConfig_createAndUpdateOnly(org, project, env, name, stackRef, useCase, stackVersion, inputVarsJSON string) string {
	return fmt.Sprintf(`
resource "cycloid_component" "test" {
  organization        = "%s"
  project           = "%s"
  environment        = "%s"
  name              = "%s"
  stack_ref         = "%s"
  use_case          = "%s"
  stack_version      = "%s"

  input_variables = jsondecode(%q)

  allow_version_update  = false
  allow_variable_update = false
  allow_destroy       = true
}
`, org, project, env, name, stackRef, useCase, stackVersion, inputVarsJSON)
}

func testAccComponentConfig_createAndUpdateOnlyUpdated(org, project, env, name, stackRef, useCase, stackVersion, inputVarsJSON string) string {
	return fmt.Sprintf(`
resource "cycloid_component" "test" {
  organization        = "%s"
  project           = "%s"
  environment        = "%s"
  name              = "%s"
  stack_ref         = "%s"
  use_case          = "%s"
  stack_version      = "%s"
  description       = "Updated description"

  input_variables = jsondecode(%q)

  allow_version_update  = false
  allow_variable_update = false
  allow_destroy       = true
}
`, org, project, env, name, stackRef, useCase, stackVersion, inputVarsJSON)
}

func testAccComponentConfig_protectedComponent(org, project, env, name, stackRef, useCase string) string {
	return fmt.Sprintf(`
resource "cycloid_component" "test" {
  organization        = "%s"
  project           = "%s"
  environment        = "%s"
  name              = "%s"
  stack_ref         = "%s"
  use_case          = "%s"

  allow_destroy       = false
}
`, org, project, env, name, stackRef, useCase)
}

func TestIsComponentNotFoundError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "component config not found operation",
			err:      errors.New("A 404 error was returned on \"getComponentConfigNotFound\" call with message: The Component was not found"),
			expected: true,
		},
		{
			name:     "component not found message",
			err:      errors.New("The Component was not found"),
			expected: true,
		},
		{
			name:     "different not found message",
			err:      errors.New("A 404 error was returned on \"getProjectNotFound\" call"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, isComponentNotFoundError(testCase.err))
		})
	}
}
