resource "cycloid_environment_type" "qa" {
  name  = "QA"
  color = "#9b59b6"
}

resource "cycloid_environment_type" "preprod" {
  canonical = "preprod"
  name      = "Pre-production"
  color     = "#f39c12"
}
