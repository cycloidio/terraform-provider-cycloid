data "cycloid_cloud_accounts" "aws" {
  cloud_provider = "aws"
}

output "aws_account_canonicals" {
  value = [for ca in data.cycloid_cloud_accounts.aws.cloud_accounts : ca.canonical]
}
