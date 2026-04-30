// Look up a specific plugin version to install it in one or more organizations.
// This pattern decouples "who publishes the version" from "who installs it".
data "cycloid_plugin_registry" "shared" {
  organization = "acme-platform"
  name         = "internal-registry"
}

data "cycloid_plugin_registry_plugin" "hello" {
  organization = "acme-platform"
  registry_id  = data.cycloid_plugin_registry.shared.id
  name         = "hello-world"
}

data "cycloid_plugin_version" "hello_v1" {
  organization = "acme-platform"
  registry_id  = data.cycloid_plugin_registry.shared.id
  plugin_id    = data.cycloid_plugin_registry_plugin.hello.id
  name         = "hello-world:1.0.0"
}

resource "cycloid_plugin" "hello" {
  organization      = "acme-prod"
  registry_id       = data.cycloid_plugin_version.hello_v1.registry_id
  plugin_id         = data.cycloid_plugin_version.hello_v1.plugin_id
  plugin_version_id = data.cycloid_plugin_version.hello_v1.id

  configuration = {
    greeting = "hello"
    target   = "world"
  }
}
