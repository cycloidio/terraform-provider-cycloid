# Organization-scoped environments are NOT linked to any project. They are
# first-class organization entities; attach them to projects later with
# cycloid_environment_link.

resource "cycloid_environment_type" "prod" {
  name  = "Production"
  color = "#27ae60"
}

# Minimal org-level environment — only a name (or canonical) is required.
resource "cycloid_organization_environment" "dev" {
  name = "Dev"
}

# Org-level environment with the full feature set.
resource "cycloid_organization_environment" "prod" {
  name        = "Production"
  type        = cycloid_environment_type.prod.canonical
  owner       = "org-admin"
  description = "Shared production environment, not bound to any single project"

  variables = [
    {
      key   = "region"
      type  = "string"
      value = "eu-west-1"
    },
    {
      key   = "max_pods"
      type  = "integer"
      value = 110
    },
  ]
}

# Once created, share the org-scoped environment with one or more projects.
resource "cycloid_project" "team_one" {
  name = "Team One"
}

resource "cycloid_environment_link" "team_one_prod" {
  project     = cycloid_project.team_one.canonical
  environment = cycloid_organization_environment.prod.canonical
}
