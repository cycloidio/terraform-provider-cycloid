#!/usr/bin/env sh

set -eu

. ci/lib.sh

_require_repo_root

retries=0
until [ "$retries" -gt 10 ]; do
  if [ "$(./ci/dc.sh cmd ps --format json --status running | jq -r '. | select(.Service == "youdeploy-api") | .Health')" = "healthy" ]; then
    exit 0
  fi
  sleep 3
  retries=$(("$retries" + "1"))
done
exit 1
