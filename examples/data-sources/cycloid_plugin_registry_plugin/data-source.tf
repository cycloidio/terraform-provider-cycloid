// Look up an existing plugin catalog entry by name within a known registry.
// Use when the plugin entry is managed elsewhere and you only need to publish
// a new version or reference the plugin ID.
data "cycloid_plugin_registry" "shared" {
  organization = "acme-platform"
  name         = "internal-registry"
}

data "cycloid_plugin_registry_plugin" "hello" {
  organization = "acme-platform"
  registry_id  = data.cycloid_plugin_registry.shared.id
  name         = "hello-world"
}

resource "cycloid_plugin_version" "hello_v2" {
  organization = "acme-platform"
  registry_id  = data.cycloid_plugin_registry_plugin.hello.registry_id
  plugin_id    = data.cycloid_plugin_registry_plugin.hello.id
  url          = "registry.example.com/hello-world:2.0.0"
}
