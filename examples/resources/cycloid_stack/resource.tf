data "cycloid_stacks" "my_stacks" {
  organization_canonical = var.cycloid_org

  lifecycle {
    postcondition {
      condition     = length(self.stacks) >= 0
      error_message = "org has ${tostring(length(self.stacks))} stacks"
    }
  }
}

locals {
  # you can use a local variable and terraform code to filter stacks
  # Catalog repository only return the canonical of a stack

  # This example filters all stack whose canonical start with `aws_`
  aws_stacks = [for s in data.cycloid_stacks.my_stacks.stacks : s if startswith(s.canonical, "aws")]

  # The datasource returns all the stack informations from the API
  # This example returns all stacks with stackforms enabled
  stackforms_stacks = { for s in data.cycloid_stacks.my_stacks.stacks : s.canonical => s if s.form_enabled }
}

resource "cycloid_stack" "hide_aws" {
  count = length(local.aws_stacks)

  canonical              = local.aws_stacks[count.index].canonical
  organization_canonical = local.aws_stacks[count.index].organization_canonical

  visibility = "hidden"
  team       = ""
}

resource "cycloid_stack" "share_stackforms_stacks" {
  for_each = local.stackforms_stacks

  canonical              = each.key
  organization_canonical = each.value.organization_canonical
  visibility             = "shared"
  team                   = "" # an empty team remove the current team
}
