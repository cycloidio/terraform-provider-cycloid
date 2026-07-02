package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccOrganizationNavOrderResource exercises the full nav-ordering
// lifecycle: create with a scoped ordering, update to a different ordering,
// then reset to defaults (items = []) before destroy.
//
// TODO(ENGBE-282): un-skip once the /organizations/{org}/nav backend endpoint
// ships — restore PR youdeploy-http-api#5977 is still open. Until it merges
// and releases, this test 404s against any currently-available backend. See
// https://linear.app/cycloid/issue/ENGBE-282.
func TestAccOrganizationNavOrderResource(t *testing.T) {
	t.Skip("blocked on ENGBE-282 / youdeploy-http-api#5977 — /organizations/{org}/nav endpoint not released yet")

	orgCanonical := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationNavOrderConfig(orgCanonical, `
    {
      type     = "native"
      key      = "dashboard"
      position = 1
    },
    {
      type     = "native"
      key      = "projects"
      position = 2
    },
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_organization_nav_order.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_organization_nav_order.test", "items.#", "2"),
					resource.TestCheckResourceAttr("cycloid_organization_nav_order.test", "items.0.key", "dashboard"),
				),
			},
			{
				// A second plan with the same config must be clean — no
				// perpetual diff.
				Config: testAccOrganizationNavOrderConfig(orgCanonical, `
    {
      type     = "native"
      key      = "dashboard"
      position = 1
    },
    {
      type     = "native"
      key      = "projects"
      position = 2
    },
`),
				PlanOnly: true,
			},
			{
				// Reset to defaults.
				Config: testAccOrganizationNavOrderConfig(orgCanonical, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_organization_nav_order.test", "items.#", "0"),
				),
			},
		},
	})
}

func testAccOrganizationNavOrderConfig(org, items string) string {
	return fmt.Sprintf(`
resource "cycloid_organization_nav_order" "test" {
  organization = %q
  items = [
%s
  ]
}
`, org, items)
}
