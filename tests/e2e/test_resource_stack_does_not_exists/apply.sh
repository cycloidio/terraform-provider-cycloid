#! /usr/bin/env false

set +eu

if terraform apply -input=false -auto-approve $(_get_default_vars); then
  _exit_failed 1 "this terraform apply should fail"
fi
