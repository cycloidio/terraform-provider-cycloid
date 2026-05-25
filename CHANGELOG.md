# Changelog

## Unreleased

## v0.6.2

### Fixed

- **`cycloid_organization_members` data source**: paginate API requests (`page_size=1000`) so
  organisations with more members than the backend default page size are listed completely.
- **`cycloid_organization_members` data source**: fix docs and example — nested member attribute
  is `role`, not `role_canonical`.

## v0.6.1

### Fixed

- **`cycloid_plugin_manager`**: registration no longer gets stuck in "invite pending" —
  the manager is now auto-registered with the Cycloid organisation on create, with no
  manual accept step required.
- **`cycloid_plugin_manager`**: `status` field now handles both string and numeric values
  returned by different backend versions, preventing plan failures after a backend upgrade.
