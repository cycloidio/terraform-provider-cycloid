#!/usr/bin/env false
# This script is meant to be sourced, not executed.

terraform apply -input=false -destroy "$default_plan_destroy" $(_get_default_vars)
