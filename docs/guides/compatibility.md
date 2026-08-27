---
page_title: "Platform compatibility - cycloid"
subcategory: ""
description: |-
  Compatibility matrix between the Cycloid Terraform provider and the Cycloid platform versions.
---

# Platform compatibility

This page lists the Cycloid platform version that was current when each
Terraform provider release was cut. It is a useful reference for on-premise
customers who need to pair the provider with their deployed platform version.

~> **Note:** This table shows the platform version at the time of each provider
release, not a hard minimum. Most provider resources are not version-gated.
Plugin resources (`cycloid_plugin`, `cycloid_plugin_*`) are the main exception
— they may require API endpoints introduced in a specific platform version.

| Provider version | Platform version at release |
|-----------------|---------------------------|
<!-- BEGIN COMPATIBILITY MATRIX -->
| v0.10.5 | v6.20.0 |
| v0.10.4 | v6.18.0 |
| v0.10.3 | v6.17.0 |
| v0.10.2 | v6.16.0 |
| v0.10.1 | v6.10.269 |
<!-- END COMPATIBILITY MATRIX -->

## How to choose

- If your platform version **matches or is newer** than the one listed for a
  provider release, that provider version is safe to use.
- Using an **older provider** against a newer platform is safe (backward-compatible)
  but you won't have access to new resources.
- Using a **newer provider** against an older platform generally works for
  non-plugin resources. Plugin resources may fail if the required API endpoints
  are not yet available on your platform.

## On-premise users with multiple environments

If you run different platform versions across staging and production, pin the
provider version in each workspace:

```terraform
terraform {
  required_providers {
    cycloid = {
      source  = "registry.terraform.io/cycloidio/cycloid"
      version = "= 0.10.3" # platform was v6.17.0 at this release
    }
  }
}
```

~> **This page is auto-generated** at each provider release from the repository
tag history. Do not edit the compatibility table manually.
