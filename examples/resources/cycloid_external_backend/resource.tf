resource "cycloid_credential" "tf_credential_aws" {
  name = "tfcredentialexternalbackend"
  path = "pathexternalbackend"
  type = "aws"
  body = {
    access_key = "AKIAZUHJFEC2CH5YXVGD"
    secret_key = "z1Zgdskagdajkgfsa"
  }
}

resource "cycloid_external_backend" "tf_external_backend" {
  credential_canonical = cycloid_credential.tf_credential_aws.canonical
  default = true
  purpose = "remote_tfstate"
  engine = "aws_storage"
  aws_storage = {
    bucket = "cycloid-tfstate"
    region = "eu-west-1"
    endpoint = "https://s3.eu-west-1.amazonaws.com"
    key = "cycloid-tfstate"
    s3_force_path_style = true
    skip_verify_ssl = true
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
