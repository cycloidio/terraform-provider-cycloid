resource "cycloid_organization_role" "project_viewer_limited" {
  name        = "Project Viewer Limited"
  description = "Read-only project visibility for a scoped set of resources"

  rules = [
    {
      action    = "organization:project:read"
      effect    = "allow"
      resources = ["organization:my-org:project:my-project"]
    },
    {
      action    = "organization:environment:read"
      effect    = "allow"
      resources = ["organization:my-org:project:my-project:environment:*"]
    }
  ]
}
