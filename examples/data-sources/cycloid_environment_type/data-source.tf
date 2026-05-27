data "cycloid_environment_type" "production" {
  canonical = "production"
}

output "production_color" {
  value = data.cycloid_environment_type.production.color
}
