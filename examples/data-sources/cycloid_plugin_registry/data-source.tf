// Look up a registry managed in a separate workspace (e.g. a shared platform module).
// Reference its ID when creating plugins without owning the registry lifecycle.
data "cycloid_plugin_registry" "shared" {
  organization = "acme-platform"
  name         = "internal-registry"
}

resource "cycloid_plugin_registry_plugin" "my_plugin" {
  organization = data.cycloid_plugin_registry.shared.organization
  registry_id  = data.cycloid_plugin_registry.shared.id
  name         = "my-plugin"
}
