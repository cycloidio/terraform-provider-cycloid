#! /usr/bin/env sh
# shellcheck disable=SC2317
#
# Docker wrapper used to inject required secrets and variable for docker commands

. ci/lib.sh

set -eu
test -n "${CI_DEBUG-}" && set -x

_requires_prog cy jq docker

_register_cmd cmd "docker compose cli wrapper"
cmd() {
  _get_secrets
  docker compose --env-file ".ci/secrets.env" "$@"
}

_register_cmd up "docker compose up with default services"
# shellcheck disable=SC2086
up() {
  cmd up "$@"

}
_register_cmd up_default "docker compose up with default flags for tests"
# shellcheck disable=SC2086
up_default() {
  cmd up -d
}

_register_cmd down "destroy project"
down() {
  cmd down "$@"
}

_register_cmd clean "destroy project and remove volumes"
clean() {
  _clean_secrets
  cmd down -v "$@" -t 2
}

_register_cmd get_secrets "pull secrets required for this repo"
get_secrets() {
  _clean_secrets
  _get_secrets
}

_register_cmd logs "see logs"
logs() {
  cmd logs -f "$@"
}

_register_cmd print_env "dump the env after script evaluation"
print_env() {
  env | grep TEST_
  env | grep ^COMPOSE_
}

_parse_subcommand "$@"
