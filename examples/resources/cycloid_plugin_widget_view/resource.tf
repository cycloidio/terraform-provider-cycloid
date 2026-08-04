// Enable a shared plugin's widget view in a child organization.
// The widget view ID can be discovered from the parent plugin's widget views.
resource "cycloid_plugin_widget_view" "enable_dashboard" {
  organization      = "child-org"
  plugin_install_id = 36
  widget_view_id    = 1
  enabled           = true
  url_slug          = "is-management"
}

// Disable an inherited widget view in a child org (creates a local override).
resource "cycloid_plugin_widget_view" "disable_in_child" {
  organization      = "child-org"
  plugin_install_id = 36
  widget_view_id    = 2
  enabled           = false
  url_slug          = "environment-creation"
}
