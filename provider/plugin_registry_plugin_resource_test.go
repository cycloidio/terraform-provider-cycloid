package provider

import (
	"fmt"
	"testing"

	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPluginRegistryPluginResource(t *testing.T) {
	orgCanonical := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)

	// Push image so the registry can pull it when needed.
	ensurePluginHelloWorld(t)

	if depManager.GetProvider().Middleware == nil {
		t.Skip("skipping acceptance test: middleware not configured")
	}

	// Create a registry to own the plugin entries.
	registryName := RandomCanonical("testreg")
	registry, _, err := depManager.GetProvider().Middleware.CreatePluginRegistry(
		orgCanonical, registryName, "http://"+clusterRegistryHost,
	)
	if err != nil {
		t.Fatalf("failed to create test plugin registry: %v", err)
	}
	registryID := int(ptr.Value(registry.ID))
	t.Cleanup(func() {
		_, _ = depManager.GetProvider().Middleware.DeletePluginRegistry(orgCanonical, uint32(registryID))
	})

	pluginName := RandomCanonical("hello-world")
	updatedName := pluginName + "-updated"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccPluginRegistryPluginConfig(orgCanonical, registryID, pluginName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_plugin_registry_plugin.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_plugin_registry_plugin.test", "name", pluginName),
					resource.TestCheckResourceAttrSet("cycloid_plugin_registry_plugin.test", "id"),
				),
			},
			{
				Config: testAccPluginRegistryPluginConfig(orgCanonical, registryID, updatedName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_plugin_registry_plugin.test", "name", updatedName),
				),
			},
		},
	})
}

func testAccPluginRegistryPluginConfig(org string, registryID int, name string) string {
	return fmt.Sprintf(`
resource "cycloid_plugin_registry_plugin" "test" {
  organization = %q
  registry_id  = %d
  name         = %q
}
`, org, registryID, name)
}
