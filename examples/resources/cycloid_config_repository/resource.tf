resource "cycloid_credential" "tf_credential" {
  name = "tfcredentialconfigrepo"
  path = "pathconfigrepo"
  type = "ssh"
  body = {
    ssh_key = var.credential_ssh_key
  }
}

resource "cycloid_config_repository" "tf_cr_repo" {
  name = "tfconfigrepo"
  credential_canonical = cycloid_credential.tf_credential.canonical
  default = false
  url = var.config_repository_url
  branch = var.config_repository_branch
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
