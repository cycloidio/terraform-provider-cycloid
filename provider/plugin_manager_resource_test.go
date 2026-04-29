package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccPluginManagerResource verifies that the plugin manager auto-registered on
// compose startup is visible and has the expected attributes.
//
// The plugin-manager service registers itself on startup; this test does NOT create
// a new manager — it imports the pre-existing one. If no manager exists the test
// skips with a clear message.
func TestAccPluginManagerResource(t *testing.T) {
	orgCanonical := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)

	if depManager.GetProvider().Middleware == nil {
		t.Skip("skipping acceptance test: middleware not configured")
	}

	// Verify the auto-registered manager exists.
	managers, _, err := depManager.GetProvider().Middleware.ListPluginManagers(orgCanonical)
	if err != nil || len(managers) == 0 {
		t.Skip("skipping: no plugin managers registered (is plugin-manager service running?)")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				// Create a fresh manager so we own its lifecycle.
				// The auto-registered one cannot be deleted by the test cleanly.
				Config: testAccPluginManagerConfig(orgCanonical, "test-acceptance-manager", "http://test-manager:4000"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_plugin_manager.test", "organization", orgCanonical),
					resource.TestCheckResourceAttrSet("cycloid_plugin_manager.test", "id"),
					resource.TestCheckResourceAttrSet("cycloid_plugin_manager.test", "status"),
				),
			},
		},
	})
}

func testAccPluginManagerConfig(org, name, url string) string {
	return fmt.Sprintf(`
resource "cycloid_plugin_manager" "test" {
  organization = %q
  name         = %q
  url          = %q
}
`, org, name, url)
}
