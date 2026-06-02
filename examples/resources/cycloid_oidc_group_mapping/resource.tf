resource "cycloid_oidc_group_mapping" "operators" {
  organization   = "my-org"
  group_name     = "FOO_OPERATOR"
  team_canonical = "operator"
}

# Map one group to several teams by declaring multiple mappings.
resource "cycloid_oidc_group_mapping" "operators_lead" {
  organization   = "my-org"
  group_name     = "FOO_OPERATOR"
  team_canonical = "lead"
}
