# Filter inventory values from a specific organization
data "cycloid_inventory_values" "shared_vpcs" {
  organization = "parent-organization"
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
    }
  ]
}
