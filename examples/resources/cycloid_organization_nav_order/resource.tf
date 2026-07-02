resource "cycloid_organization_nav_order" "this" {
  organization = "my-org"

  items = [
    {
      type     = "native"
      key      = "dashboard"
      position = 1
    },
    {
      type     = "plugin_widget"
      key      = "42"
      position = 2
    },
    {
      type     = "native"
      key      = "projects"
      position = 3
    },
  ]
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
