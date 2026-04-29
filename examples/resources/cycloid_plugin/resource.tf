// Zero-to-running plugin: the four resources below cover the full lifecycle
// from registering a Docker registry to installing the plugin in the
// organization.
//
//   1. cycloid_plugin_registry           — declare the registry that hosts the artifact
//   2. cycloid_plugin_registry_plugin    — declare the plugin entry inside the registry
//   3. cycloid_plugin_version            — publish the version (artifact URL)
//   4. cycloid_plugin                    — install the plugin in the organization
//
// A `cycloid_plugin_manager` is also required at the org level (see its own
// example); it is the runtime that executes installed plugins.

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

// Installing the plugin. The provider polls the install until status is
// "running"; a "failed" status fails the apply. All fields force replacement
// because the API does not support upgrading an installed plugin in place —
// changing the version requires uninstall + reinstall.
resource "cycloid_plugin" "hello_world" {
  organization      = "my-org"
  registry_id       = cycloid_plugin_registry.internal.id
  plugin_id         = cycloid_plugin_registry_plugin.hello_world.id
  plugin_version_id = cycloid_plugin_version.hello_world_v1.id

  // Plugin configuration (Stack Forms key/value pairs). Values are sensitive
  // and never echoed in plan output.
  configuration = {
    greeting = "hello"
    target   = "world"
  }
}
