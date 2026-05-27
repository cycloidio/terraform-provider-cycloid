data "cycloid_cloud_account" "aws_main" {
  canonical = "aws-main"
}

output "aws_main_credential" {
  value = data.cycloid_cloud_account.aws_main.credential_canonical
}
