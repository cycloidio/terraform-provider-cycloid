#!/usr/bin/env false

# ensure catalog repo gets deleted
export CY_API_KEY="${TF_VAR_cycloid_api_key?}"
cy --api-url "http://127.0.0.1:3001" \
  --org "${TF_VAR_cycloid_org?}" \
  cr delete --canonical test-catalog-repo 1>/dev/null 2>&1 || true
