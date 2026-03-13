# Agent Development Notes

This file contains important notes and context for the AI agent working on this terraform provider.

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
- Use `make install-provider` or `go install .` for the provider binary. Do not use `make install` or any codegen targets (`make tf-generate`, `make convert-swagger`) — codegen is being dropped in a future release.
- All provider operations require a remote Cycloid API (`CY_API_URL`, `CY_API_KEY`, `CY_ORG` env vars). A dedicated testing environment is used for tests — do not run against production. No local services to start.
- `staticcheck` reports pre-existing warnings (unused vars/funcs); these are not blockers.
