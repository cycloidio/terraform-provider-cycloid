package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConfigRepositoryResource(t *testing.T) {
	t.Parallel()

	repoName := RandomCanonical("test-config-repo")
	ctx := context.Background()
	orgCanonical := testAccGetOrganizationCanonical()
	cfg := testAccGetTestConfig(t)
	if cfg.Repositories.Config.Credential == "" {
		t.Skip("repositories.config.credential must be set in test_config.yaml for this test")
	}
	repoURL := cfg.Repositories.Config.URL
	credCanonical := cfg.Repositories.Config.Credential
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create config repository with organization_canonical parameter
			{
				Config: testAccConfigRepositoryConfig_basic(orgCanonical, repoName, repoURL, credCanonical),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_config_repository.test", "organization_canonical", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_config_repository.test", "name", repoName),
					resource.TestCheckResourceAttr("cycloid_config_repository.test", "url", repoURL),
					resource.TestCheckResourceAttr("cycloid_config_repository.test", "branch", "main"),
					resource.TestCheckResourceAttr("cycloid_config_repository.test", "credential_canonical", credCanonical),
					resource.TestCheckResourceAttr("cycloid_config_repository.test", "default", "false"),
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

// Test configuration functions (credCanonical from test config)
func testAccConfigRepositoryConfig_basic(org, name, url, credCanonical string) string {
	return fmt.Sprintf(`
resource "cycloid_config_repository" "test" {
  organization_canonical = "%s"
  name                  = "%s"
  url                   = "%s"
  branch                = "main"
  credential_canonical  = "%s"
  default               = false
}
`, org, name, url, credCanonical)
}

func testAccConfigRepositoryConfig_updated(org, name, url, credCanonical string) string {
	return fmt.Sprintf(`
resource "cycloid_config_repository" "test" {
  organization_canonical = "%s"
  name                  = "%s"
  url                   = "%s"
  branch                = "main"
  credential_canonical  = "%s"
  default               = false
}
`, org, name, url, credCanonical)
}
