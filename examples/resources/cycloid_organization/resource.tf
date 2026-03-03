resource "cycloid_organization" "some_organization" {
  name = "My org"
  // Canonical will be inferred from the name to be a value like
  // canonical = "my-org"
}

// Organization with licence
resource "cycloid_organization" "org_with_licence" {
  name = "Licenced org"
  licence = {
    key = "my_licence_jwt (sensitive!)"
  }
}

// Child organization with a subscription plan
// Please, use the terraform `time_offset` resource to fill the expiration timestamp
resource "time_offset" "one_year" {
  offset_years = 1 // Licence valid 1 year
}

resource "cycloid_organization" "some_organization" {
  name                = "My sub org"
  parent_organization = cycloid_organization.org_with_licence.canonical
  subscription = {
    expires_at_rfc3339 = time_offset.one_year.rfc3339
    plan               = "platform_team"
    members_count      = 5
  }
}

