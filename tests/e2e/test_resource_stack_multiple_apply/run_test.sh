run_step "setup"

# ensure catalog repo is deleted before the tests runs
export CY_API_KEY="${TF_VAR_cycloid_api_key?}"
cy --api-url "http://127.0.0.1:3001" \
  --org "${TF_VAR_cycloid_org?}" \
  cr delete --canonical "catalog-${TF_VAR_cycloid_org?}" 1>/dev/null 2>&1 || true

# We need to create the catalog first using terafomr apply -target
# It's a terraform limitation
_log_warn "catalog plan"
terraform plan -input=false -compact-warnings -out="$default_plan_apply" $(_get_default_vars) -target "cycloid_catalog_repository.catalog_repository_public"
terraform apply -input=false -auto-approve "$default_plan_apply"

# Make aws stack hidden
run_step "plan_apply"
run_step "apply"

# Switch them to local it shoudl work
export TF_VAR_aws_visibility="local"
run_step "plan_apply"
run_step "apply"

run_step "plan_destroy"
run_step "destroy"
run_step "cleanup"
