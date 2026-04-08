package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProjectResource(t *testing.T) {
	t.Parallel()

	projectName := RandomCanonical("test-project")
	const projectDesc = "Test project for acceptance testing"
	ctx := context.Background()
	orgCanonical := testAccGetOrganizationCanonical()
	cfg := testAccGetTestConfig(t)
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create project with organization parameter
			{
				Config: testAccProjectConfig_basic(orgCanonical, projectName, projectDesc, cfg.ConfigRepository),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_project.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_project.test", "name", projectName),
					resource.TestCheckResourceAttr("cycloid_project.test", "description", projectDesc),
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
func testAccProjectConfig_basic(org, name, desc, configRepo string) string {
	return fmt.Sprintf(`
resource "cycloid_project" "test" {
  organization = "%s"
  name         = "%s"
  description  = "%s"
  config_repository = "%s"
}
`, org, name, desc, configRepo)
}

