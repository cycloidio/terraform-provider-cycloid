resource "cycloid_organization" "tf_organization" {
  name = "tforganization"
  organization_canonical = var.cycloid_organization
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
