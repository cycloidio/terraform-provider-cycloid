# Testing

This document describes how to run, configure, and extend the acceptance test suite for the Cycloid Terraform provider.

## Contents

- [Test types](#test-types)
- [Running tests](#running-tests)
  - [Unit tests](#unit-tests)
  - [Acceptance tests — local docker-compose stack](#acceptance-tests--local-docker-compose-stack)
  - [Running a single test](#running-a-single-test)
  - [Remote env override](#remote-env-override-advanced)
- [Bootstrap](#bootstrap)
- [Test infrastructure](#test-infrastructure)
  - [TestMain](#testmain)
  - [TestDependencyManager](#testdependencymanager)
  - [testAccGetTestConfig](#testaccgettestconfig)
  - [testAccPreCheck](#testaccprecheck)
  - [RandomCanonical](#randomcanonical)
- [Plugin resource tests](#plugin-resource-tests)
- [Writing a test for a new resource](#writing-a-test-for-a-new-resource)
- [Known skipped tests](#known-skipped-tests)

---

## Test types

| Type | Command | What runs |
|------|---------|-----------|
| **Unit** | `just test-unit` | Fast, no API calls, no credentials needed |
| **Acceptance** | `just test-acc` | Real API calls against the local docker-compose stack |
| **Single acc test** | `just test-acc-one TestName` | One test, full stack must be up |

Unit tests are the default (`just test`). Acceptance tests require `TF_ACC=1` and a running local stack.

---

## Running tests

### Unit tests

```bash
just test-unit
# or
go test ./... -short
```

No environment variables required.

### Acceptance tests — local docker-compose stack

The acceptance suite runs against a local 12-service Cycloid stack. The `TestMain` bootstrap provisions the admin user, API key, and shared fixtures automatically on first run.

**Prerequisites**

- Docker with Compose v2
- `cy` CLI in PATH (for secret resolution via `cy uri interpolate`)
- `CY_SAAS_API_KEY` — a valid API key for the `cycloid` org in Cycloid SaaS (used only to resolve the licence key secret)

**One-shot path (recommended)**

```bash
export CY_SAAS_API_KEY=<key-from-1Password>
just test-acc-fresh   # brings stack up, mints .env, waits, runs all tests
```

**Step-by-step**

```bash
# 1. Mint .env (resolves cy:// URI for the licence key)
export CY_SAAS_API_KEY=<key>
just env              # creates .env from .env.sample

# 2. Source the env and start the stack
source .env
just be-start
sleep 10              # wait for db + plugin-manager to finish registering

# 3. Run tests (bootstrap auto-provisions org + API key on first run)
just test-acc
```

### Running a single test

```bash
just test-acc-one TestAccProjectResource
just test-acc-one TestAccPluginResource
```

### Remote env override (advanced)

To run against a remote Cycloid environment instead of the local stack:

```bash
export CY_TEST_PROVISION_API=0      # skip bootstrap
export CY_API_KEY=<remote-api-key>
export CY_API_URL=https://http-api.cycloid.io
export CY_ORG=<org-canonical>
TF_ACC=1 go test ./provider/... -count=1
```

> The `test_config.yaml` file is no longer required when running against the local stack.
> For remote env runs, create a `test_config.yaml` to supply repo URLs and credentials
> (see README_TEST_CONFIG.md for the schema).

---

## Bootstrap

When `TF_ACC=1` and `CY_TEST_PROVISION_API=1` (default), the `TestMain` function in
`provider/main_test.go` runs once before all acceptance tests:

1. Calls `testcfg.NewConfig("tfprovider")` from `github.com/cycloidio/cycloid-cli/pkg/testcfg`.
2. Provisions the first org, admin user, and an API key via `InitFirstOrg`.
3. Creates shared fixtures: SSH credential (`local-git`), config repo (`cli-test-config`),
   catalog repo (`cli-test-stacks`), and a common test project / env / component.
4. Writes `CY_API_KEY`, `CY_API_URL`, `CY_ORG` into the process env so all tests see them.
5. Primes `LoadTestConfig` with values derived from the bootstrapped stack.
6. On exit, calls `cfg.Cleanup()` which removes all bootstrapped resources in reverse order.

The `API_LICENCE_KEY` env var is required for provisioning. It is resolved from
`cy://org/cycloid/credentials/...` by `just env`.

---

## Test infrastructure

| File | Purpose |
|------|---------|
| `provider/main_test.go` | `TestMain` — one-time bootstrap via `testcfg.NewConfig` |
| `provider/testconfig.go` | `LoadTestConfig`, `primeTestConfig`, `TestConfig` struct |
| `provider/test_utils.go` | `testAccPreCheck`, `testAccGetTestConfig`, `RandomCanonical` |
| `provider/test_dependencies.go` | `TestDependencyManager` — creates and cleans up API resources |
| `provider/plugin_test_helpers.go` | `ensurePluginHelloWorld` — pushes test image to local registry |

### TestMain

`TestMain` in `provider/main_test.go` is the entry point for the whole acceptance test run.
It wraps `testcfg.NewConfig` (from cycloid-cli) to bootstrap the local stack, then defers
cleanup. Tests do not need to call any bootstrap function themselves.

### TestDependencyManager

Some resources require pre-existing Cycloid objects. `TestDependencyManager` creates them
via middleware and registers automatic LIFO cleanup.

```go
ctx := context.Background()
depManager := NewTestDependencyManager(t)
defer depManager.Cleanup(ctx, t)

project, err := depManager.EnsureTestProject(ctx, t, orgCanonical, projectName, "")
if err != nil {
    t.Fatalf("failed to ensure test project: %v", err)
}
projectCanonical := ptr.Value(project.Canonical)

resource.Test(t, resource.TestCase{
    ProtoV6ProviderFactories: depManager.GetProviderFactories(),
    PreCheck:                 func() { testAccPreCheck(t) },
    Steps:                    []resource.TestStep{...},
})
```

`EnsureTestProject` and `EnsureTestEnvironment` are idempotent and call `t.Skip` when
credentials are not configured.

### testAccGetTestConfig

```go
cfg := testAccGetTestConfig(t)
```

Returns the parsed `TestConfig`. When running against the local stack, this is auto-populated
from the bootstrap (no `test_config.yaml` needed). For remote env runs, falls back to the yaml
file. If neither is available, the test skips.

### testAccPreCheck

```go
PreCheck: func() { testAccPreCheck(t) },
```

Validates `CY_API_URL`, `CY_API_KEY`, `CY_ORG` are set. Skips when `testing.Short()` is true.

### RandomCanonical

```go
name := RandomCanonical("test-project")   // → "test-project483921"
```

Appends 6 random digits. Use for every resource name — hardcoded names cause collisions.

---

## Plugin resource tests

Plugin tests (`TestAccPlugin*`) have additional prerequisites:

- The `plugin-manager` service must be running and have auto-registered (happens automatically on `just be-start`).
- `docker` must be in PATH — the test helper pushes `docker.io/cycloid/plugin-hello-world:1.0.0` to `localhost:5000`.
- Tests that need the image call `ensurePluginHelloWorld(t)` which handles pull/tag/push idempotently.
- Plugin install status flow on the local stack: `pending → running` (never `installed`). The resource polls for `running`.
- Widget-query assertions are deferred: the plugin-manager proxy has a known bug routing queries to the API host instead of the plugin pod. TODO(plugin-manager-proxy-fix).

---

## Writing a test for a new resource

### Simple resource (no dependencies)

```go
func TestAccMyResource(t *testing.T) {
    resourceName := RandomCanonical("test-myresource")
    orgCanonical := testAccGetOrganizationCanonical()
    depManager := NewTestDependencyManager(t)

    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: depManager.GetProviderFactories(),
        PreCheck:                 func() { testAccPreCheck(t) },
        Steps: []resource.TestStep{
            {
                Config: testAccMyResourceConfig(orgCanonical, resourceName),
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("cycloid_my_resource.test", "name", resourceName),
                ),
            },
        },
    })
}
```

### Resource with pre-existing dependencies

```go
project, err := depManager.EnsureTestProject(ctx, t, orgCanonical, projectName, "")
if err != nil {
    t.Fatalf("failed to ensure test project: %v", err)
}
```

### Conditional skip pattern

```go
cfg := testAccGetTestConfig(t)
if cfg.Repositories.Config.Credential == "" {
    t.Skip("repositories.config.credential must be set for this test")
}
```

---

## Known skipped tests

| Test | Reason |
|------|--------|
| `TestAccOrganizationResource_WithAllowDestroy` | Child org creation requires elevated API permissions |
| `TestAccStackResource` | HCL config needs rework; tracked separately |
| `TestAccTeamMemberResource` | `assignMemberToTeam` returns HTTP 500; under investigation |
| `TestAccCatalogRepositoryResource` | Catalog repo test requires a non-empty `credential_canonical`; local catalog repo uses no credential (public GitHub URL). Set `CY_TEST_REPOSITORIES_CATALOG_CREDENTIAL` or `test_config.yaml` to enable. |
| `TestAccConfigRepositoryResource` | Requires `repositories.config.credential` — auto-set by bootstrap to `local-git` on local stack |
