resource "cycloid_environment_type" "qa" {
  name  = "QA"
  color = "#9b59b6"
}

resource "cycloid_environment_type" "preprod" {
  canonical = "preprod"
  name      = "Pre-production"
  color     = "#f39c12"
}

resource "cycloid_environment_type" "production_hardened" {
  name  = "Production"
  color = "#27ae60"

  label_selector = {
    enforcement = "hard"

    requirements = [
      {
        key      = "env-type"
        operator = "in"
        values   = ["production"]
      }
    ]
  }
}
