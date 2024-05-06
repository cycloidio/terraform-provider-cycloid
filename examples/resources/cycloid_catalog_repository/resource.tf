resource "cycloid_credential" "tf_credential_catalog_repo" {
  name = "tfcredentialcatalogrepo"
  path = "pathcatalogrepo"
  type = "ssh"
  body = {
    ssh_key = var.credential_ssh_key
  }
  canonical = "tfcredentialcatalogrepo"
}

resource "cycloid_catalog_repository" "tf_catalog_repository" {
  name = "tfcatalogrepo"
  credential_canonical = cycloid_credential.tf_credential_catalog_repo.canonical
  url = var.catalog_repository_url
  branch = var.catalog_repository_branch
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
