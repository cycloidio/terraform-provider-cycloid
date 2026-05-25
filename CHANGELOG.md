# Changelog

## Unreleased

## v0.7.0

### Added

- **`cycloid_plugin_manager` resource** — manage plugin-manager registration and lifecycle
  (create, import, destroy). Auto-registers the manager with the Cycloid org on create via
  `auto_register = true`; supports `wait_until_connected` to block until the manager
  reports a connected status.
- **`cycloid_plugin_registry` resource** — register and manage plugin registries.
- **`cycloid_plugin_registry_plugin` resource** — associate a plugin with a registry.
- **`cycloid_plugin_version` resource** — manage plugin versions within a registry plugin.
- **`cycloid_plugin` resource** — full plugin install/uninstall lifecycle (pulls image from
  registry, deploys to plugin-manager, tracks running status).
- **`cycloid_organization_members` data source** — list members of a Cycloid organisation
  with optional role filtering.

### Fixed

- `cycloid_plugin_manager`: `auto_register` is now always `true` and not exposed as a
  configurable attribute (CLI-121). Previous versions required an explicit `accept` step
  that left the manager in `invite_pending` if the CLI didn't set `auto_register`.
- `cycloid_plugin_manager`: `status` field correctly handles both string and numeric values
  returned by different backend versions (workaround for v6.10.8-rc backend).
