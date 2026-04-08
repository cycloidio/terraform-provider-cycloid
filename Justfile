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

# Run all tests including acceptance (requires CY_API_URL, CY_API_KEY, CY_ORG)
test-acc:
    TF_ACC=1 go test ./... -v

# Run a single acceptance test by name (e.g. just test-acc-one TestAccProjectResource)
test-acc-one TEST:
    TF_ACC=1 go test -v -run '^{{ TEST }}$' ./provider/...

# Run all tests (same as previous make test: unit + acceptance)
test: test-acc

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
