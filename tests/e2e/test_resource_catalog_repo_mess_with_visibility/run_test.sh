#! /usr/bin/env sh

# shellcheck disable=SC2317

run_step setup

for v in "shared" "local" "hidden"; do
  _log_info "Changed visibility to $v"
  export TF_VAR_visibility="$v"
  run_step plan_apply
  run_step apply
done

run_step plan_destroy
run_step destroy

for v in "shared" "local" "hidden"; do
  _log_info "Changed visibility to $v"
  export TF_VAR_visibility="$v"
  run_step plan_apply
  run_step apply

  run_step plan_destroy
  run_step destroy
done

run_step cleanup
