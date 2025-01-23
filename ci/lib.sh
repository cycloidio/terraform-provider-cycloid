#! /usr/bin/env sh
# common tooling for script
# everything is in one file to keep sourcing simple.

test -n "${CI_DEBUG-}" && set -x

# Using error codes to be able to identify the issue
# for external programs
export ERR_PROGRAM_NOT_FOUND=2
export ERR_MISSING_SECRET=2
export ERR_NOT_AT_REPO_ROOT=3
export ERR_REPO_ROOT_NOT_FOUND=4
export ERR_ORG_INIT_FAILED=5
export ERR_ORG_MISSING_API_KEY=6

# Command managmeent
# List of registered commands
COMMANDS=""
# Help text
HELP="
=== ${0:-./dc.sh} help ==="

# <cmd>, <help_text>...
# register a <cmd> where cmd is a function name and add <help_text> to it.
# registered commands will make a function as a script subcommand using _parse_subcommand
_register_cmd() {
  cmd=${1:?cmd name as first arg}
  shift
  help="$*"
  COMMANDS="${COMMANDS} $cmd"

  if [ "${HELP}" != "" ]; then
    HELP="${HELP}
    ${cmd}:
      ${help}
    "
  fi
}

# logging
_log_info() {
  echo >&2 "$(tput setaf 2)info$(tput sgr0): $*"
}

_log_err() {
  echo >&2 "$(tput setaf 1)info$(tput sgr0): $*"
}

_log_warn() {
  echo >&2 "$(tput setaf 3)info$(tput sgr0): $*"
}

# check if we are at the repo's root
_check_repo_root() {
  if [ -f ./ci/lib.sh ] && [ -f .git ]; then
    return 0
  else
    return "$ERR_NOT_AT_REPO_ROOT"
  fi
}

# Exit if your are not on the repo's root
_require_repo_root() {
  _check_repo_root || _exit_failed "$ERR_NOT_AT_REPO_ROOT" "this script '${0}' must be executed from the root of the repository and we are in '$(pwd)'"
}

# return the path to the repo root
_get_repo_root() {
  until _check_repo_root; do
    if [ "$(pwd)" = "/" ]; then
      _exit_failed "$ERR_REPO_ROOT_NOT_FOUND" "failed to get to repository's root, we reached: '$(pwd)'"
    fi

    cd ..
  done

  pwd
}

# exit with <code> and print <msg>
_exit_failed() {
  code=${1:-1}
  shift

  _log_err "$@"
  exit "$code"
} 1>&2

# <prog>
# check if the required programs are installed, exit script it doesn't
_requires_prog() {
  while [ "${1-}" != "" ]; do
    which "$1" >/dev/null || _exit_failed $ERR_PROGRAM_NOT_FOUND "program $1 not found"
    shift
  done
}

# Parse args where $1 is considered a subcommand
# Will execute the subcommand if a $1 match a command registered with _register_cmd
# Args starting from $2 will be passed to the subcommand
# If $1 is `-h`, `--help` or `help` -> print help
_parse_subcommand() {
  for cmd in $COMMANDS; do
    case "${1-}" in
    "$cmd")
      shift
      eval "$cmd" "$*"
      exit $?
      ;;
    "-h" | "--help" | "help")
      help
      exit 0
      ;;
    esac
  done
  _exit_failed $ERR_PROGRAM_NOT_FOUND "Invalid subcommand '${1:-}'.
$(help)"
}

# Fetch required secrets using Cycloid CLI
# This is hardcoded, for now
_get_secrets() {
  root="$(_get_repo_root)"
  file="$root/.ci/secrets.env"
  test -f "$file" && return 0
  mkdir -p "$root/.ci"

  _log_info "fetching backend secrets"
  CY_API_URL=https://http-api.cycloid.io \
    cy credential get \
    --verbosity error --output json \
    --org cycloid --canonical backend-dev-config |
    jq -r '.raw.raw | to_entries[] | "\(.key)=\(.value)"' >"$file"
}

# remove secrets from temp folder
_clean_secrets() {
  root="$(_get_repo_root)"
  rm -f "$root/.ci/secrets.env"
}

# return the standard path to where token for $org should be stored
_get_org_token_path() {
  org=${1?org as first arg}

  mkdir -p "$(_get_repo_root)/.ci/api-keys"
  echo "$(_get_repo_root)/.ci/api-keys/${org}"
}

_register_cmd help "print this help"
help() {
  echo "$HELP"
}

# Load env and secrets
_requires_prog cy

set -o allexport
eval "$(cat ".env")"
set +o allexport

# Fetch licence using cy CLI
if [ "${API_LICENCE_KEY-}" = "" ]; then
  _log_info "fetching the api licence key for backend."

  API_LICENCE_KEY="$(
    CY_API_URL=https://http-api.cycloid.io \
      cy credential get \
      --verbosity error \
      --org cycloid --canonical scaleway-cycloid-backend --output json |
      jq -r .raw.raw.licence_key
  )"

  test -n "$API_LICENCE_KEY" || _exit_failed "$ERR_MISSING_SECRET" "failed to retrieve licence from cycloid."
  export API_LICENCE_KEY
fi
