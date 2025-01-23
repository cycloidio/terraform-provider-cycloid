#!/usr/bin/env false
# This script is meant to be sourced, not executed.

terraform apply -input=false -auto-approve $(_get_default_vars)
