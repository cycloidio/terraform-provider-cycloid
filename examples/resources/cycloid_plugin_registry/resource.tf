// A plugin registry is a Docker registry that hosts plugin images.
// Cycloid pulls plugin versions from it when you install a plugin.
resource "cycloid_plugin_registry" "internal" {
  organization = "my-org"
  name         = "Internal Docker registry"
  url          = "https://registry.example.com"
}

// Reference a public registry by URL.
resource "cycloid_plugin_registry" "dockerhub" {
  organization = "my-org"
  name         = "Docker Hub"
  url          = "https://index.docker.io"
}
