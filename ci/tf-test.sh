#! /usr/bin/env sh
#
# shellcheck disable=SC2317
# shellcheck disable=SC2046
# shellcheck disable=SC1090
# shellcheck disable=SC2016

. ./ci/lib.sh

set -eu

test -n "${CI_DEBUG-}" && set -x

HELP='
USAGE:
TF_TEST_TARGET_DIR=<tf_test_dir> '"${0-./ci/tf-test.sh}"' <command> ...<args>

DESCRIPTION:
This script implement the way we run e2e tests for the terraform provider.
Must be executed from repo'\''s root

`run_test` will execute each test case with those steps in that order:
1 - setup
2 - init
3 - pre_plan_apply
4 - plan_apply
5 - post_plan_apply
6 - apply
7 - pre_plan_destroy
8 - plan_destroy
9 - post_plan_destroy
10 - destroy
11 - cleanup

You can override a step by putting a \${step_name}.sh script in the folder of your test, in that case, the script will be sourced in place of the default behavior.

You can also override the `run_test` order by overriding the command by making a `run_test.sh` script in your test folder.

The scripts must be executable to be used as override. It will have access to the same context as the _override_step function in this script.

If a vars.tfvars is set on the current repo, it will be added as a -var-file arg to the default steps.

By default, you test will have a cycloid child org create with from the folder name without the leading `test_` and `_` will be replaced to `-`.
  > Exemple: from a `test_resource_stack_visibiltiy` folder name, you will have an org named `resource-stack-visibility` org.

/!\\ The org is created at the `setup` stage, so if you override this step, be careful.

The API_KEY and org will be passed to terraform via the `TF_VAR_cycloid_api_key` and `TF_VAR_cycloid_org` variable.
'

# those are default values used in steps
export default_plan_apply="apply.tfplan"
export default_plan_destroy="destroy.tfplan"

# <TF_TEST_TARGET_DIR> -> <org_name>
# Generate the org name from the test dir.
# Removes the leading `test_` and replace `_` with `-`
_get_org_name() {
  testDir=$(basename "${TF_TEST_TARGET_DIR?target dir is required}")

  if [ -f "${TF_TEST_TARGET_DIR}/org" ]; then
    org="$(cat "${TF_TEST_TARGET_DIR}/org")"
    _log_warn "org overriden as ${org}"
  else
    org="$(echo "$testDir" | cut -f2- -d'_' | sed 's/_/-/g')"
    _log_info "using org $org"
  fi

  echo "$org"
}

# <org> -> <TF_VAR_cycloid_api_key> <TF_VAR_cycloid_org>
# Create an <org>, licence it and generate the related API Key
_init_org() {
  org="${1?org as first arg}"
  repoRoot="${REPO_ROOT?}"
  tokenFile="$(_get_org_token_path "${org}")"

  (
    cd "$repoRoot"
    ./ci/init-backend.sh init_child_org "$org"
  ) || _exit_failed "$ERR_ORG_INIT_FAILED" "failed to init org '$org' for tests."

  if [ "$(pwd)" = "$repoRoot" ]; then
    _exit_failed 1 "failed to cd back to test"
  fi

  test -f "$tokenFile" || _exit_failed "$ERR_ORG_MISSING_API_KEY" "missing api key for ${org} at '$tokenFile', check init script logs."

  TF_VAR_cycloid_api_key="$(cat "$tokenFile")"
  export TF_VAR_cycloid_api_key
  export TF_VAR_cycloid_org="$org"
}

# <TF_TEST_TARGET_DIR> <step_name> -> <path>
# _get_override_script will look for a <step>.sh script in the test folder and return its path.
_find_override_step() {
  if script="$(realpath "${REPO_ROOT?}/${TF_TEST_TARGET_DIR?}/${1?step name as first arg}.sh")"; then
    echo "$script"
  else
    echo ""
  fi
}

# <step_name> -> <script_path>
# _get_step will look for a <step>.sh script in the ./ci/tf_test_default_step folder
# and return its path.
_get_step() {
  if script="$(realpath "${REPO_ROOT?}/ci/tf_test_default_steps/${1?step name as first arg}.sh" 2>/dev/null)"; then
    echo "$script"
  else
    echo ""
  fi
}

# <TF_TEST_TARGET_DIR> -> <tf_args>
# _get_default_vars will search for a `vars.tfvars` on the test folder and return
# the terraform -var-file <path> arg.
_get_default_vars() {
  var_file="$(realpath "${TF_TEST_TARGET_DIR}/vars.tfvars" 2>/dev/null)"
  test -f "$var_file" && echo '-var-file "'"${var_file}"'"'
}

_register_cmd "run_step" "run a specific step"
run_step() {
  step=${1?step first arg}
  overrided_step="$(_find_override_step "$step")"
  if [ -f "$overrided_step" ]; then
    _log_warn "override step '$step' with script '$overrided_step'."
    . "$overrided_step"
    return "$?"
  fi

  default_step="$(_get_step "$step")"
  if [ -f "$default_step" ]; then
    _log_info "running default step '$step'"
    . "$default_step"
    return $?
  fi

  _log_info "no actions for step '$step'"
}

_register_cmd "run_test" "run the full test in all step order"
run_test() {
  script="$(_find_override_step "run_test")"
  if [ -f "$script" ]; then
    . "$script"
    return "$?"
  fi

  # Check if the use override step order
  custom_steps="$REPO_ROOT/$TF_TEST_TARGET_DIR/steps"
  if [ -f "$custom_steps" ]; then
    _log_warn "override step order with the one contained here '$custom_steps'"
    TF_TEST_STEPS="$(cat "$REPO_ROOT/$TF_TEST_TARGET_DIR/steps")"
    _log_warn "step order: '$TF_TEST_STEPS'"
  fi

  for step in $TF_TEST_STEPS; do
    run_step "$step"
  done
}

case "${1--h}" in "-h" | "--help" | "help")
  help
  exit 1
  ;;
esac

_require_repo_root
REPO_ROOT=$(_get_repo_root)

mkdir -p .ci
cat <<EOF >.ci/dev.tfrc
provider_installation {
  dev_overrides {
    "registry.terraform.io/cycloidio/cycloid" = "$REPO_ROOT"
  }

  direct {}
}
EOF

TF_CLI_CONFIG_FILE="$(realpath .ci/dev.tfrc 2>/dev/null)" || _exit_failed 1 "failed to setup dev.tfrc file for testing"
export TF_CLI_CONFIG_FILE

TF_TEST_STEPS=${TF_TEST_STEPS-setup init pre_plan_apply plan_apply post_plan_apply apply pre_plan_destroy plan_destroy post_plan_destroy destroy cleanup}
TF_TEST_TARGET_DIR=${TF_TEST_TARGET_DIR?set the terraform test dir with TF_TEST_TARGET_DIR env var}
cd "$TF_TEST_TARGET_DIR" || _exit_failed 1 "cannot change dir to '$TF_TEST_TARGET_DIR'."
_parse_subcommand "$@"
