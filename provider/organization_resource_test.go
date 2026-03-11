package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOrganizationResource_WithAllowDestroy(t *testing.T) {
	t.Parallel()

	// Test constants
	const (
		orgName             = "Test Destroy Organization"
		orgCanonical        = "test-org-destroy"
		orgNameUpdated      = "Test Destroy Organization Updated"
		orgCanonicalUpdated = "test-org-destroy-updated"
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"cycloid": providerserver.NewProtocol6WithError(&CycloidProvider{}),
		},
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationConfig_allowDestroy(orgCanonical, orgName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_organization.test", "name", orgName),
					resource.TestCheckResourceAttr("cycloid_organization.test", "canonical", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_organization.test", "allow_destroy", "true"),
					resource.TestCheckResourceAttr("cycloid_organization.test", "soft_destroy", "false"),
				),
			},
			// Test update with allow_destroy
			{
				Config: testAccOrganizationConfig_allowDestroy(orgCanonicalUpdated, orgNameUpdated),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_organization.test", "name", orgNameUpdated),
					resource.TestCheckResourceAttr("cycloid_organization.test", "canonical", orgCanonicalUpdated),
					resource.TestCheckResourceAttr("cycloid_organization.test", "allow_destroy", "true"),
				),
			},
			// Test destroy with allow_destroy=true should succeed
			{
				Config: " ", // Empty config to trigger destroy
				Check:  resource.ComposeTestCheckFunc(
				// Resource should be destroyed successfully when allow_destroy=true
				),
				Destroy: true,
			},
		},
	})
}

// Test configuration functions
func testAccOrganizationConfig_allowDestroy(canonical, name string) string {
	return fmt.Sprintf(`
resource "cycloid_organization" "test" {
  name         = "%s"
  canonical    = "%s"
  allow_destroy = true
}
`, name, canonical)
}
