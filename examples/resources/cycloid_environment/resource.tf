resource "cycloid_project" "my_project" {
  name        = "My project"
  description = "Some nice description for my users"
  owner       = "some-team"
}

resource "cycloid_environment_type" "prod" {
  name  = "Production"
  color = "#27ae60"
}

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
}

# Minimal environment — only the project link is mandatory.
resource "cycloid_environment" "dev" {
  project = cycloid_project.my_project.canonical
  name    = "Dev"
}

# Environment with the full meta-gov-env feature set.
resource "cycloid_environment" "prod" {
  project     = cycloid_project.my_project.canonical
  name        = "Production"
  type        = cycloid_environment_type.prod.canonical
  owner       = "frederic"
  description = "Customer-facing production environment"

  cloud_account_canonicals = [
    cycloid_cloud_account.aws_main.canonical,
  ]

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
    {
      key       = "api_token"
      type      = "string"
      value     = var.api_token
      sensitive = true
    },
  ]
}

# Bulk creation pattern stays supported.
resource "cycloid_environment" "standard_environments" {
  for_each = toset(["staging", "preprod"])
  canonical = each.value
  name      = each.value
  project   = cycloid_project.my_project.canonical
}
