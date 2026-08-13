# Filter a single inventory value from a specific organization
data "cycloid_inventory_value" "shared_vpc" {
  organization = "parent-organization"
  select_first = true
  filters = [
    {
      attribute = "provider"
      condition = "eq"
      value     = "aws"
    },
    {
      attribute = "type"
      condition = "eq"
      value     = "aws_vpc"
    },
    {
      attribute = "project_canonical"
      condition = "eq"
      value     = "shared-network"
    },
    {
      attribute = "environment_canonical"
      condition = "eq"
      value     = "prod"
    }
  ]
}
