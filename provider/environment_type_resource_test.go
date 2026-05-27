package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccEnvironmentTypeResource_basic(t *testing.T) {
	t.Parallel()

	typeName := RandomCanonical("test-env-type")
	updatedName := typeName + "-updated"

	ctx := context.Background()
	orgCanonical := testAccGetOrganizationCanonical()

	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccEnvironmentTypeConfig_basic(orgCanonical, typeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_environment_type.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_environment_type.test", "name", typeName),
					resource.TestCheckResourceAttrSet("cycloid_environment_type.test", "canonical"),
					resource.TestCheckResourceAttrSet("cycloid_environment_type.test", "id"),
				),
			},
			// Import
			{
				ResourceName:      "cycloid_environment_type.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["cycloid_environment_type.test"]
					if rs == nil {
						return "", fmt.Errorf("cycloid_environment_type.test not in state")
					}
					return rs.Primary.Attributes["canonical"], nil
				},
				ImportStateVerifyIgnore: []string{"organization"},
			},
			// Update
			{
				Config: testAccEnvironmentTypeConfig_basic(orgCanonical, updatedName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_environment_type.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_environment_type.test", "name", updatedName),
				),
			},
			// Destroy
			{
				Config:  " ",
				Destroy: true,
			},
		},
	})
}

func testAccEnvironmentTypeConfig_basic(org, name string) string {
	return fmt.Sprintf(`
resource "cycloid_environment_type" "test" {
  organization = "%s"
  name         = "%s"
  color        = "#3498db"
}
`, org, name)
}
