# OIDC integration for a split-network setup where the identity provider is
# only reachable via an internal discovery URL rather than the public issuer.
#
# The client_secret is write-only: Terraform tracks it in state but the API
# never returns it. Change the value here to rotate the secret; use the
# has_secret attribute to confirm a secret is stored on the server.

resource "cycloid_oidc_integration" "this" {
  organization = "my-org"
  enabled      = true

  issuer        = "https://sso.internal.example.com"
  discovery_url = "https://sso.internal.example.com/.well-known/openid-configuration"
  client_id     = "cycloid-prod"
  client_secret = var.oidc_client_secret

  groups_claim_name   = "groups"
  session_ttl_seconds = 28800 # 8 hours
}
