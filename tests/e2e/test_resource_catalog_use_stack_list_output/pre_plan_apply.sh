# delete catalog if not deleted on past tests run
org="${TF_VAR_cycloid_org?}"
set +eu
just test-cy "${org}" cr get --canonical "test-catalog-repo-${org}" &&
  just test-cy "${org}" cr delete --canonical "test-catalog-repo-${org}"
set -eu
