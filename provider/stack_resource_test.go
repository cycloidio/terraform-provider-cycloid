package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStackResource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	orgCanonical := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccStackConfig_basic(orgCanonical),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_stack.test", "organization_canonical", orgCanonical),
					resource.TestCheckResourceAttrSet("cycloid_stack.test", "canonical"),
					resource.TestCheckResourceAttr("cycloid_stack.test", "visibility", "local"),
				),
			},
			{
				Config: testAccStackConfig_updated(orgCanonical),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_stack.test", "organization_canonical", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_stack.test", "visibility", "shared"),
				),
			},
			{
				Config:  " ",
				Destroy: true,
			},
		},
	})
}

func testAccStackConfig_basic(org string) string {
	return fmt.Sprintf(`
data "cycloid_stacks" "existing" {
  organization_canonical = %q
}

resource "cycloid_stack" "test" {
  organization_canonical = %q
  canonical              = data.cycloid_stacks.existing.stacks[0].canonical
  visibility             = "local"
}
`, org, org)
}

func testAccStackConfig_updated(org string) string {
	return fmt.Sprintf(`
data "cycloid_stacks" "existing" {
  organization_canonical = %q
}

resource "cycloid_stack" "test" {
  organization_canonical = %q
  canonical              = data.cycloid_stacks.existing.stacks[0].canonical
  visibility             = "shared"
}
`, org, org)
}
