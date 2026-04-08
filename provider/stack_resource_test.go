package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStackResource(t *testing.T) {
	t.Parallel()
	t.Skip("TestAccStackResource will be fixed in a separate PR")

	const stackName = "web-app-stack"
	ctx := context.Background()
	orgCanonical := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Manage existing stack with organization_canonical parameter
			{
				Config: testAccStackConfig_basic(orgCanonical, stackName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_stack.test", "organization_canonical", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_stack.test", "canonical", stackName),
				),
			},
			// Update stack visibility/team
			{
				Config: testAccStackConfig_updated(orgCanonical, stackName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_stack.test", "organization_canonical", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_stack.test", "canonical", stackName),
					resource.TestCheckResourceAttr("cycloid_stack.test", "visibility", "shared"),
				),
			},
			// Destroy testing
			{
				Config:  " ", // Empty config to trigger destroy
				Destroy: true,
			},
		},
	})
}

// Test configuration functions
func testAccStackConfig_basic(org, stack string) string {
	return fmt.Sprintf(`
data "cycloid_stacks" "existing" {
  organization_canonical = "%s"
}

resource "cycloid_stack" "test" {
  organization_canonical = "%s"
  canonical              = data.cycloid_stacks.existing.stacks[0].canonical
  visibility             = "local"
}
`, org, org)
}

func testAccStackConfig_updated(org, stack string) string {
	return fmt.Sprintf(`
data "cycloid_stacks" "existing" {
  organization_canonical = "%s"
}

resource "cycloid_stack" "test" {
  organization_canonical = "%s"
  canonical              = data.cycloid_stacks.existing.stacks[0].canonical
  visibility             = "shared"
  team                   = "admin-team"
}
`, org, org)
}
