#!/usr/bin/env false
# This script is meant to be sourced, not executed.

# Remove old tfstates and plan
rm -f terraform.tfstate
rm -f apply.tfplan
rm -f destroy.tfplan

_init_org "$(_get_org_name)"
