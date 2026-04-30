package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccPluginManagerResource imports the plugin manager that auto-registers on
// compose startup. The API only allows one manager per org (singleton), so we
// import the existing one rather than creating a fresh one.
func TestAccPluginManagerResource(t *testing.T) {
	orgCanonical := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)

	if depManager.GetProvider().Middleware == nil {
		t.Skip("skipping acceptance test: middleware not configured")
	}

	managers, _, err := depManager.GetProvider().Middleware.ListPluginManagers(orgCanonical)
	if err != nil || len(managers) == 0 {
		t.Skip("skipping: no plugin managers registered (is plugin-manager service running?)")
	}
	managerID := strconv.FormatInt(int64(ptr.Value(managers[0].ID)), 10)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				// Import the auto-registered manager — we don't own Create/Destroy here.
				ResourceName:            "cycloid_plugin_manager.test",
				ImportState:             true,
				ImportStateId:           managerID,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"url", "wait_until_connected"},
				Config:                  testAccPluginManagerImportConfig(orgCanonical),
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

func testAccPluginManagerImportConfig(org string) string {
	return fmt.Sprintf(`
resource "cycloid_plugin_manager" "test" {
  organization = %q
  name         = "placeholder"
  url          = "http://placeholder:4000"
}
`, org)
}
