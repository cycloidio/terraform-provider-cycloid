# Project command runner (replaces Makefile for most targets).
# See: https://just.systems/man/en/
# Load .env so CY_ORG, CY_API_URL, CY_API_KEY etc. are available.

set dotenv-load := true

# Export Terraform provider env vars for plan/apply/destroy (from .env or CY_*)

export TF_VAR_cycloid_org := env("CY_ORG", "")
export TF_VAR_cycloid_api_url := env("CY_API_URL", "")
export TF_VAR_cycloid_api_key := env("CY_API_KEY", "")

# Show this help (default recipe)
[default]
help:
    @just --list

# Build the provider
build *args:
    go build -gcflags 'all=-l' -trimpath {{ args }}

# Run unit tests only (no TF_ACC; acceptance tests are skipped)
test-unit:
    go test ./... -v -short

# Mint .env from .env.sample using cy uri interpolate (requires CY_SAAS_API_KEY).
# Mirrors cycloid-cli's `make .env` target.
env:
    @rm -f .env || true
    @CY_API_KEY=$${CY_SAAS_API_KEY?A valid API key to the cycloid org in cycloid SaaS is required. Set CY_SAAS_API_KEY.} \
        CY_API_URL=https://http-api.cycloid.io \
        CY_ORG=cycloid \
        cy uri interpolate .env.sample > .env

# Run all tests including acceptance (requires CY_API_URL, CY_API_KEY, CY_ORG or .env)
test-acc:
    TF_ACC=1 go test ./... -v

# Run a single acceptance test by name (e.g. just test-acc-one TestAccProjectResource)
test-acc-one TEST:
    TF_ACC=1 go test -v -run '^{{ TEST }}$' ./provider/...

# One-shot: bring up a fresh stack, mint .env, run the full acceptance suite.
# Requires CY_SAAS_API_KEY and API_LICENCE_KEY to be set in the environment.
test-acc-fresh: be-reset env
    sleep 10
    TF_ACC=1 go test ./provider/... -count=1 -timeout 30m -v

# --- Local backend (mirror of cycloid-cli compose stack) ---
# Bring up the full local stack: youdeploy-api + plugin-manager + plugin-registry
# + docker-registry + concourse + vault + db + redis + smtp + git-server.
# CY_API_URL defaults to http://localhost:3001 once the stack is healthy.
be-start:
    docker compose up -dV

be-stop:
    docker compose down -v

be-reset: be-stop be-start

be-pull:
    docker compose pull

# Tail logs from plugin-manager / plugin-registry (the most common debug targets)
be-logs SERVICE="plugin-manager":
    docker compose logs -f {{ SERVICE }}

# Run unit tests (default; use test-acc for acceptance tests)
test: test-unit

# Convert swagger
convert-swagger:
    go run ./swagger_converter exec

# Regenerate provider spec and models (run convert-swagger first)
tf-generate:
    # Some datasources / credentials have separate codegen scripts
    ./datasource_credential/gen.sh
    ./datasource_credentials/gen.sh
    ./datasource_stacks/gen.sh
    ./resource_catalog_repository/gen.sh
    tfplugingen-framework generate resources --input ./out_code_spec.json --output .
    tfplugingen-framework generate data-sources --input ./out_code_spec.json --output .

# Install codegen and doc tools
install:
    go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

# Install the provider binary
install-provider:
    go install .

# Run terraform plan (requires install-provider and CY_* in .env)
plan: install-provider
    terraform plan

# Run terraform apply -auto-approve
apply: install-provider
    terraform apply -auto-approve

# Run terraform destroy -auto-approve
destroy: install-provider
    terraform destroy -auto-approve

# Generate provider docs
docs:
    tfplugindocs generate --examples-dir examples/ --provider-dir . --provider-name cycloid ./..

# Replace the line below with your usual playground command (e.g. terraform plan, apply, or a script).
playground:
    terraform plan

# Watch and re-run playground on file changes
watch:
    watchexec -w . -w justfile -e tf -e sh -e go -c -r -- just playground
