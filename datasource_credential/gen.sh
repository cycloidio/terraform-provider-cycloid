#! /usr/bin/env sh

set -e

# use from repo root !
test -f ".git" || {
  echo >&2 "error: must be executed from repo root"
  exit 1
}

gen_config_path="datasource_credential/openapi_gen_config.yml"
spec_path="datasource_credential/spec.json"

# # Uncomment this if you really need to re-generate from openapi spec
# tfplugingen-openapi generate --config "$gen_config_path" \
#   --output "$spec_path" \
#   openapi.yaml

tfplugingen-framework generate data-sources \
  --input "./$spec_path" --output "."
