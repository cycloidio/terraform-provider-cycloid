export CY_API_KEY="$TF_VAR_cycloid_api_key"
export CY_ORG="$TF_VAR_cycloid_org"

cy catalog-repository get --canonical cycloid_community_catalog 2>/dev/null ||
  cy catalog-repository create \
    --name cycloid_community_catalog \
    --url https://github.com/cycloid-community-catalog/public-stacks.git \
    --branch master
