data "cycloid_environment_types" "all" {}

output "env_type_canonicals" {
  value = [for et in data.cycloid_environment_types.all.environment_types : et.canonical]
}
