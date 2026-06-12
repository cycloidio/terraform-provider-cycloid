package provider

import (
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestPluginInstallToModel_NilFields is the regression test for ENG-183:
// the provider crashed with a SIGSEGV in pluginInstallToModel because the
// GET/refresh response returns a PluginInstall whose UUID (*strfmt.UUID) is
// nil, and install.UUID.String() dereferenced the nil pointer.
//
// pluginInstallToModel must not panic on any nil field, and must preserve the
// UUID already present in state when the API omits it.
func TestPluginInstallToModel_NilFields(t *testing.T) {
	// data simulates prior state from a successful create — UUID is populated.
	data := pluginResourceModel{
		UUID: types.StringValue("11111111-1111-1111-1111-111111111111"),
	}

	// install simulates the refresh GET response: required-in-swagger fields
	// come back nil/zero, mirroring the production crash on DIGIT.
	install := &models.PluginInstall{
		ID:        ptr.Ptr(uint32(42)),
		UUID:      nil, // <- the field that caused the panic
		Status:    nil,
		CreatedAt: nil,
		UpdatedAt: nil,
		Version:   nil,
	}

	// Must not panic.
	pluginInstallToModel("my-org", install, &data)

	if got := data.Organization.ValueString(); got != "my-org" {
		t.Errorf("Organization: got %q, want %q", got, "my-org")
	}
	if got := data.ID.ValueInt64(); got != 42 {
		t.Errorf("ID: got %d, want 42", got)
	}
	// UUID omitted by the API → preserved from prior state, not blanked.
	if got := data.UUID.ValueString(); got != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("UUID: got %q, want preserved-from-state value", got)
	}
	// Nil pointer fields degrade to their zero values without panicking.
	if got := data.CreatedAt.ValueInt64(); got != 0 {
		t.Errorf("CreatedAt: got %d, want 0", got)
	}
	if got := data.UpdatedAt.ValueInt64(); got != 0 {
		t.Errorf("UpdatedAt: got %d, want 0", got)
	}
	if !data.Status.IsNull() {
		t.Errorf("Status: got %v, want null", data.Status)
	}
}

// TestPluginInstallToModel_FullyPopulated verifies the create/import path where
// the API returns a fully populated install: every field is mapped through.
func TestPluginInstallToModel_FullyPopulated(t *testing.T) {
	uuid := strfmt.UUID("22222222-2222-2222-2222-222222222222")
	data := pluginResourceModel{}

	install := &models.PluginInstall{
		ID:        ptr.Ptr(uint32(7)),
		UUID:      &uuid,
		Status:    ptr.Ptr("running"),
		CreatedAt: ptr.Ptr(uint64(1000)),
		UpdatedAt: ptr.Ptr(uint64(2000)),
		Version:   &models.PluginVersion{ID: ptr.Ptr(uint32(99))},
	}

	pluginInstallToModel("org", install, &data)

	if got := data.UUID.ValueString(); got != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("UUID: got %q, want populated value", got)
	}
	if got := data.Status.ValueString(); got != "running" {
		t.Errorf("Status: got %q, want running", got)
	}
	if got := data.CreatedAt.ValueInt64(); got != 1000 {
		t.Errorf("CreatedAt: got %d, want 1000", got)
	}
	if got := data.UpdatedAt.ValueInt64(); got != 2000 {
		t.Errorf("UpdatedAt: got %d, want 2000", got)
	}
	if got := data.PluginVersionID.ValueInt64(); got != 99 {
		t.Errorf("PluginVersionID: got %d, want 99", got)
	}
}
