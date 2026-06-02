resource "cycloid_oidc_organization_settings" "this" {
  organization           = "my-org"
  default_role_canonical = "member"
  oidc_managed           = true
  oidc_no_match_policy   = "eject"
}
