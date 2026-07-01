resource "cycloid_organization_api_key" "ci_key" {
  name        = "ci-pipeline-key"
  description = "API key used by the CI pipeline"

  rules = [
    {
      action    = "organization:project:read"
      effect    = "allow"
      resources = []
    },
  ]
}

# Persist the token in a Cycloid credential instead of a raw sensitive output,
# so it can be referenced from pipelines without ever leaving Cycloid-managed storage.
resource "cycloid_credential" "ci_key_credential" {
  name = "ci-pipeline-key"
  path = "ci-pipeline-key"
  type = "custom"
  body = {
    raw = {
      token = cycloid_organization_api_key.ci_key.token
    }
  }
}
