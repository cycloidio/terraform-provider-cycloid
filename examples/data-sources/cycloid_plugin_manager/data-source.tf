// Look up the Plugin Manager registered in an organization (e.g. by the infrastructure team).
// Reference it in outputs or use it to assert the manager is present before installing plugins.
data "cycloid_plugin_manager" "default" {
  organization = "acme-prod"
  name         = "default-plugin-manager"
}

output "plugin_manager_status" {
  value = data.cycloid_plugin_manager.default.status
}

// Install a plugin, making the depends_on explicit via the data source lookup.
resource "cycloid_plugin" "hello" {
  organization      = "acme-prod"
  registry_id       = 42
  plugin_id         = 7
  plugin_version_id = 3

  configuration = {
    greeting = "hello"
    target   = "world"
  }

  depends_on = [data.cycloid_plugin_manager.default]
}
