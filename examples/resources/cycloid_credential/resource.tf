resource "cycloid_credential" "tf_credential" {
  name = "tfcredential"
  path = "pathcredential"
  type = "ssh"
  body = {
    ssh_key = var.credential_ssh_key
  }
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
