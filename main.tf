terraform {
  required_providers {
    cycloid = {
      source = "registry.terraform.io/cycloid/cycloid"
    }
  }
}

resource "cycloid_credential" "test_from_tf" {
  name = "test_from_tf"
  type = "aws"
  body = {
    access_key = "access"
    secret_key = "secret"
  }
  path        = "/test/from/tf"
  description = "so this is kind of working now"
  // TODO: Remove this in favor of the provider one
  organization_canonical = "seraf"
}

provider "cycloid" {
  url                    = "https://api.staging.cycloid.io/"
  jwt                    = "eyJhbGciOiJIUzI1NiIsImtpZCI6IjJmMjEyMmRlLTYzZjItNGVlYy05YzZmLWM2YWJiM2UxZjAwNyIsInR5cCI6IkpXVCJ9.eyJjeWNsb2lkIjp7InVzZXIiOnsiaWQiOjAsInVzZXJuYW1lIjoidGVzdC1hbGwiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJwaWN0dXJlX3VybCI6IiIsImxvY2FsZSI6IiJ9LCJhcGlfa2V5IjoidGVzdC1hbGwiLCJvcmdhbml6YXRpb24iOnsiaWQiOjEyLCJjYW5vbmljYWwiOiJzZXJhZiIsIm5hbWUiOiJDeWNsb2lkIHN0YWdpbmciLCJibG9ja2VkIjpbXSwiaGFzX2NoaWxkcmVuIjpmYWxzZSwic3Vic2NyaXB0aW9uIjp7ImV4cGlyZXNfYXQiOi02MjEzNTU5NjgwMCwicGxhbiI6eyJuYW1lIjoiSW52YWxpZCIsImNhbm9uaWNhbCI6ImludmFsaWQifX0sImFwcGVhcmFuY2UiOnsibmFtZSI6IiIsImNhbm9uaWNhbCI6IiIsImRpc3BsYXlfbmFtZSI6IiIsImxvZ28iOiIiLCJmYXZpY29uIjoiIiwiZm9vdGVyIjoiIiwiaXNfYWN0aXZlIjpmYWxzZSwiY29sb3IiOnsiYiI6MCwiZyI6MCwiciI6MH19fSwiaGFzaCI6Ijg0MTIxMGYwYzZjMjkzODU2NWIzNDBiNjNjNGU0MDc1YzZmOGNkZTUifSwic2NvcGUiOiJhcGkta2V5IiwiYXVkIjoiY3VzdG9tZXIiLCJqdGkiOiJlNjdjM2MxMi0wOTY2LTQwOGQtYmE5My1hYjgyZjI2ZGY3YzgiLCJpYXQiOjE3MDI5ODQ4NTcsImlzcyI6Imh0dHBzOi8vY3ljbG9pZC5pbyIsIm5iZiI6MTcwMjk4NDg1Nywic3ViIjoiaHR0cHM6Ly9jeWNsb2lkLmlvIn0.xReZoSKwYxskgVvfI7hzULc1m6js_GXiy7YDM78yk34"
  organization_canonical = "seraf"
}
