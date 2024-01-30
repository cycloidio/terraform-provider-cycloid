# Terraform Provider Cycloid

How it works:

```hcl
terraform {
  required_providers {
    cycloid = {
      source = "registry.terraform.io/cycloid/cycloid"
    }
  }
}

resource "cycloid_organization" "child_test" {
  name = "terraform organization test"
}

provider "cycloid" {
  url                    = "URL"
  jwt                    = "JWT"
  organization_canonical = "ORG_CAN"
}
```
