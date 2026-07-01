resource "cycloid_organization_api_key" "admin_key" {
  name        = "full-admin-key"
  description = "Full admin access, use sparingly"

  rules = [
    {
      action    = "**"
      effect    = "allow"
      resources = []
    },
  ]
}
