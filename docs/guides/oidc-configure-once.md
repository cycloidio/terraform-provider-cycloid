---
page_title: "Configure OIDC SSO once across an organization tree - cycloid"
subcategory: ""
description: |-
  Declare teams, roles, OIDC group mappings and per-organization OIDC settings once in Terraform and fan them out across an organization tree.
---

# Configure OIDC SSO once, across an organization tree

This guide shows the "configure once" pattern for wiring an external OIDC
identity provider into Cycloid with Terraform: you declare the SSO integration,
the per-organization reconciliation settings, the teams/roles, and the
group→team mappings **once** in a reusable module, then fan the same declaration
out across every organization in your tree.

It is written for a platform team managing a hierarchy of organizations — a
**parent organization with one child organization per information system (IS)** —
where every organization should share the same OIDC wiring.

~> **Resource availability.** The `cycloid_oidc_integration`,
`cycloid_oidc_organization_settings` and `cycloid_oidc_group_mapping` resources
ship with the OIDC release of the provider. Make sure your provider version
exposes them before applying this guide.

## The building blocks

| Resource | What it configures | Cardinality |
|---|---|---|
| `cycloid_oidc_integration` | The org's AuthenticationOIDC SSO connection: issuer / discovery URL, client id+secret, session TTL, groups claim, TLS options. | One per organization |
| `cycloid_oidc_organization_settings` | Reconciliation policy: the default role granted to OIDC users, strict `oidc_managed` mode, and the no-match policy. | One per organization |
| `cycloid_oidc_group_mapping` | Maps one OIDC group claim value to one team. Declare several mappings to grant a group multiple teams. | N per organization |
| `cycloid_organization_role` / `cycloid_team` | The roles and teams the mappings point at. | N per organization |

Two things to keep in mind:

- The **org-level role** for OIDC users is set on
  `cycloid_oidc_organization_settings.default_role_canonical`, *not* on the
  mapping. Mappings only grant **teams**.
- `oidc_no_match_policy = "eject"` requires `oidc_managed = true` — the API
  rejects the combination otherwise.

## Step 1 — a reusable module

Put the whole OIDC wiring for a single organization in one module. It takes the
organization canonical, the IdP connection details, and the desired mappings as
inputs.

`modules/cycloid-oidc/variables.tf`

```terraform
variable "organization" {
  type        = string
  description = "Organization canonical to configure."
}

variable "issuer" {
  type        = string
  description = "OIDC issuer URL."
}

variable "discovery_url" {
  type        = string
  default     = null
  description = "Optional discovery URL override (split-network setups)."
}

variable "client_id" {
  type = string
}

variable "client_secret" {
  type      = string
  sensitive = true
}

variable "session_ttl_seconds" {
  type        = number
  default     = null
  description = "Session lifetime for OIDC users. null = Cycloid default (7 days)."
}

variable "default_role" {
  type        = string
  description = "Org-level role canonical granted to OIDC-managed users."
}

variable "oidc_managed" {
  type        = bool
  default     = true
  description = "Strict mode: disable local member/team/invite edits."
}

variable "no_match_policy" {
  type        = string
  default     = "eject"
  description = "keep_membership | eject. eject requires oidc_managed = true."
}

# group_name => list of team canonicals
variable "group_mappings" {
  type        = map(list(string))
  default     = {}
  description = "OIDC group claim value mapped to the teams it grants."
}
```

`modules/cycloid-oidc/main.tf`

```terraform
resource "cycloid_oidc_integration" "this" {
  organization        = var.organization
  enabled             = true
  issuer              = var.issuer
  discovery_url       = var.discovery_url
  client_id           = var.client_id
  client_secret       = var.client_secret
  groups_claim_name   = "groups"
  session_ttl_seconds = var.session_ttl_seconds
}

resource "cycloid_oidc_organization_settings" "this" {
  organization           = var.organization
  default_role_canonical = var.default_role
  oidc_managed           = var.oidc_managed
  oidc_no_match_policy   = var.no_match_policy
}

# Flatten {group => [team, team]} into one mapping resource per (group, team).
locals {
  flattened_mappings = flatten([
    for group, teams in var.group_mappings : [
      for team in teams : {
        key   = "${group}/${team}"
        group = group
        team  = team
      }
    ]
  ])
}

resource "cycloid_oidc_group_mapping" "this" {
  for_each = { for m in local.flattened_mappings : m.key => m }

  organization   = var.organization
  group_name     = each.value.group
  team_canonical = each.value.team
}
```

