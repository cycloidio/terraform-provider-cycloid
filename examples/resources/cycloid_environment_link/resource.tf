resource "cycloid_project" "platform" { name = "Platform" }
resource "cycloid_project" "billing"  { name = "Billing" }
resource "cycloid_project" "auth"     { name = "Auth" }

# Primary link: the environment is created and linked to the platform project.
resource "cycloid_environment" "shared_prod" {
  project = cycloid_project.platform.canonical
  name    = "Production"
  type    = "production"
}

# Additional links: the same org-scoped environment is shared with billing and auth.
resource "cycloid_environment_link" "billing_prod" {
  project     = cycloid_project.billing.canonical
  environment = cycloid_environment.shared_prod.canonical
}

resource "cycloid_environment_link" "auth_prod" {
  project     = cycloid_project.auth.canonical
  environment = cycloid_environment.shared_prod.canonical
}
