# Agent Development Notes

This file contains important notes and context for the AI agent working on this terraform provider.

## Test Dependency Management

When working with acceptance tests that require dependencies (e.g., environment resources needing projects):

- **Use TestDependencyManager**: Always use the dependency management system for creating test dependencies
- **Idempotent Provisioning**: Ensure dependency creation is idempotent - overwrite existing entities if needed
- **Robust Cleanup**: Cleanup failures should be logged but never cause test failures
- **Dual Mode Support**: System should work for both unit tests (without middleware) and acceptance tests (with middleware)

**Example Usage:**
```go
depManager := NewTestDependencyManager(t)
defer depManager.Cleanup(ctx, t)

projectCanonical, err := depManager.EnsureTestProject(ctx, t, projectName, "Test project")
```

**Key Principles:**
- Provisioning should be idempotent - don't fail if entity already exists
- Cleanup should never fail the test - log warnings and continue
- Handle missing middleware gracefully for unit tests

## API Naming Conventions

From the cycloid API or middleware, you can map these names:

- `service_catalog` == `stack`
- `service_catalog_source` == `catalog_repository`

The former are the legacy API names, the latter are the customer-facing naming conventions.

## Field Mapping Notes

When working with the cycloid-cli models, keep in mind:

- Component fields may use internal API names that differ from Terraform resource field names
- Service catalog references are accessed via `component.ServiceCatalog.Ref`
- Stack versions can be of type "tag", "branch", or commit hash
- Environment and Project names are nested objects that need safe dereferencing

## Implementation Patterns

- Follow the same patterns as `TeamToModel` when implementing `*ToModel` functions
- Use `types.StringPointerValue()` for pointer fields and `types.StringValue()` for direct string fields
- Always handle nil cases for nested objects
- Set default boolean values for fields that don't exist in the API model but are required in Terraform schema

## Error Handling Guidelines

- **Avoid Panics**: Never use `panic()` in production code. Always return proper error diagnostics.
- **Use Diagnostics**: Return `diag.Diagnostics` with descriptive error messages for Terraform framework operations.
- **Clear Messages**: Provide specific error messages that explain what went wrong and what was expected.
- **Graceful Failures**: Handle unexpected input types gracefully instead of crashing the provider.

**Not-found handling**: Use `isNotFoundError(err)` (in `provider/not_found.go`) in every resource `Read`. On a not-found signal, call `resp.State.RemoveResource(ctx)` and return — do not add an error diagnostic. This lets Terraform recreate the resource cleanly.

**Typed API error matching**: The cycloid middleware returns `*cycloidmiddleware.APIResponseError`. Use `errors.As` with the concrete type to match HTTP status codes precisely — do not match status codes by substring on `err.Error()`. See `isCredentialInUseError` in `provider/not_found.go` for an example.

**Model pointer initialization**: Always initialize API response model pointers with `&models.T{}` before passing to `GenericRequest`. A nil pointer causes `json: Unmarshal(nil *models.T)` at runtime even on HTTP 200 responses.

**Example:**
```go
// Bad: panic(litter.Sdump("Incorrect type", valueType))

// Good: 
return nil, diag.Diagnostics{
    diag.NewErrorDiagnostic(
        "Failed to convert dynamic value to variables",
        fmt.Sprintf("Unsupported value type: %T. Expected map[string]interface{}", valueType),
    ),
}
```

## Code Style Guidelines

- **No Comments**: Avoid adding comments to code. The code should be self-explanatory.
- **Descriptive Names**: Use clear, descriptive variable and function names.
- **Minimal Documentation**: Keep code concise without explanatory comments.

## Terraform Naming Conventions

By convention, when referring to the name of an entity in a terraform attribute, we speak of the canonical.

The only exception is when the resource is about the said entity, we refer then to it canonical attribute by the name `canonical`.

**Example:**
- For the `team_resource` we refer to its organization canonical as `organization` 
- But the canonical of the team itself is named `canonical`

## Cursor Cloud specific instructions

This is a **Go Terraform Provider** for the Cycloid DevOps platform. It is a single Go module (not a monorepo) with no local databases or Docker dependencies.

### Services

| Service | Purpose | Command |
|---|---|---|
| Terraform Provider (Go binary) | The only artifact | `make build` or `go install .` |

### Key commands

- **Build**: `make build`
- **Test**: `go test ./... -v` (or `make test`)
- **Lint**: `go vet ./...` (staticcheck available at `~/go/bin/staticcheck ./...`)
- **Install provider**: `go install .` (installs to `~/go/bin/`)

### Running locally with Terraform

To test the provider with Terraform, create a dev override file (see `README.md` and `DEVELOPING_TIPS.md`). The provider binary installed via `go install .` lands in `~/go/bin/`. Point `TF_CLI_CONFIG_FILE` at a `.tfrc` file that uses a `dev_overrides` block referencing that path.

### Gotchas

- Go 1.25+ is required (per `go.mod`). The VM already has Go 1.25.0 pre-installed.
- Terraform is installed at `/usr/local/bin/terraform`.
- Use `make install-provider` or `go install .` for the provider binary. Note: `make install` installs codegen tools, not the provider itself.
- All provider operations require a remote Cycloid API (`CY_API_URL`, `CY_API_KEY`, `CY_ORG` env vars). A dedicated testing environment is used for tests — do not run against production. No local services to start.
- `staticcheck` reports pre-existing warnings (unused vars/funcs); these are not blockers.
- `just` (command runner) is required for Makefile targets. Install via `curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin`.
- Unit tests (`go test ./... -short`) will show `TestAccEnvironmentResource` as FAIL — this is a pre-existing bug where the test doesn't skip in non-acceptance mode. All actual unit tests pass. Use `-run 'Test[^A]|TestA[^c]'` to exclude acceptance tests cleanly, or just ignore that single failure.

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **terraform-provider-cycloid** (3316 symbols, 6205 relationships, 142 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/terraform-provider-cycloid/context` | Codebase overview, check index freshness |
| `gitnexus://repo/terraform-provider-cycloid/clusters` | All functional areas |
| `gitnexus://repo/terraform-provider-cycloid/processes` | All execution flows |
| `gitnexus://repo/terraform-provider-cycloid/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->
