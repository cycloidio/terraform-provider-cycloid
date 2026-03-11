package provider

import (
	"fmt"
	"math/big"
	"regexp"
	"testing"

	"github.com/cycloidio/terraform-provider-cycloid/internal/dynamic"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
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

func TestAccComponentResource(t *testing.T) {
	t.Parallel()

	// Test constants
	const (
		orgName       = "test-org"
		projectName   = "test-project"
		envName       = "test-environment"
		componentName = "test-component"
		componentDesc = "Test component for acceptance testing"
		stackRef      = "test-org:web-app-stack"
		useCase       = "production"
		stackVersion  = "v1.0.0"
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"cycloid": providerserver.NewProtocol6WithError(&CycloidProvider{}),
		},
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create project and environment first
			{
				Config: testAccComponentConfig_projectEnv(orgName, projectName, envName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_project.test", "name", projectName),
					resource.TestCheckResourceAttr("cycloid_project.test", "canonical", projectName),
					resource.TestCheckResourceAttr("cycloid_environment.test", "name", envName),
					resource.TestCheckResourceAttr("cycloid_environment.test", "canonical", envName),
				),
			},
			// Create component with full control - tests CreateAndConfigureComponent middleware
			{
				Config: testAccComponentConfig_fullControl(orgName, projectName, envName, componentName, componentDesc, stackRef, useCase, stackVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "organization", orgName),
					resource.TestCheckResourceAttr("cycloid_component.test", "project", projectName),
					resource.TestCheckResourceAttr("cycloid_component.test", "environment", envName),
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "canonical", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "description", componentDesc),
					resource.TestCheckResourceAttr("cycloid_component.test", "stack_ref", stackRef),
					resource.TestCheckResourceAttr("cycloid_component.test", "use_case", useCase),
					resource.TestCheckResourceAttr("cycloid_component.test", "stack_version", stackVersion),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_version_update", "true"),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_variable_update", "true"),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_destroy", "true"),
					resource.TestCheckResourceAttrSet("cycloid_component.test", "id"),
				),
			},
			// Update component
			{
				Config: testAccComponentConfig_fullControlUpdated(orgName, projectName, envName, componentName+"-updated", componentDesc+" updated", stackRef, "staging", stackVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName+"-updated"),
					resource.TestCheckResourceAttr("cycloid_component.test", "description", componentDesc+" updated"),
					resource.TestCheckResourceAttr("cycloid_component.test", "use_case", "staging"),
				),
			},
			// Import testing
			{
				ResourceName:      "cycloid_component.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Destroy testing
			{
				Config:  testAccComponentConfig_projectEnv(orgName, projectName, envName), // Keep project/env for cleanup
				Destroy: true,
			},
		},
	})
}

func TestAccComponentResource_WithVariableUpdateFalse(t *testing.T) {
	t.Parallel()

	// Test constants
	const (
		orgName       = "test-org"
		projectName   = "test-project"
		envName       = "test-environment"
		componentName = "no-var-update-component"
		stackRef      = "test-org:web-app-stack"
		useCase       = "production"
		stackVersion  = "v1.0.0"
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"cycloid": providerserver.NewProtocol6WithError(&CycloidProvider{}),
		},
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create project and environment first
			{
				Config: testAccComponentConfig_projectEnv(orgName, projectName, envName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_project.test", "name", projectName),
					resource.TestCheckResourceAttr("cycloid_environment.test", "name", envName),
				),
			},
			// Create component with allow_variable_update=false
			{
				Config: testAccComponentConfig_noVariableUpdate(orgName, projectName, envName, componentName, stackRef, useCase, stackVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_variable_update", "false"),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_version_update", "true"),
					resource.TestCheckResourceAttrSet("cycloid_component.test", "current_config"),
				),
			},
			// Try to update variables with allow_variable_update=false (should not detect drift)
			{
				Config: testAccComponentConfig_noVariableUpdateUpdated(orgName, projectName, envName, componentName, stackRef, useCase, stackVersion),
				Check: resource.ComposeTestCheckFunc(
					// Variables should remain unchanged due to allow_variable_update=false
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_variable_update", "false"),
				),
			},
			// Destroy testing
			{
				Config:  testAccComponentConfig_projectEnv(orgName, projectName, envName), // Keep project/env for cleanup
				Destroy: true,
			},
		},
	})
}

