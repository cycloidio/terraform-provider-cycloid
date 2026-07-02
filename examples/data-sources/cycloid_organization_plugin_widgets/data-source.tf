data "cycloid_organization_plugin_widgets" "side_menu" {
  organization = "my-org"
  placement    = "sideMenuPage"
}

locals {
  first_side_menu_plugin_widget_id = [
    for widget in data.cycloid_organization_plugin_widgets.side_menu.widgets : widget.id
    if widget.type == "iframe"
  ][0]
}

resource "cycloid_organization_nav_order" "this" {
  organization = "my-org"

  items = [
    {
      type     = "native"
      key      = "dashboard"
      position = 1
    },
    {
      type     = "plugin_widget"
      key      = tostring(local.first_side_menu_plugin_widget_id)
      position = 2
    },
    {
      type     = "native"
      key      = "projects"
      position = 3
    },
  ]
}
