// Publish a version for a registered plugin. The URL points at the artifact
// (typically a Docker image reference). Cycloid asynchronously pulls and
// validates the artifact; Terraform waits until processing completes.
resource "cycloid_plugin_registry" "internal" {
  organization = "my-org"
  name         = "Internal Docker registry"
  url          = "https://registry.example.com"
}

resource "cycloid_plugin_registry_plugin" "hello_world" {
  organization = "my-org"
  registry_id  = cycloid_plugin_registry.internal.id
  name         = "hello-world"
}

resource "cycloid_plugin_version" "hello_world_v1" {
  organization = "my-org"
  registry_id  = cycloid_plugin_registry.internal.id
  plugin_id    = cycloid_plugin_registry_plugin.hello_world.id
  url          = "registry.example.com/hello-world:1.0.0"
}
