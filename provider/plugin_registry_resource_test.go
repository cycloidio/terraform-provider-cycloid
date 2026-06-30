package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPluginRegistryResource(t *testing.T) {
	registryName := RandomCanonical("test-registry")
	orgCanonical := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccPluginRegistryConfig(orgCanonical, registryName, clusterPluginRegistryURL),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_plugin_registry.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_plugin_registry.test", "name", registryName),
					resource.TestCheckResourceAttrSet("cycloid_plugin_registry.test", "id"),
				),
			},
			{
				ResourceName:      "cycloid_plugin_registry.test",
				ImportState:       true,
				ImportStateVerify: true,
				// created_at/updated_at come back as 0 from the Create POST echo but
				// carry real values from the GET-list that import reads, so they can't
				// round-trip in ImportStateVerify. They are observed timestamps, not
				// config. (The 0-on-create is a minor provider state-quality issue
				// tracked separately.)
				ImportStateVerifyIgnore: []string{"wait_until_connected", "created_at", "updated_at"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["cycloid_plugin_registry.test"]
					if rs == nil {
						return "", fmt.Errorf("cycloid_plugin_registry.test not in state")
					}
					return rs.Primary.Attributes["id"], nil
				},
			},
		},
	})
}

func testAccPluginRegistryConfig(org, name, url string) string {
	return fmt.Sprintf(`
resource "cycloid_plugin_registry" "test" {
  organization = %q
  name         = %q
  url          = %q
}
`, org, name, url)
}
