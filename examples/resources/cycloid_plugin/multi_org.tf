// Flavor B — shared catalog, per-org installs.
//
// One registry + plugin + version serves as the shared catalog (declared in a
// "platform" org). Each target org gets its own plugin manager and install.
//
// Note: this pattern requires the upstream plugin-team fix that allows
// registry/plugin/version resources to be shared across organizations.
// Until that fix lands, registry/plugin/version must be declared per-org.

locals {
  target_orgs = ["acme-prod", "acme-staging"]
}

// ── Shared catalog (declared once, in the platform org) ──────────────────────

resource "cycloid_plugin_registry" "shared" {
  organization = "acme-platform"
  name         = "internal-registry"
  url          = "https://registry.example.com"
}

resource "cycloid_plugin_registry_plugin" "hello" {
  organization = "acme-platform"
  registry_id  = cycloid_plugin_registry.shared.id
  name         = "hello-world"
}

resource "cycloid_plugin_version" "hello_v1" {
  organization = "acme-platform"
  registry_id  = cycloid_plugin_registry.shared.id
  plugin_id    = cycloid_plugin_registry_plugin.hello.id
  url          = "registry.example.com/hello-world:1.0.0"
}

// ── Per-org runtime + install ────────────────────────────────────────────────

resource "cycloid_plugin_manager" "default" {
  for_each     = toset(local.target_orgs)
  organization = each.key
  name         = "default-plugin-manager"
  url          = "https://plugin-manager.example.com"
}

resource "cycloid_plugin" "hello" {
  for_each          = toset(local.target_orgs)
  organization      = each.key
  registry_id       = cycloid_plugin_version.hello_v1.registry_id
  plugin_id         = cycloid_plugin_version.hello_v1.plugin_id
  plugin_version_id = cycloid_plugin_version.hello_v1.id

  configuration = {
    greeting = "hello"
    target   = each.key
  }

  depends_on = [cycloid_plugin_manager.default]
}