The flattening is the key trick: a single `group_mappings` map of
`group => [teams]` expands into one `cycloid_oidc_group_mapping` per
`(group, team)` pair, which is what the API models.

## Step 2 — fan out across the organization tree

Declare the list of information systems once and loop the module over it with
`for_each`. Shared inputs (issuer, client id/secret) are passed straight
through; per-IS overrides live in the IS object.

`main.tf`

```terraform
locals {
  # The shared IdP connection — declared once.
  idp = {
    issuer        = "https://sso.example.eu"
    client_id     = "cycloid-prod"
    client_secret = var.oidc_client_secret # from a TF_VAR_ / vault, never committed
  }

  # The org tree. Add an IS by adding one entry here.
  information_systems = {
    "org-root" = {
      organization = "org-root"
      default_role = "organization-admin"
      session_ttl  = 7200 # 2h
      mappings = {
        "PLATFORM_ADMIN" = ["admins"]
      }
    }
    "is-alpha" = {
      organization = "is-alpha"
      default_role = "default-project-viewer"
      session_ttl  = 7200 # shorter session for this IS: 2h
      mappings = {
        "ALPHA_OPERATOR" = ["operators", "leads"]
        "ALPHA_VIEWER"   = ["viewers"]
      }
    }
    "is-beta" = {
      organization = "is-beta"
      default_role = "default-project-viewer"
      session_ttl  = null # default (7 days)
      mappings = {
        "BETA_OPERATOR" = ["operators"]
        "BETA_ADMIN"    = ["admins"]
      }
    }
    "is-gamma" = {
      organization = "is-gamma"
      default_role = "default-project-viewer"
      session_ttl  = null
      mappings = {
        "GAMMA_OPERATOR" = ["operators"]
        "GAMMA_READONLY" = ["viewers"]
      }
    }
  }
}

module "oidc" {
  source   = "./modules/cycloid-oidc"
  for_each = local.information_systems

  organization        = each.value.organization
  issuer              = local.idp.issuer
  client_id           = local.idp.client_id
  client_secret       = local.idp.client_secret
  session_ttl_seconds = each.value.session_ttl
  default_role        = each.value.default_role
  group_mappings      = each.value.mappings
}
```

One declaration, applied uniformly to every information system and the root org.
The complexity of "N organizations × M mappings" collapses into one editable
`information_systems` map.

## Day-2 operations

**Add a new information system.** Add one entry to `local.information_systems`
with its organization canonical, default role and mappings, then
`terraform apply`. The module is instantiated for the new IS only.

**Add a mapping (grant a group a team).** Add the team canonical to the group's
list under that IS's `mappings`:

```terraform
"BETA_OPERATOR" = ["operators", "release-managers"] # added release-managers
```

`apply` creates exactly one new `cycloid_oidc_group_mapping`.

**Remove a group.** Delete its key from `mappings`. Because each mapping is a
discrete resource keyed by `(group, team)`, removing the key destroys only those
mappings — no other group is touched.

**Dial the session TTL.** Change `session_ttl` for the IS (e.g. to `7200`
for 2h, or back to `null` for the 7-day default). Only that org's
`cycloid_oidc_integration` is updated.

## Notes

- `client_secret` is **write-only**: the API never returns it, so Terraform
  keeps the value from your config/state and the `has_secret` attribute reflects
  whether a secret is stored server-side. Rotate by changing the variable value.
- Roles and teams referenced by mappings must exist in each organization. Manage
  them with `cycloid_organization_role` and `cycloid_team` — either in this same
  module or a shared one applied first.
- Strict mode (`oidc_managed = true`) disables local membership edits; make sure
  your mappings cover every group that needs access before enabling it, and keep
  a break-glass admin path.
