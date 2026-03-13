# Agent Development Notes

See also `agent.md` for API naming conventions, field mapping notes, implementation patterns, and code style guidelines.

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
- The `make install` target installs **codegen tools** (not the provider itself). Use `make install-provider` or `go install .` for the provider binary.
- Code generation (`make tf-generate`) requires `jq` and `curl`, plus `tfplugingen-openapi`/`tfplugingen-framework` installed via `make install`. Do **not** regenerate `out_code_spec.json` from the OpenAPI spec without reading `DEVELOPING_TIPS.md` first.
- All provider operations require a remote Cycloid API (`CY_API_URL`, `CY_API_KEY`, `CY_ORG` env vars). No local services to start.
- `staticcheck` reports pre-existing warnings (unused vars/funcs); these are not blockers.
