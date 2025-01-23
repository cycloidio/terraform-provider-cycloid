#!/usr/bin/env false
# This script is meant to be sourced, not executed.

terraform plan -input=false -destroy -out="$default_plan_destroy" $(_get_default_vars)
