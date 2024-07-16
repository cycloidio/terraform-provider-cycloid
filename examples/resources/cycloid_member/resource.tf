resource "cycloid_organization_member" "tf_org_member" {
  email = "random.email@host.com"
  role_canonical = "organization-admin"
}

provider "cycloid" {
  url                    = var.cycloid_api_url
  jwt                    = var.cycloid_api_key
  organization_canonical = var.cycloid_organization
}

terraform {
  required_providers {
    cycloid = {
      source = "cycloidio/cycloid"
    }
  }
}
