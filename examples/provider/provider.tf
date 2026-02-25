provider "cycloid" {
  # The Cycloid API URL to use, you can also use the CY_API_URL environment variable.
  api_url = var.cycloid_api_url
  # The Cycloid API key to use, you can also use the CY_API_KEY environment variable.
  api_key = var.cycloid_api_key
  # Organization canonical points to the organization that is governing all the entities in Cycloid (except users).
  # It's used as a default 'organization' parameter for all the resources that are created in the Cycloid.
  # You can also fill this with the CY_ORG environment variable.
  default_organization = var.cycloid_organization

  # Use this parameter only if you have self signed TLS certificates for the API
  # This is not a good practice
  insecure = true
}

terraform {
  required_providers {
    cycloid = {
      source = "registry.terraform.io/cycloidio/cycloid"
    }
  }
}
