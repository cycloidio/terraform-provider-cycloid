package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOrganizationResource_WithAllowDestroy(t *testing.T) {
	rootOrg := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)

	orgCanonical := RandomCanonical("test-org")
	orgName := orgCanonical
	orgCanonicalUpdated := orgCanonical + "-updated"
	orgNameUpdated := orgCanonicalUpdated

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationConfig_allowDestroy(rootOrg, orgCanonical, orgName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_organization.test", "name", orgName),
					resource.TestCheckResourceAttr("cycloid_organization.test", "canonical", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_organization.test", "allow_destroy", "true"),
					resource.TestCheckResourceAttr("cycloid_organization.test", "soft_destroy", "false"),
				),
			},
			{
				Config: testAccOrganizationConfig_allowDestroy(rootOrg, orgCanonicalUpdated, orgNameUpdated),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_organization.test", "name", orgNameUpdated),
					resource.TestCheckResourceAttr("cycloid_organization.test", "canonical", orgCanonicalUpdated),
					resource.TestCheckResourceAttr("cycloid_organization.test", "allow_destroy", "true"),
				),
			},
			{
				Config:  " ",
				Destroy: true,
			},
		},
	})
}

// TestAccOrganizationResource_NoDriftWithoutSubscription is the regression guard
// for TFPRO-49: a cycloid_organization without a subscription block must produce
// an empty plan on the second step (no eternal "subscription will be added" diff).
func TestAccOrganizationResource_NoDriftWithoutSubscription(t *testing.T) {
	rootOrg := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)

	orgCanonical := RandomCanonical("test-drift")
	orgName := orgCanonical

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Step 1: create the org with no subscription block.
			{
				Config: testAccOrganizationConfig_noSubscription(rootOrg, orgCanonical, orgName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_organization.test", "name", orgName),
					resource.TestCheckResourceAttr("cycloid_organization.test", "canonical", orgCanonical),
					// subscription must not be set in state (null).
					resource.TestCheckNoResourceAttr("cycloid_organization.test", "subscription.#"),
				),
			},
			// Step 2: same config, PlanOnly — must produce an empty plan.
			// Any diff here means TFPRO-49 is not fixed.
			{
				Config:             testAccOrganizationConfig_noSubscription(rootOrg, orgCanonical, orgName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config:  " ",
				Destroy: true,
			},
		},
	})
}

func testAccOrganizationConfig_allowDestroy(parentOrg, canonical, name string) string {
	return fmt.Sprintf(`
resource "cycloid_organization" "test" {
  parent_organization = %q
  name                = %q
  canonical           = %q
  allow_destroy       = true
}
`, parentOrg, name, canonical)
}

func testAccOrganizationConfig_noSubscription(parentOrg, canonical, name string) string {
	return fmt.Sprintf(`
resource "cycloid_organization" "test" {
  parent_organization = %q
  name                = %q
  canonical           = %q
  allow_destroy       = true
}
`, parentOrg, name, canonical)
}
