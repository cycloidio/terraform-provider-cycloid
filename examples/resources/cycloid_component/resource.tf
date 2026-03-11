# Create project and environment first
resource "cycloid_project" "example" {
  organization = "my-org"
  name         = "infrastructure"
  description  = "Example project for component demos"
}

resource "cycloid_environment" "example" {
  organization = "my-org"
  project     = cycloid_project.example.name
  name         = "production"
}

# Example 1: Full Terraform Control
resource "cycloid_component" "full_control" {
  organization = "my-org"
  project      = cycloid_project.example.name
  environment  = cycloid_environment.example.name
  name         = "My web app"
  
  description  = "Production web application with full Terraform management"
  stack_ref    = "my-org:web-app-stack"
  use_case     = "production"
  stack_version = "v2.1.0"
  
  input_variables = {
    "application" = {
      "config" = {
        "replicas"  = 3
        "cpu_limit" = "1000m"
      }
    }
  }
  
  allow_version_update  = true
  allow_variable_update = true
  allow_destroy         = true
}

# Example 2: Hybrid Control
resource "cycloid_component" "hybrid_control" {
  organization = "my-org"
  project      = cycloid_project.example.name
  environment  = cycloid_environment.example.name
  name         = "web-app-staging"
  
  description  = "Staging web application with hybrid management"
  stack_ref    = "my-org:web-app-stack"
  use_case     = "production"
  stack_version = "v2.1.0"
  
  input_variables = {
    "application" = {
      "config" = {
        "replicas"  = 2
        "cpu_limit" = "500m"
      }
    }
  }
  
  allow_version_update  = true
  allow_variable_update = false
  allow_destroy         = true
}

# Example 3: Clone from Example 2 using current_config
resource "cycloid_component" "cloned_component" {
  organization = "my-org"
  project      = cycloid_project.example.name
  environment  = cycloid_environment.example.name
  name         = "web-app-clone"
  
  description  = "Cloned component using current_config from staging"
  stack_ref    = "my-org:web-app-stack"
  use_case     = "production"
  stack_version = "v2.1.0"
  
  # Clone variables from the staging component's current config
  input_variables = cycloid_component.hybrid_control.current_config
  
  allow_version_update  = false
  allow_variable_update = false
  allow_destroy         = false
}

# Example 4: Development Environment
resource "cycloid_component" "dev_environment" {
  organization = "my-org"
  project      = cycloid_project.example.name
  environment  = cycloid_environment.example.name
  name         = "feature-branch-test"
  
  description  = "Development environment for feature testing"
  stack_ref    = "my-org:web-app-stack"
  use_case     = "production"
  stack_version = "feature/new-auth-system"
  
  input_variables = {
    "application" = {
      "config" = {
        "replicas"  = 1
        "cpu_limit" = "200m"
        "debug_mode" = true
      }
    }
  }
  
  allow_version_update  = true
  allow_variable_update = true
  allow_destroy         = false
}