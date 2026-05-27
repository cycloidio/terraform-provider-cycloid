data "cycloid_environment" "shared_prod" {
  canonical = "production"
}

output "shared_prod_clouds" {
  value = data.cycloid_environment.shared_prod.cloud_account_canonicals
}
