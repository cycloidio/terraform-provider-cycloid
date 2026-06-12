resource "cycloid_organization_api_key" "ci_key" {
  name        = "ci-pipeline-key"
  description = "API key used by the CI pipeline"

  rules = [
    {
      action    = "organization:project:read"
      effect    = "allow"
      resources = []
    },
    {
      action    = "organization:pipeline:read"
      effect    = "allow"
      resources = []
    },
  ]
}

# The token is only returned at creation time. Capture it at apply time;
# subsequent reads return an empty token from the API.
output "ci_api_key_token" {
  value     = cycloid_organization_api_key.ci_key.token
  sensitive = true
}

provider "cycloid" {
  url                    = var.cycloid_api_url
  jwt                    = var.cycloid_api_key
  organization_canonical = var.cycloid_organization
}

terraform {
  required_providers {
    cycloid = {
      source = "cycloidio/cycloid"
    }
  }
}
