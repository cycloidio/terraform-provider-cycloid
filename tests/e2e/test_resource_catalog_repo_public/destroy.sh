#!/bin/false

. "$(_get_step "destroy")"

export CY_API_KEY="${TF_VAR_cycloid_api_key?}"
cy --api-url "http://127.0.0.1:3001" \
  --org "${TF_VAR_cycloid_org?}" \
  cr get --canonical test-catalog-repo 1>/dev/null 2>&1 && _exit_failed 1 "catalog repo should be deleted, and it's not"

true # the tests will use the last return code to decide if it fails or not
