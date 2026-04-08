# Testing

This document describes how to run, configure, and extend the acceptance test suite for the Cycloid Terraform provider.

## Contents

- [Test types](#test-types)
- [Running tests](#running-tests)
  - [Unit tests](#unit-tests)
  - [Acceptance tests](#acceptance-tests)
  - [Running a single test](#running-a-single-test)
- [Test configuration](#test-configuration)
  - [Environment variables](#environment-variables)
  - [test_config.yaml](#test_configyaml)
  - [Overriding individual fields](#overriding-individual-fields)
- [Test infrastructure](#test-infrastructure)
  - [TestDependencyManager](#testdependencymanager)
  - [testAccGetTestConfig](#testaccgettestconfig)
  - [testAccPreCheck](#testaccprecheck)
  - [RandomCanonical](#randomcanonical)
- [Writing a test for a new resource](#writing-a-test-for-a-new-resource)
  - [Simple resource (no dependencies)](#simple-resource-no-dependencies)
  - [Resource with pre-existing dependencies](#resource-with-pre-existing-dependencies)
  - [Conditional skip pattern](#conditional-skip-pattern)
- [Known skipped tests](#known-skipped-tests)

---

## Test types

| Type | Command | What runs |
|------|---------|-----------|
| **Unit** | `just test-unit` | Fast, no API calls, no credentials needed |
| **Acceptance** | `just test-acc` | Real API calls against a dedicated test environment |

Unit tests are the default (`just test`). Acceptance tests require `TF_ACC=1` and valid Cycloid credentials — the Terraform Plugin Testing framework skips all `resource.Test` calls automatically when `TF_ACC` is unset.

---

## Running tests

### Unit tests

```bash
just test-unit
# or
go test ./... -short
```

No environment variables required. Runs in seconds.

### Acceptance tests

```bash
# 1. Copy and edit the example config
cp test_config.yaml.example test_config.yaml
$EDITOR test_config.yaml   # fill in credentials, repo URLs, etc.

# 2. Set required environment variables (or put them in a .env file)
export CY_API_URL=https://my-cycloid.example.com
export CY_API_KEY=<your-api-key>
export CY_ORG=<your-org-canonical>

# 3. Run all acceptance tests
just test-acc
```

> **Warning**: Acceptance tests create and destroy real resources in your Cycloid organisation. Use a dedicated test organisation, not production.

### Running a single test

```bash
just test-acc-one TestAccProjectResource
just test-acc-one TestAccCredentialResource_AWS
```

---

## Test configuration

### Environment variables

These three variables are **required** for all acceptance tests:

| Variable | Description |
|----------|-------------|
| `TF_ACC` | Must be `1` to enable acceptance tests (Terraform framework requirement) |
| `CY_API_URL` | Cycloid API endpoint, e.g. `https://http-api.cycloid.io` |
| `CY_API_KEY` | API key with sufficient permissions for the test org |
| `CY_ORG` | Canonical of the organisation used by all tests |

### test_config.yaml

Copy `test_config.yaml.example` to `test_config.yaml` (gitignored) and fill in values for your test environment.

```yaml
# Canonical of the config repository used when creating test projects.
config_repository: "my-config-repo"

repositories:
  # Used by TestAccConfigRepositoryResource; skipped if credential is empty.
  config:
    url: "git@github.com:my-org/my-config-repo.git"
    branch: "main"
    credential: "my-github-credential"   # canonical of a credential in the test org
  # Used by TestAccCatalogRepositoryResource; skipped if credential is empty.
  catalog:
    url: "git@github.com:my-org/my-catalog-repo.git"
    branch: "main"
    credential: "my-github-credential"

# Component tests skip automatically when stack_canonical is empty.
component:
  stack_canonical: "my-stack"   # canonical of a stack that exists in the test org
  use_case: "default"           # use case defined on that stack
  stack_version: "main"         # version tag (branch/tag) on the stack
  input_variables: {}           # nested map serialized as JSON in the resource
```

The file is searched at repo root first, then `provider/`, then the current directory. Set `CY_TEST_CONFIG_FILE` to use a custom path.

Tests that depend on a config field skip automatically when the field is empty. Tests that require the file itself (e.g. project, component) also skip gracefully when the file is missing.

### Overriding individual fields

Any `test_config.yaml` key can be overridden with an environment variable using the `CY_TEST_` prefix and the uppercased key name:

```bash
export CY_TEST_CONFIG_REPOSITORY=my-other-config-repo
export CY_TEST_REPOSITORIES_CATALOG_CREDENTIAL=my-other-cred
```

Nested keys use `_` as a separator between levels.

---

## Test infrastructure

The acceptance test infrastructure lives in three files:

| File | Purpose |
|------|---------|
| `provider/testconfig.go` | Loads `test_config.yaml` with env-var override |
| `provider/test_utils.go` | `testAccPreCheck`, `testAccGetTestConfig`, `RandomCanonical` |
| `provider/test_dependencies.go` | `TestDependencyManager` — pre-creates and cleans up API resources |

### TestDependencyManager

Some resources require pre-existing Cycloid objects (a project before an environment, a project+environment before a component). Rather than creating these inline in Terraform configs — which couples tests to multiple resource implementations — the `TestDependencyManager` creates them via the Cycloid middleware directly and registers automatic cleanup.

```go
ctx := context.Background()
depManager := NewTestDependencyManager(t)
defer depManager.Cleanup(ctx, t)

// Pre-create a project the test depends on
project, err := depManager.EnsureTestProject(ctx, t, orgCanonical, projectName, "")
if err != nil {
    t.Fatalf("failed to ensure test project: %v", err)
}
projectCanonical := ptr.Value(project.Canonical)

// The test Terraform config only manages the resource under test
resource.Test(t, resource.TestCase{
    ProtoV6ProviderFactories: depManager.GetProviderFactories(),
    PreCheck: func() { testAccPreCheck(t) },
    Steps: []resource.TestStep{
        {
            Config: fmt.Sprintf(`
                resource "cycloid_environment" "test" {
                    organization = %q
                    project      = %q
                    name         = %q
                }
            `, orgCanonical, projectCanonical, envName),
        },
    },
})
```

**Key behaviours:**

- `NewTestDependencyManager` initialises the middleware from `CY_API_URL` / `CY_API_KEY` / `CY_ORG`. When credentials are absent, `EnsureTestProject` and `EnsureTestEnvironment` call `t.Skip` — the test does not run.
- `EnsureTestProject` / `EnsureTestEnvironment` are idempotent: they list existing resources and only create if absent.
- `Cleanup` runs items in **reverse creation order** (environments before projects) so dependency constraints are respected.
- Cleanup failures are logged with `t.Logf` rather than failing the test; they indicate leaked resources but should not mask actual test results.
- `GetProviderFactories()` returns the provider factory for `resource.TestCase`, configured with the same credentials.

### testAccGetTestConfig

```go
cfg := testAccGetTestConfig(t)
```

Returns the parsed `TestConfig`. If the config file is not found, **skips** the test with a message telling you where to put the file. This ensures tests are skipped gracefully in CI environments where no `test_config.yaml` is present.

### testAccPreCheck

```go
PreCheck: func() { testAccPreCheck(t) },
```

Called by the Terraform testing framework before each test case. Validates that `CY_API_URL`, `CY_API_KEY`, and `CY_ORG` are set. Skips when `testing.Short()` is true (i.e. `go test -short`), keeping unit test runs fast.

### RandomCanonical

```go
name := RandomCanonical("test-project")   // → "test-project483921"
```

Appends 6 random digits to `baseName`. Use this for any resource that must be unique within the org (projects, environments, teams, credentials, repos). Do **not** use a hardcoded constant — repeated or parallel runs will collide.

---

## Writing a test for a new resource

### Simple resource (no dependencies)

Use this pattern when the resource under test does not require any pre-existing Cycloid objects.

```go
func TestAccMyResource(t *testing.T) {
    t.Parallel()

    resourceName := RandomCanonical("test-myresource")
    orgCanonical := testAccGetOrganizationCanonical()

    ctx := context.Background()
    depManager := NewTestDependencyManager(t)
    defer depManager.Cleanup(ctx, t)

    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: depManager.GetProviderFactories(),
        PreCheck:                 func() { testAccPreCheck(t) },
        Steps: []resource.TestStep{
            // Create
            {
                Config: testAccMyResourceConfig_basic(orgCanonical, resourceName),
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("cycloid_my_resource.test", "organization_canonical", orgCanonical),
                    resource.TestCheckResourceAttr("cycloid_my_resource.test", "name", resourceName),
                ),
            },
            // Update
            {
                Config: testAccMyResourceConfig_updated(orgCanonical, resourceName),
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("cycloid_my_resource.test", "some_field", "new-value"),
                ),
            },
            // Destroy
            {
                Config:  " ",
                Destroy: true,
            },
        },
    })
}

func testAccMyResourceConfig_basic(org, name string) string {
    return fmt.Sprintf(`
resource "cycloid_my_resource" "test" {
  organization_canonical = %q
  name                   = %q
}
`, org, name)
}

func testAccMyResourceConfig_updated(org, name string) string {
    return fmt.Sprintf(`
resource "cycloid_my_resource" "test" {
  organization_canonical = %q
  name                   = %q
  some_field             = "new-value"
}
`, org, name)
}
```

**Config helper conventions:**
- Use `%q` (Go quoted string) rather than `"%s"` in format strings — it handles escaping correctly.
- Name helpers `testAcc<Resource>Config_<variant>`. Only define `_updated` if the body actually differs from `_basic`.
- Put all config helpers at the bottom of the test file.

### Resource with pre-existing dependencies

Use `EnsureTestProject` / `EnsureTestEnvironment` when the resource needs a project or environment to exist before the Terraform step runs.

```go
func TestAccMyResource(t *testing.T) {
    t.Parallel()

    projectName := RandomCanonical("test-project")
    envName     := RandomCanonical("test-env")
    orgCanonical := testAccGetOrganizationCanonical()

    ctx := context.Background()
    depManager := NewTestDependencyManager(t)
    defer depManager.Cleanup(ctx, t)

    project, err := depManager.EnsureTestProject(ctx, t, orgCanonical, projectName, "")
    if err != nil {
        t.Fatalf("failed to ensure test project: %v", err)
    }
    projectCanonical := ptr.Value(project.Canonical)

    env, err := depManager.EnsureTestEnvironment(ctx, t, orgCanonical, projectCanonical, envName)
    if err != nil {
        t.Fatalf("failed to ensure test environment: %v", err)
    }
    envCanonical := ptr.Value(env.Canonical)

    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: depManager.GetProviderFactories(),
        PreCheck:                 func() { testAccPreCheck(t) },
        Steps: []resource.TestStep{ /* ... */ },
    })
}
```

The cleanup registered by `EnsureTestProject` / `EnsureTestEnvironment` runs after `resource.Test` completes, in reverse order (environment first, then project).

### Conditional skip pattern

Use `testAccGetTestConfig` when the test needs values from `test_config.yaml`. Add a `t.Skip` guard early if a required field is empty:

```go
cfg := testAccGetTestConfig(t)   // skips if file not found
if cfg.Repositories.Config.Credential == "" {
    t.Skip("repositories.config.credential must be set in test_config.yaml for this test")
}
```

---

## Known skipped tests

Some tests are unconditionally skipped pending fixes or missing prerequisites:

| Test | Reason |
|------|--------|
| `TestAccOrganizationResource_WithAllowDestroy` | Child org creation requires elevated API permissions not available in the standard test environment |
| `TestAccStackResource` | HCL config referencing the `cycloid_stacks` data source needs rework; tracked separately |
| `TestAccTeamMemberResource` | `assignMemberToTeam` returns HTTP 500; under investigation |
