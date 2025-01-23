#! /usr/bin/env sh

# use from repo root !
test -f "./ci/lib.sh" || {
  echo >&2 "error: must be executed from repo root"
  exit 1
}

. ci/lib.sh

gen_config_path="datasource_stacks/openapi_gen_config.yml"
spec_path="datasource_stacks/spec.json"

# Uncomment this if you really need to re-generate from openapi spec
tfplugingen-openapi generate --config "$gen_config_path" \
  --output "$spec_path" \
  openapi.yaml

# Those keywords are in the docs string
sed -i 's/"name": "data"/"name": "stacks"/' "$spec_path"
sed -i 's/Service Catalog/stacks/' "$spec_path"
sed -i 's/Service Catalogs/stacks/' "$spec_path"
sed -i 's/ServiceCatalog/stacks/' "$spec_path"
sed -i 's/ServiceCatalogs/stacks/' "$spec_path"

# Rename all attributes to match our wording in docs
sed -i 's/service_catalog_sources/catalog_repositories/' "$spec_path"
sed -i 's/service_catalog_source/catalog_repository/' "$spec_path"
sed -i 's/service_catalogs/stacks/' "$spec_path"
sed -i 's/service_catalog/stack/' "$spec_path"

# `stack_own` and `stack_template` parameters are not supported in the middleware
# we should implement them later
result="$(jq -r 'walk(
  if (
    type == "object"
    and (.name == "stack_own" or .name == "stack_template")
  )
  then del(.) else . end
)' "$spec_path" | sed '/null,/d')"
echo "$result" >"$spec_path"

tfplugingen-framework generate data-sources \
  --input "./$spec_path" --output "."
