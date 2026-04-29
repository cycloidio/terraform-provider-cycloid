// A plugin manager is the runtime that executes installed plugins. The
// resource registers the manager with the organization and immediately accepts
// the invite (invite_status is always "accepted" once the resource exists).
resource "cycloid_plugin_manager" "default" {
  organization = "my-org"
  name         = "default-plugin-manager"
  url          = "https://plugin-manager.example.com"
}
