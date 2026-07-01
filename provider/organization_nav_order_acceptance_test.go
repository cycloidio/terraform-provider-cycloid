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
// NOTE: as of writing, this requires the /organizations/{org}/nav backend
// endpoint, which is missing from develop due to a regression (ENGBE-282,
// restore PR youdeploy-http-api#5977). Until that lands (and the local
// docker-compose backend image picks it up), this test will fail with a 404
// against any currently-available backend — could not be run locally.
func TestAccOrganizationNavOrderResource(t *testing.T) {
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
