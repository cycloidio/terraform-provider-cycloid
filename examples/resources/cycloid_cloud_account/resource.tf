resource "cycloid_credential" "aws_main" {
  name = "AWS main"
  type = "aws"
  body = {
    access_key = var.aws_access_key
    secret_key = var.aws_secret_key
  }
}

resource "cycloid_cloud_account" "aws_main" {
  name                 = "AWS main"
  cloud_provider       = "aws"
  credential_canonical = cycloid_credential.aws_main.canonical
  description          = "Production AWS account, eu-west-1"
}

resource "cycloid_credential" "vsphere" {
  name = "vSphere root"
  type = "custom"
  body = {
    raw = {
      host     = var.vsphere_host
      user     = var.vsphere_user
      password = var.vsphere_password
    }
  }
}

resource "cycloid_cloud_account" "vsphere_homelab" {
  canonical            = "vsphere-homelab"
  name                 = "vSphere homelab"
  cloud_provider       = "vsphere"
  credential_canonical = cycloid_credential.vsphere.canonical
}
