provider "cycloid" {
  # The Cycloid API URL to use.
  url = var.cycloid_api_url
  # The Cycloid API key to use.
  jwt = var.cycloid_api_key
  # Organization canonical points to the organization that is governing all the entities in Cycloid (except users).
  # It's used as a default 'organization_canonical' parameter for all the resources that are created in the Cycloid.
  organization_canonical = var.cycloid_organization
}

terraform {
  required_providers {
    cycloid = {
      source = "registry.terraform.io/cycloidio/cycloid"
    }
  }
}