func TestAccComponentResource_DriftDetection(t *testing.T) {
	t.Parallel()

	// Test constants
	const (
		orgName       = "test-org"
		projectName   = "test-project"
		envName       = "test-environment"
		componentName = "drift-test-component"
		stackRef      = "test-org:web-app-stack"
		useCase       = "production"
		stackVersion  = "v1.0.0"
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"cycloid": providerserver.NewProtocol6WithError(&CycloidProvider{}),
		},
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create project and environment first
			{
				Config: testAccComponentConfig_projectEnv(orgName, projectName, envName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_project.test", "name", projectName),
					resource.TestCheckResourceAttr("cycloid_environment.test", "name", envName),
				),
			},
			// Create component with allow_variable_update=true for drift detection
			{
				Config: testAccComponentConfig_fullControl(orgName, projectName, envName, componentName, "Drift test component", stackRef, useCase, stackVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "description", "Drift test component"),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_variable_update", "true"),
					resource.TestCheckResourceAttrSet("cycloid_component.test", "current_config"),
				),
			},
			// Update variables to trigger drift detection
			{
				Config: testAccComponentConfig_fullControlUpdated(orgName, projectName, envName, componentName, "Drift test component updated", stackRef, useCase, stackVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "description", "Drift test component updated"),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_variable_update", "true"),
					// Should detect and apply variable changes when allow_variable_update=true
				),
			},
			// Destroy testing
			{
				Config:  testAccComponentConfig_projectEnv(orgName, projectName, envName), // Keep project/env for cleanup
				Destroy: true,
			},
		},
	})
}

func TestAccComponentResource_CreateAndUpdateOnly(t *testing.T) {
	t.Parallel()

	// Test constants
	const (
		orgName       = "test-org"
		projectName   = "test-project"
		envName       = "test-environment"
		componentName = "create-update-component"
		stackRef      = "test-org:web-app-stack"
		useCase       = "production"
		stackVersion  = "v1.0.0"
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"cycloid": providerserver.NewProtocol6WithError(&CycloidProvider{}),
		},
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create project and environment first
			{
				Config: testAccComponentConfig_projectEnv(orgName, projectName, envName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_project.test", "name", projectName),
					resource.TestCheckResourceAttr("cycloid_environment.test", "name", envName),
				),
			},
			// Create component - tests CreateAndConfigureComponent middleware
			{
				Config: testAccComponentConfig_createAndUpdateOnly(orgName, projectName, envName, componentName, stackRef, useCase, stackVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "stack_ref", stackRef),
					resource.TestCheckResourceAttr("cycloid_component.test", "use_case", useCase),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_version_update", "false"),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_variable_update", "false"),
					resource.TestCheckResourceAttrSet("cycloid_component.test", "current_config"),
				),
			},
			// Update component - tests CreateAndConfigureComponent middleware for updates
			{
				Config: testAccComponentConfig_createAndUpdateOnlyUpdated(orgName, projectName, envName, componentName, stackRef, useCase, stackVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName+"-updated"),
					resource.TestCheckResourceAttr("cycloid_component.test", "description", "Updated description"),
					// Should not update versions or variables due to flags being false
				),
			},
			// Destroy testing
			{
				Config:  testAccComponentConfig_projectEnv(orgName, projectName, envName), // Keep project/env for cleanup
				Destroy: true,
			},
		},
	})
}

func TestAccComponentResource_WithPreventDestroy(t *testing.T) {
	t.Parallel()

	// Test constants
	const (
		orgName       = "test-org"
		projectName   = "test-project"
		envName       = "test-environment"
		componentName = "protected-component"
		stackRef      = "test-org:web-app-stack"
		useCase       = "production"
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"cycloid": providerserver.NewProtocol6WithError(&CycloidProvider{}),
		},
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create project and environment first
			{
				Config: testAccComponentConfig_projectEnv(orgName, projectName, envName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_project.test", "name", projectName),
					resource.TestCheckResourceAttr("cycloid_environment.test", "name", envName),
				),
			},
			// Create component with allow_destroy=false (should be default)
			{
				Config: testAccComponentConfig_protectedComponent(orgName, projectName, envName, componentName, stackRef, useCase),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_component.test", "name", componentName),
					resource.TestCheckResourceAttr("cycloid_component.test", "allow_destroy", "false"),
				),
			},
			// Try to destroy with allow_destroy=false (should fail)
			{
				Config:      testAccComponentConfig_projectEnv(orgName, projectName, envName), // Keep project/env for cleanup
				Destroy:     true,
				ExpectError: regexp.MustCompile(`Component deletion not allowed`),
			},
		},
	})
}

