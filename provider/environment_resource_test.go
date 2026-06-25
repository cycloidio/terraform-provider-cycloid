package provider

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"

	middleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccEnvironmentResource provisions the project via dependencies (EnsureTestProject) and only
// manages the environment in the Terraform manifest. Deleting the project (e.g. in Cleanup) results
// in deletion of its environments by the API.
func TestAccEnvironmentResource(t *testing.T) {
	t.Parallel()

	projectName := RandomCanonical("test-project")
	envName := RandomCanonical("test-env")

	ctx := context.Background()
	orgCanonical := testAccGetOrganizationCanonical()

	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	project, err := depManager.EnsureTestProject(ctx, t, orgCanonical, projectName, "Test project for environment testing")
	if err != nil {
		t.Fatalf("Failed to create test project dependency: %v", err)
	}
	projectCanonical := ptr.Value(project.Canonical)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		// Destroying the resource must delete the org-level environment, not just
		// unlink it from the project, otherwise the environment lingers and blocks
		// its environment type from being deleted (ENGBE-279).
		CheckDestroy: func(s *terraform.State) error {
			m := depManager.GetProvider().Middleware
			for _, rs := range s.RootModule().Resources {
				if rs.Type != "cycloid_environment" {
					continue
				}
				canonical := rs.Primary.Attributes["canonical"]
				_, _, err := m.GetOrgEnv(orgCanonical, canonical)
				if err == nil {
					return fmt.Errorf("environment %q still exists after destroy", canonical)
				}
				var apiErr *middleware.APIResponseError
				if !stderrors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
					return fmt.Errorf("unexpected error checking environment %q after destroy: %w", canonical, err)
				}
			}
			return nil
		},
		Steps: []resource.TestStep{
			// Create environment with existing project dependency
			{
				Config: testAccEnvironmentConfig_basic_withDependency(orgCanonical, projectCanonical, envName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_environment.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_environment.test", "project", projectCanonical),
					resource.TestCheckResourceAttr("cycloid_environment.test", "name", envName),
				),
			},
			// Update environment
			{
				Config: testAccEnvironmentConfig_updated_withDependency(orgCanonical, projectCanonical, envName+"-updated"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_environment.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_environment.test", "project", projectCanonical),
					resource.TestCheckResourceAttr("cycloid_environment.test", "name", envName+"-updated"),
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

func testAccEnvironmentConfig_basic_withDependency(org, projectCanonical, env string) string {
	return fmt.Sprintf(`
resource "cycloid_environment" "test" {
  organization = "%s"
  project     = "%s"
  name        = "%s"
}
`, org, projectCanonical, env)
}

func testAccEnvironmentConfig_updated_withDependency(org, projectCanonical, env string) string {
	return fmt.Sprintf(`
resource "cycloid_environment" "test" {
  organization = "%s"
  project     = "%s"
  name        = "%s"
}
`, org, projectCanonical, env)
}
