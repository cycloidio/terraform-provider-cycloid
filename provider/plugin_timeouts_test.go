package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestPluginSchemaTimeoutsBlock verifies the cycloid_plugin resource exposes a
// standard `timeouts` block covering create and update (the two operations that
// poll for the async install to reach "running"), and not delete/read which do
// no polling.
func TestPluginSchemaTimeoutsBlock(t *testing.T) {
	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}
	NewPluginResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	block, ok := resp.Schema.Blocks["timeouts"]
	if !ok {
		t.Fatal(`expected a "timeouts" block on cycloid_plugin`)
	}
	nested, ok := block.(schema.SingleNestedBlock)
	if !ok {
		t.Fatalf("timeouts block is not a SingleNestedBlock: %T", block)
	}
	for _, want := range []string{"create", "update"} {
		if _, ok := nested.Attributes[want]; !ok {
			t.Errorf("timeouts block missing %q", want)
		}
	}
	for _, no := range []string{"delete", "read"} {
		if _, ok := nested.Attributes[no]; ok {
			t.Errorf("timeouts block should not expose %q (no polling on that op)", no)
		}
	}
}

// TestPluginTimeoutDefaultsWhenUnset verifies that when the config omits the
// timeouts block (zero-value Timeouts), create/update fall back to
// defaultPluginInstallTimeout — preserving the pre-feature 5m behavior.
func TestPluginTimeoutDefaultsWhenUnset(t *testing.T) {
	ctx := context.Background()
	var data pluginResourceModel // null Timeouts, as with no timeouts block

	ct, diags := data.Timeouts.Create(ctx, defaultPluginInstallTimeout)
	if diags.HasError() || ct != defaultPluginInstallTimeout {
		t.Errorf("create: got %s diags=%v, want %s", ct, diags, defaultPluginInstallTimeout)
	}
	ut, diags := data.Timeouts.Update(ctx, defaultPluginInstallTimeout)
	if diags.HasError() || ut != defaultPluginInstallTimeout {
		t.Errorf("update: got %s diags=%v, want %s", ut, diags, defaultPluginInstallTimeout)
	}
}
