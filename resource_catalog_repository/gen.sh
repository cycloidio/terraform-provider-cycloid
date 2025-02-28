#! /usr/bin/env sh

# use from repo root !
test -f ".git" || {
  echo >&2 "error: must be executed from repo root"
  exit 1
}

spec_path=resource_catalog_repository/catalog_spec.json
gen_config_path=resource_catalog_repository/catalog_openapi_gen_config.yml

# Uncomment this if you really need to re-generate from openapi spec
tfplugingen-openapi generate --config "$gen_config_path" \
  --output "$spec_path" \
  openapi.yaml

# Those keywords are in the docs string
sed -i 's/ServiceCatalog/stacks/' "$spec_path"
sed -i 's/ServiceCatalogs/stacks/' "$spec_path"

# Rename all attributes to match our wording in docs
sed -i 's/service_catalog_sources/catalog_repositories/' "$spec_path"
sed -i 's/service_catalog_source/catalog_repository/' "$spec_path"
sed -i 's/service_catalogs/stacks/' "$spec_path"
sed -i 's/service_catalog/stack/' "$spec_path"

tfplugingen-framework generate resources \
  --input "$spec_path" --output "."
