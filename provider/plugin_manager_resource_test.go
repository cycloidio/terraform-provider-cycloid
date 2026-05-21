package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
				ImportStateVerifyIgnore: []string{"url", "wait_until_connected", "auto_register"},
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

// TestAccPluginManagerResource_AutoRegister exercises the full create/destroy path
// with auto_register=true (default). The API allows one manager per org, so the
// test clears existing registrations first and restores the compose fixture in cleanup.
func TestAccPluginManagerResource_AutoRegister(t *testing.T) {
	orgCanonical := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)
	m := depManager.GetProvider().Middleware

	if m == nil {
		t.Skip("skipping acceptance test: middleware not configured")
	}

	deleteAllPluginManagers(t, m, orgCanonical)
	t.Cleanup(func() {
		restoreClusterPluginManager(t, m, orgCanonical)
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccPluginManagerConfigWithAutoRegister(
					orgCanonical,
					clusterTestPluginManager,
					clusterPluginManagerURL,
					true,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_plugin_manager.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_plugin_manager.test", "name", clusterTestPluginManager),
					resource.TestCheckResourceAttr("cycloid_plugin_manager.test", "auto_register", "true"),
					resource.TestCheckResourceAttrSet("cycloid_plugin_manager.test", "id"),
					resource.TestCheckResourceAttrSet("cycloid_plugin_manager.test", "status"),
					testAccCheckPluginManagerInviteAccepted(orgCanonical, m),
				),
			},
		},
	})
}

func deleteAllPluginManagers(t *testing.T, m middleware.Middleware, org string) {
	t.Helper()

	managers, _, err := m.ListPluginManagers(org)
	if err != nil {
		t.Fatalf("failed to list plugin managers before test setup: %v", err)
	}
	for _, mgr := range managers {
		if mgr.ID == nil {
			continue
		}
		if _, err := m.DeletePluginManager(org, *mgr.ID); err != nil {
			t.Fatalf("failed to delete plugin manager %d during test setup: %v", *mgr.ID, err)
		}
	}
}

func restoreClusterPluginManager(t *testing.T, m middleware.Middleware, org string) {
	t.Helper()

	managers, _, err := m.ListPluginManagers(org)
	if err == nil {
		for _, mgr := range managers {
			if mgr.ID == nil {
				continue
			}
			if _, delErr := m.DeletePluginManager(org, *mgr.ID); delErr != nil {
				t.Logf("cleanup: failed to delete plugin manager %d: %v", *mgr.ID, delErr)
			}
		}
	}

	managers, _, err = m.ListPluginManagers(org)
	if err == nil {
		for _, mgr := range managers {
			if mgr.Name != nil && *mgr.Name == clusterTestPluginManager {
				if mgr.InviteStatus != nil {
					switch *mgr.InviteStatus {
					case "accepted", "invite_accepted":
						return
					}
				}
			}
		}
	}

	_, _, err = m.CreatePluginManager(org, clusterTestPluginManager, clusterPluginManagerURL, true)
	if err != nil {
		t.Logf("cleanup: failed to restore compose plugin manager: %v", err)
	}
}

func testAccCheckPluginManagerInviteAccepted(org string, m middleware.Middleware) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["cycloid_plugin_manager.test"]
		if !ok {
			return fmt.Errorf("cycloid_plugin_manager.test not in state")
		}

		id, err := strconv.ParseUint(rs.Primary.Attributes["id"], 10, 32)
		if err != nil {
			return fmt.Errorf("invalid plugin manager id %q: %w", rs.Primary.Attributes["id"], err)
		}

		pm, _, err := m.GetPluginManager(org, uint32(id))
		if err != nil {
			return fmt.Errorf("failed to read plugin manager %d from API: %w", id, err)
		}
		if pm.InviteStatus == nil {
			return fmt.Errorf("plugin manager %d has no invite_status", id)
		}

		switch *pm.InviteStatus {
		case "accepted", "invite_accepted":
			return nil
		default:
			return fmt.Errorf("plugin manager %d invite_status = %q, want accepted", id, *pm.InviteStatus)
		}
	}
}

func testAccPluginManagerConfig(org, name, url string) string {
	return testAccPluginManagerConfigWithAutoRegister(org, name, url, true)
}

func testAccPluginManagerConfigWithAutoRegister(org, name, url string, autoRegister bool) string {
	return fmt.Sprintf(`
resource "cycloid_plugin_manager" "test" {
  organization  = %q
  name          = %q
  url           = %q
  auto_register = %t
}
`, org, name, url, autoRegister)
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