// Test configuration functions
func testAccComponentConfig_projectEnv(org, project, env string) string {
	return fmt.Sprintf(`
resource "cycloid_project" "test" {
  organization = "%s"
  name         = "%s"
  description  = "Test project for component acceptance tests"
}

resource "cycloid_environment" "test" {
  organization = "%s"
  project     = cycloid_project.test.name
  name         = "%s"
}
`, org, project, org, env)
}

func testAccComponentConfig_fullControl(org, project, env, name, desc, stackRef, useCase, stackVersion string) string {
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

  input_variables = {
    "application" = {
      "config" = {
        "replicas"  = 2
        "cpu_limit" = "500m"
      }
    }
  }

  allow_version_update  = true
  allow_variable_update = true
  allow_destroy       = true
}
`, org, project, env, name, desc, stackRef, useCase, stackVersion)
}

func testAccComponentConfig_fullControlUpdated(org, project, env, name, desc, stackRef, useCase, stackVersion string) string {
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

  input_variables = {
    "application" = {
      "config" = {
        "replicas"  = 3
        "cpu_limit" = "1000m"
      }
    }
  }

  allow_version_update  = true
  allow_variable_update = true
  allow_destroy       = true
}
`, org, project, env, name, desc, stackRef, useCase, stackVersion)
}

func testAccComponentConfig_noVariableUpdate(org, project, env, name, stackRef, useCase, stackVersion string) string {
	return fmt.Sprintf(`
resource "cycloid_component" "test" {
  organization        = "%s"
  project           = "%s"
  environment        = "%s"
  name              = "%s"
  stack_ref         = "%s"
  use_case          = "%s"
  stack_version      = "%s"

  input_variables = {
    "application" = {
      "config" = {
        "replicas"  = 1
        "cpu_limit" = "200m"
      }
    }
  }

  allow_version_update  = true
  allow_variable_update = false
  allow_destroy       = true
}
`, org, project, env, name, stackRef, useCase, stackVersion)
}

func testAccComponentConfig_noVariableUpdateUpdated(org, project, env, name, stackRef, useCase, stackVersion string) string {
	return fmt.Sprintf(`
resource "cycloid_component" "test" {
  organization        = "%s"
  project           = "%s"
  environment        = "%s"
  name              = "%s"
  stack_ref         = "%s"
  use_case          = "%s"
  stack_version      = "%s"

  input_variables = {
    "application" = {
      "config" = {
        "replicas"  = 2
        "cpu_limit" = "400m"
      }
    }
  }

  allow_version_update  = true
  allow_variable_update = false
  allow_destroy       = true
}
`, org, project, env, name, stackRef, useCase, stackVersion)
}

func testAccComponentConfig_createAndUpdateOnly(org, project, env, name, stackRef, useCase, stackVersion string) string {
	return fmt.Sprintf(`
resource "cycloid_component" "test" {
  organization        = "%s"
  project           = "%s"
  environment        = "%s"
  name              = "%s"
  stack_ref         = "%s"
  use_case          = "%s"
  stack_version      = "%s"

  input_variables = {
    "application" = {
      "config" = {
        "replicas"  = 1
        "cpu_limit" = "200m"
      }
    }
  }

  allow_version_update  = false
  allow_variable_update = false
  allow_destroy       = true
}
`, org, project, env, name, stackRef, useCase, stackVersion)
}

func testAccComponentConfig_createAndUpdateOnlyUpdated(org, project, env, name, stackRef, useCase, stackVersion string) string {
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

  allow_version_update  = false
  allow_variable_update = false
  allow_destroy       = true
}
`, org, project, env, name, stackRef, useCase, stackVersion)
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
