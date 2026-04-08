package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
