# Changelog

## Unreleased

## v0.6.1

### Fixed

- **`cycloid_plugin_manager`**: registration no longer gets stuck in "invite pending" —
  the manager is now auto-registered with the Cycloid organisation on create, with no
  manual accept step required.
- **`cycloid_plugin_manager`**: `status` field now handles both string and numeric values
  returned by different backend versions, preventing plan failures after a backend upgrade.
