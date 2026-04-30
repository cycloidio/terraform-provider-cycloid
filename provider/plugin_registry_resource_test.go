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
				ResourceName:            "cycloid_plugin_registry.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"wait_until_connected"},
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
