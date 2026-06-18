# cycloid_plugin (Resource)

Installs a plugin version into an organization. Terraform polls until the installation reaches
`running` status (up to 10 minutes); a `failed` status fails the apply.

**All attributes trigger replacement on change.** The API does not support upgrading an installed
plugin in-place — changing the version, registry, or configuration requires uninstall + reinstall.
Note: `create_before_destroy` is not usable here because the API rejects two simultaneous installs
of the same image URL.

`configuration` holds visible key-value pairs shown in plan output (Stack Forms syntax).
`configuration_sensitive` holds secrets — values are masked in plan output. Keys must not overlap
between the two maps; the provider merges them before sending to the API.

> **Terraform import note:** `configuration` and `configuration_sensitive` cannot be recovered
> from the API after import (the API returns a merged map with no way to split). After importing,
> add them to your configuration manually and run `terraform plan` to confirm no replacement is
> triggered.


## Example Usage

### Single organization

```terraform
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

// Installing the plugin. The provider polls until status is "running";
// "failed" status fails the apply. All fields force replacement because
// the API does not support upgrading an installed plugin in place —
// changing the version requires uninstall + reinstall.
//
// Reference registry_id/plugin_id from the version resource to avoid
// repeating the dependency chain.
//
// configuration        — visible key/value pairs (shown in plan output)
// configuration_sensitive — sensitive key/value pairs (masked in plan output)
// Keys must not overlap between the two maps.
resource "cycloid_plugin" "hello_world" {
  organization      = "my-org"
  registry_id       = cycloid_plugin_version.hello_world_v1.registry_id
  plugin_id         = cycloid_plugin_version.hello_world_v1.plugin_id
  plugin_version_id = cycloid_plugin_version.hello_world_v1.id

  configuration = {
    greeting = "hello"
    target   = "world"
  }

  configuration_sensitive = {
    api_token = "my-secret-token"
  }
}
```

### Shared catalog across multiple organizations

One registry + plugin + version serves as the shared catalog. Each target organization gets its
own Plugin Manager and install. This pattern requires the upstream plugin-team fix that allows
registry/plugin/version resources to be shared across organizations; until that lands,
declare the registry/plugin/version per-org.

```terraform
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
```


<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `plugin_id` (Number) The ID of the plugin within the registry.
- `plugin_version_id` (Number) The ID of the plugin version to install. Changing this triggers replacement.
- `registry_id` (Number) The ID of the plugin registry containing the plugin to install.

### Optional

- `configuration` (Map of String) Visible key-value configuration for the plugin (Stack Forms syntax). Values appear in plan output. Can be updated in-place.
- `configuration_sensitive` (Map of String, Sensitive) Sensitive key-value configuration for the plugin (Stack Forms syntax). Values are hidden in plan output. Can be updated in-place. Keys must not overlap with `configuration`.
- `organization` (String) The organization canonical, defaults to the provider `default_organization`.

### Read-Only

- `created_at` (Number) Unix timestamp of install creation.
- `id` (Number) The numeric ID of the installed plugin.
- `status` (String) Installation status: `pending`, `running`, or `failed`.
- `updated_at` (Number) Unix timestamp of last install update.
- `uuid` (String) The UUID of the installed plugin.

## Import

Import by composite ID `<registry_id>:<plugin_id>:<install_id>`:

```shell
terraform import cycloid_plugin.example 42:7:15
```

After importing, add `configuration` and `configuration_sensitive` to your config manually —
they cannot be read back from the API.
