provider "cycloid" {
  url                    = var.cycloid_api_url
  jwt                    = var.cycloid_api_key
  organization_canonical = var.cycloid_organization
}

terraform {
  required_providers {
    cycloid = {
      source = "registry.terraform.io/cycloidio/cycloid"
    }
  }
}

# For resource specific examples look into the examples directory
