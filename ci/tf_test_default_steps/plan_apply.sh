#!/usr/bin/env false
# This script is meant to be sourced, not executed.

terraform plan -input=false -out="$default_plan_apply" $(_get_default_vars)
