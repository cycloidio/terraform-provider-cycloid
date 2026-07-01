# Agent Development Notes

This file contains important notes and context for the AI agent working on this terraform provider.

## Dev environment & testing (devenv)

The toolchain (Go, `golangci-lint`, `tfplugindocs`, `just`, `opentofu`/`terraform`,
`jq`, the `docker compose` client, …) is pinned in [`devenv.nix`](./devenv.nix)
and `devenv.yaml` (`allowUnfree: true` for BSL terraform). **Do not install these
by hand** — enter the devenv:

```sh
devenv shell            # interactive; or `direnv allow` once for auto-enter
devenv shell -- <cmd>   # run a single command inside it
```

[direnv](https://direnv.net/) auto-enters the shell on `cd` (and loads `.env`) —
recommended. VS Code users: the [direnv](https://github.com/direnv/direnv-vscode)
and [devenv](https://marketplace.visualstudio.com/items?itemName=datakurre.devenv)
extensions wire the editor to the same environment.

CI runs the exact same env via `devenv shell -- <cmd>` on the `[self-hosted, cycloid]`
runner (see `.github/workflows/ci.yml`).

**Standard / unit tests** (no backend, no creds; acceptance auto-skips):

```sh
devenv shell -- just test-unit          # go test ./... -short
```

**Acceptance tests** (`TF_ACC=1`) run against a **local docker-compose backend**
(`youdeploy-api` + plugin-manager/-registry + docker-registry + concourse + vault

- db + redis + git-server), not remote staging. Requires `CY_SAAS_API_KEY` (to
  mint `.env` via `just env` → `cy uri interpolate`) and `API_LICENCE_KEY` (first-boot
  licence). Bring the stack up and run:

```sh
devenv shell -- just be-start           # docker compose up -dV (local backend)
devenv shell -- just env                # mint .env from CY_SAAS_API_KEY
devenv shell -- just test-acc           # TF_ACC=1 go test ./... -v
devenv shell -- just test-acc-one TestAccProjectResource   # single test
devenv shell -- just be-stop            # docker compose down -v
# one-shot (reset stack + .env + full suite):
devenv shell -- just test-acc-fresh
```

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

## Docs regeneration

After adding or changing a resource or datasource schema, regenerate the docs with:

```sh
~/go/bin/tfplugindocs generate --examples-dir examples/ --provider-dir . --provider-name cycloid ./..
```

(Or `just docs` if `tfplugindocs` is in your PATH — install it with `just install`.)

Docs are generated, not hand-written. Never edit `docs/` by hand; always re-run the generator before release.

## Cursor Cloud specific instructions

This is a **Go Terraform Provider** for the Cycloid DevOps platform — a single Go
module (not a monorepo). Unit tests need nothing extra; **acceptance tests spin
up a local docker-compose backend** (see "Dev environment & testing" above). The
toolchain comes from devenv — don't hand-install Go/just/etc.

### Services

| Service | Purpose | Command |
|---|---|---|
| Terraform Provider (Go binary) | The only build artifact | `devenv shell -- just build` / `go install .` |
| Local backend stack (acceptance only) | youdeploy-api + plugin/registry + concourse + db/redis/vault/… | `devenv shell -- just be-start` |

### Key commands (inside `devenv shell`)

- **Build**: `just build`
- **Unit tests**: `just test-unit` (`go test ./... -short`)
- **Acceptance**: `just be-start` then `just test-acc` (see the testing section above)
- **Lint**: `golangci-lint run ./...` (and `go vet ./...`)
- **Docs**: `just docs`
- **Install provider**: `go install .` (lands in `~/go/bin/`)

### Running locally with Terraform

To test the provider with Terraform, create a dev override file (see `README.md` and `DEVELOPING_TIPS.md`). The provider binary installed via `go install .` lands in `~/go/bin/`. Point `TF_CLI_CONFIG_FILE` at a `.tfrc` file that uses a `dev_overrides` block referencing that path.

### Gotchas

- Get the toolchain from `devenv shell` (Go 1.25, `just`, `golangci-lint`, `tfplugindocs`, `opentofu`/`terraform`, `jq`, `docker compose`). Don't `curl`-install `just` or rely on a system Go/terraform.
- Acceptance tests run against the **local** docker-compose backend (`just be-start`), not remote staging — never point them at production. They need `CY_SAAS_API_KEY` (mints `.env`) + `API_LICENCE_KEY`.
- `golangci-lint run ./...` is the lint gate (CI runs it); keep it clean.
- Unit tests (`go test ./... -short`) may show `TestAccEnvironmentResource` as FAIL — a pre-existing bug where the test doesn't skip in non-acceptance mode. Use `just test-unit` and ignore that single failure, or `-run 'Test[^A]|TestA[^c]'`.
- Never set `Computed: true` on a `List`/`Set`/`Map` attribute meant to be cleared by omission (scoped resources, tags, permissions, …) — see [DEVELOPING_TIPS.md § Never mark a "collection of things to configure" as Optional + Computed](DEVELOPING_TIPS.md#never-mark-a-collection-of-things-to-configure-as-optional--computed) before adding a new list/set/map attribute to any schema. Shipped twice already (TFPRO-42, `cycloid_organization_api_key`).

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
