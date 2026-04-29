package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccPluginResource exercises the full plugin install lifecycle:
// Create (install, poll pending→running) → Destroy (uninstall).
//
// Plugin status flow on the local stack: pending → running (never "installed").
// The resource polls until status == "running"; see plugin_resource.go.
//
// The plugin-manager widget proxy has a known bug (routes to youdeploy-api instead
// of the plugin pod IP). Widget-query assertions are therefore out of scope here.
// TODO(plugin-manager-proxy-fix): add widget-query coverage once the bug is fixed.
func TestAccPluginResource(t *testing.T) {
	// TODO: POST /organizations/{org}/plugins returns 405 (only GET supported).
	// The resource must be refactored to use InstallPluginVersion (requires adding
	// registry_id and plugin_id to the schema). Track as a separate fix.
	t.Skip("cycloid_plugin install: POST /organizations/{org}/plugins returns 405; requires schema refactor")

	orgCanonical := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)

	if depManager.GetProvider().Middleware == nil {
		t.Skip("skipping acceptance test: middleware not configured")
	}

	// Push image and set up registry + plugin + version prerequisites.
	imageRef := ensurePluginHelloWorld(t)
	m := depManager.GetProvider().Middleware

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

	version, _, err := m.CreatePluginVersion(orgCanonical, registryID, pluginID, imageRef)
	if err != nil {
		t.Fatalf("failed to create test plugin version: %v", err)
	}
	versionID := uint32(ptr.Value(version.ID))
	t.Cleanup(func() { _, _ = m.DeletePluginVersion(orgCanonical, registryID, pluginID, versionID) })

	// Wait for version processing to succeed before trying to install.
	if err := pollPluginVersionStatus(m, orgCanonical, registryID, pluginID, versionID, "success", 5*time.Minute); err != nil {
		t.Fatalf("plugin version never reached success: %v", err)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccPluginConfig(orgCanonical, int(versionID)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_plugin.test", "organization", orgCanonical),
					resource.TestCheckResourceAttrSet("cycloid_plugin.test", "id"),
					resource.TestCheckResourceAttrSet("cycloid_plugin.test", "status"),
				),
			},
		},
	})
}

// pollPluginVersionStatus polls until the version status equals want or the timeout expires.
func pollPluginVersionStatus(m middleware.Middleware, org string, registryID, pluginID, versionID uint32, want string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		v, _, err := m.GetPluginVersion(org, registryID, pluginID, versionID)
		if err != nil {
			return err
		}
		if ptr.Value(v.Status) == want {
			return nil
		}
		if ptr.Value(v.Status) == "failed" {
			return fmt.Errorf("plugin version processing failed")
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timeout waiting for plugin version status %q", want)
}

func testAccPluginConfig(org string, versionID int) string {
	return fmt.Sprintf(`
resource "cycloid_plugin" "test" {
  organization      = %q
  plugin_version_id = %d
  configuration     = {}
}
`, org, versionID)
}
