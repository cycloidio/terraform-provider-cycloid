// Declare a plugin entry inside a registry. The registry is the storage; this
// resource is the catalog entry users can install from.
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
