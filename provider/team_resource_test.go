package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTeamResource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	orgCanonical := testAccGetOrganizationCanonical()
	teamName := RandomCanonical("test-team")
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create team with organization parameter and required roles
			{
				Config: testAccTeamConfig_basic(orgCanonical, teamName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_team.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_team.test", "name", teamName),
					resource.TestCheckResourceAttr("cycloid_team.test", "roles.#", "1"),
					resource.TestCheckResourceAttr("cycloid_team.test", "roles.0", "organization-admin"),
				),
			},
			// Update team
			{
				Config: testAccTeamConfig_updated(orgCanonical, teamName+"-updated"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_team.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_team.test", "name", teamName+"-updated"),
					resource.TestCheckResourceAttr("cycloid_team.test", "roles.#", "1"),
					resource.TestCheckResourceAttr("cycloid_team.test", "roles.0", "organization-admin"),
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
func testAccTeamConfig_basic(org, name string) string {
	return fmt.Sprintf(`
resource "cycloid_team" "test" {
  organization = "%s"
  name         = "%s"
  roles        = ["organization-admin"]
}
`, org, name)
}

func testAccTeamConfig_updated(org, name string) string {
	return fmt.Sprintf(`
resource "cycloid_team" "test" {
  organization = "%s"
  name         = "%s"
  roles        = ["organization-admin"]
}
`, org, name)
}
