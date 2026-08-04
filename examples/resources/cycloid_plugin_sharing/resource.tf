// Share a plugin with all child organizations.
resource "cycloid_plugin_sharing" "share_all" {
  organization      = "parent-org"
  plugin_install_id = cycloid_plugin.my_plugin.id
  visibility        = "shared"
  mode              = "include"
}

// Share a plugin only with specific child organizations.
resource "cycloid_plugin_sharing" "share_selected" {
  organization      = "parent-org"
  plugin_install_id = cycloid_plugin.my_plugin.id
  visibility        = "shared"
  mode              = "include"
  organizations     = ["child-org-1", "child-org-2"]
}

// Share with all children except specific organizations.
resource "cycloid_plugin_sharing" "share_except" {
  organization      = "parent-org"
  plugin_install_id = cycloid_plugin.my_plugin.id
  visibility        = "shared"
  mode              = "exclude"
  organizations     = ["excluded-org"]
}
