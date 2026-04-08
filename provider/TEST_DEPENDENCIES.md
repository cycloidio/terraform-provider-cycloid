# Test Dependency Management System

This document describes the test dependency management system for handling resource dependencies in Terraform acceptance tests.

## Overview

The `TestDependencyManager` provides a way to create and manage test dependencies using the Cycloid middleware, allowing tests to work with pre-existing resources instead of creating them inline in Terraform configurations.

## Key Benefits

1. **Cleaner Test Configurations**: Test configurations no longer need to include dependency resources
2. **Realistic Testing**: Tests work with actual API-created resources, not mock ones
3. **Dependency Management**: Automatic cleanup of created dependencies
4. **Flexible**: Works for both unit tests (without middleware) and acceptance tests (with middleware)

## Usage Example

### Environment Resource Test

```go
func TestAccEnvironmentResource(t *testing.T) {
    t.Parallel()

    // Test constants
    const (
        projectName = "test-project"
        envName     = "test-environment"
    )

    ctx := context.Background()
    orgCanonical := testAccGetOrganizationCanonical()

    // Set up dependency manager
    depManager := NewTestDependencyManager(t)
    defer depManager.Cleanup(ctx, t)

    // Create test project dependency
    projectCanonical, err := depManager.EnsureTestProject(ctx, t, projectName, "Test project for environment testing")
    if err != nil {
        t.Fatalf("Failed to create test project dependency: %v", err)
    }

    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: depManager.GetProviderFactories(),
        PreCheck: func() { testAccPreCheck(t) },
        Steps: []resource.TestStep{
            // Create environment with existing project dependency
            {
                Config: testAccEnvironmentConfig_basic_withDependency(orgCanonical, projectCanonical, envName),
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("cycloid_environment.test", "organization", orgCanonical),
                    resource.TestCheckResourceAttr("cycloid_environment.test", "project", projectCanonical),
                    resource.TestCheckResourceAttr("cycloid_environment.test", "name", envName),
                ),
            },
            // ... other test steps
        },
    })
}
```

### Configuration Functions

The new configuration functions work with pre-existing dependencies:

```go
func testAccEnvironmentConfig_basic_withDependency(org, projectCanonical, env string) string {
    return fmt.Sprintf(`
resource "cycloid_environment" "test" {
  organization = "%s"
  project     = "%s"
  name        = "%s"
}
`, org, projectCanonical, env)
}
```

## API Reference

### TestDependencyManager

#### Methods

- `NewTestDependencyManager(t *testing.T) *TestDependencyManager`: Creates a new dependency manager
- `EnsureTestProject(ctx, t, org, name, description) (string, error)`: Creates or returns existing project in the given org
- `CreateTestProject(ctx, t, org, name, description) (string, error)`: Creates a new project in the given org
- `Cleanup(ctx, t)`: Cleans up all created dependencies
- `GetProviderFactories()`: Returns provider factories for resource.TestCase

## Behavior

### Unit Tests (No Environment Variables)
- When `CY_API_KEY`, `CY_API_URL`, or `CY_ORG` are not set
- Middleware is not initialized
- `EnsureTestProject` returns the project name as canonical
- Dependencies are managed by Terraform during test execution

### Acceptance Tests (With Environment Variables)
- When all required environment variables are set
- Middleware is initialized with Cycloid API
- `EnsureTestProject` creates actual API resources
- Dependencies are automatically cleaned up

## Extending the System

To add support for other resource dependencies:

1. Add creation methods to `TestDependencyManager`
2. Add cleanup logic to track created resources
3. Create new configuration functions that use pre-existing dependencies
4. Update tests to use the dependency manager

### Example: Adding Team Support

```go
// Add to TestDependencyManager
func (dm *TestDependencyManager) EnsureTestTeam(ctx context.Context, t *testing.T, name string) (string, error) {
    if dm.provider.Middleware == nil {
        return name, nil
    }
    // ... API logic to create/check team
}

// Configuration function
func testAccTeamMemberConfig_withDependency(org, teamCanonical, username, email string) string {
    return fmt.Sprintf(`
resource "cycloid_team_member" "test" {
  organization = "%s"
  team         = "%s"
  username     = "%s"
  email        = "%s"
}
`, org, teamCanonical, username, email)
}
```

## Migration Guide

To migrate existing tests to use the dependency system:

1. **Before**: Inline dependency creation
```go
Config: testAccEnvironmentConfig_basic(org, project, env)
```

2. **After**: Pre-existing dependency
```go
projectCanonical, _ := depManager.EnsureTestProject(ctx, t, project, "Test project")
Config: testAccEnvironmentConfig_basic_withDependency(org, projectCanonical, env)
```

This approach provides cleaner, more maintainable tests while ensuring dependencies are properly managed.
