package provider

import (
	"fmt"
	"testing"

	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPluginVersionResource(t *testing.T) {
	orgCanonical := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)

	if depManager.GetProvider().Middleware == nil {
		t.Skip("skipping acceptance test: middleware not configured")
	}

	// Push image first — version publish pulls it from docker-registry.
	imageRef := ensurePluginHelloWorld(t)

	m := depManager.GetProvider().Middleware

	// Bootstrap: registry → plugin (prerequisites the TF resource depends on).
	registry, _, err := m.CreatePluginRegistry(orgCanonical, RandomCanonical("testreg"), clusterPluginRegistryURL)
	if err != nil {
		t.Fatalf("failed to create test registry: %v", err)
	}
	registryID := uint32(ptr.Value(registry.ID))
	t.Cleanup(func() { _, _ = m.DeletePluginRegistry(orgCanonical, registryID) })

	plugin, _, err := m.CreateRegistryPlugin(orgCanonical, registryID, RandomCanonical("hello-world"))
	if err != nil {
		t.Fatalf("failed to create test plugin: %v", err)
	}
	pluginID := uint32(ptr.Value(plugin.ID))
	t.Cleanup(func() { _, _ = m.DeleteRegistryPlugin(orgCanonical, registryID, pluginID) })

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				// Publish version 1.0.0 — image tag must match \d+\.\d+\.\d+.
				Config: testAccPluginVersionConfig(orgCanonical, int(registryID), int(pluginID), imageRef),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_plugin_version.test", "organization", orgCanonical),
					resource.TestCheckResourceAttrSet("cycloid_plugin_version.test", "id"),
					resource.TestCheckResourceAttrSet("cycloid_plugin_version.test", "status"),
				),
			},
			{
				ResourceName:      "cycloid_plugin_version.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["cycloid_plugin_version.test"]
					if rs == nil {
						return "", fmt.Errorf("cycloid_plugin_version.test not in state")
					}
					return fmt.Sprintf("%s:%s:%s",
						rs.Primary.Attributes["registry_id"],
						rs.Primary.Attributes["plugin_id"],
						rs.Primary.Attributes["id"],
					), nil
				},
			},
		},
	})
}

func testAccPluginVersionConfig(org string, registryID, pluginID int, imageURL string) string {
	return fmt.Sprintf(`
resource "cycloid_plugin_version" "test" {
  organization = %q
  registry_id  = %d
  plugin_id    = %d
  url          = %q
}
`, org, registryID, pluginID, imageURL)
}
