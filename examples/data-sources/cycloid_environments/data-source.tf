# All environments in the org.
data "cycloid_environments" "catalog" {}

# Only environments linked to one project.
data "cycloid_environments" "billing" {
  project = "billing"
}

output "billing_envs" {
  value = [for e in data.cycloid_environments.billing.environments : e.canonical]
}
