package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/datasource_organization_plugin_widgets"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestPluginWidgetsToData(t *testing.T) {
	ctx := context.Background()
	widgetType := "iframe"
	isDefault := true
	relationEnabled := true

	var data organizationPluginWidgetsDatasourceModel
	diags := pluginWidgetsToData(ctx, "my-org", []*models.PluginWidget{
		{
			ID:        42,
			Type:      &widgetType,
			IsDefault: &isDefault,
			Placement: map[string]any{
				"type": "sideMenuPage",
			},
			Widget: map[string]any{
				"url": "/plugins/hello",
			},
			Relation: &models.PluginRelation{
				Enabled: &relationEnabled,
				Relations: []any{
					"component",
				},
			},
		},
		nil,
	}, &data)
	if diags.HasError() {
		t.Fatalf("pluginWidgetsToData returned diagnostics: %v", diags)
	}

	if got := data.Organization.ValueString(); got != "my-org" {
		t.Fatalf("organization = %q, want my-org", got)
	}
	if got := len(data.Widgets.Elements()); got != 1 {
		t.Fatalf("widgets length = %d, want 1", got)
	}

	var widgets []datasource_organization_plugin_widgets.PluginWidgetModel
	diags = data.Widgets.ElementsAs(ctx, &widgets, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs returned diagnostics: %v", diags)
	}

	if got := widgets[0].ID.ValueInt64(); got != 42 {
		t.Fatalf("widget id = %d, want 42", got)
	}
	if got := widgets[0].Type.ValueString(); got != "iframe" {
		t.Fatalf("widget type = %q, want iframe", got)
	}
	if got := widgets[0].IsDefault.ValueBool(); !got {
		t.Fatalf("is_default = %t, want true", got)
	}

	var relation map[string]any
	if err := json.Unmarshal([]byte(widgets[0].Relation.ValueString()), &relation); err != nil {
		t.Fatalf("relation is not valid JSON: %v", err)
	}
	if got := relation["enabled"]; got != true {
		t.Fatalf("relation.enabled = %v, want true", got)
	}
}

func TestAccOrganizationPluginWidgetsDataSource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping acceptance test in short mode")
	}

	orgCanonical := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)

	if depManager.GetProvider().Middleware == nil {
		t.Skip("skipping acceptance test: middleware not configured")
	}

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

	if err := pollPluginVersionStatus(m, orgCanonical, registryID, pluginID, versionID, "success", 5*time.Minute); err != nil {
		t.Fatalf("plugin version never reached success: %v", err)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationPluginWidgetsDataSourceConfig(orgCanonical, int(registryID), int(pluginID), int(versionID)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cycloid_organization_plugin_widgets.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("data.cycloid_organization_plugin_widgets.test", "placement", "sideMenuPage"),
					resource.TestCheckResourceAttrWith("data.cycloid_organization_plugin_widgets.test", "widgets.#", func(value string) error {
						count, err := strconv.Atoi(value)
						if err != nil {
							return err
						}
						if count == 0 {
							return fmt.Errorf("expected at least one plugin widget")
						}
						return nil
					}),
					resource.TestCheckResourceAttrSet("data.cycloid_organization_plugin_widgets.test", "widgets.0.id"),
					resource.TestCheckResourceAttrSet("data.cycloid_organization_plugin_widgets.test", "widgets.0.type"),
				),
			},
		},
	})
}

func testAccOrganizationPluginWidgetsDataSourceConfig(org string, registryID, pluginID, versionID int) string {
	return fmt.Sprintf(`
resource "cycloid_plugin" "test" {
  organization      = %[1]q
  registry_id       = %[2]d
  plugin_id         = %[3]d
  plugin_version_id = %[4]d
  configuration = {
    greeting = "hello"
  }
  configuration_sensitive = {
    token = "test-token"
  }
}

data "cycloid_organization_plugin_widgets" "test" {
  organization = %[1]q
  placement    = "sideMenuPage"

  depends_on = [cycloid_plugin.test]
}
`, org, registryID, pluginID, versionID)
}
