resource "cycloid_project" "my_project" {
  name = "My project"
  # canonical = "my-project" # if you omit the canonical parameter, the backend will make it from the name.
  description = "Some nice description for my users"
  owner       = "some-team"
}

resource "cycloid_environment" "standard_environments" {
  for_each = toset(["Dev", "Prod", "Staging"])
  name     = each.value
  project  = cycloid_project.my_project.canonical
}

