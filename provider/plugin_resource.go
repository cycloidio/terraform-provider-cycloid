package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cycloidio/cycloid-cli/client/models"
	middleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_plugin"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &pluginResource{}
var _ resource.ResourceWithImportState = &pluginResource{}

type pluginResourceModel resource_plugin.PluginModel

func NewPluginResource() resource.Resource {
	return &pluginResource{}
}

type pluginResource struct {
	provider *CycloidProvider
}

func (r *pluginResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin"
}

func (r *pluginResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_plugin.PluginResourceSchema(ctx)
}

func (r *pluginResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pv, ok := req.ProviderData.(*CycloidProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider data at Configure()",
			fmt.Sprintf("Expected *CycloidProvider, got: %T. Please report this issue.", req.ProviderData),
		)
		return
	}
	r.provider = pv
}

func (r *pluginResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data pluginResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Middleware

	registryID := uint32(data.RegistryID.ValueInt64())
	pluginID := uint32(data.PluginID.ValueInt64())
	versionID := uint32(data.PluginVersionID.ValueInt64())

	config, err := mergePluginConfiguration(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("invalid plugin configuration", err.Error())
		return
	}

	_, err = m.InstallPluginVersion(org, registryID, pluginID, versionID, config)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to install plugin version %d in org %q", versionID, org), err.Error())
		return
	}

	// InstallPluginVersion is async (pending → running). Poll ListPlugins until
	// the install for this registry+plugin pair appears with a terminal status.
	install, err := pollPluginInstall(m, org, registryID, pluginID, versionID, 5*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("plugin install did not reach running status in org %q", org), err.Error())
		return
	}

	data.RegistryID = types.Int64Value(int64(registryID))
	data.PluginID = types.Int64Value(int64(pluginID))
	pluginInstallToModel(org, install, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// pollPluginInstall polls ListPlugins until the install for the given registry+plugin
// appears with status "running", then returns the PluginInstall. Returns an error on
// timeout or when the install status is "failed".
func pollPluginInstall(m middleware.Middleware, org string, registryID, pluginID, versionID uint32, timeout time.Duration) (*models.PluginInstall, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		plugins, _, err := m.ListPlugins(org)
		if err != nil {
			return nil, err
		}
		for _, p := range plugins {
			if p.Install == nil || p.Registry == nil {
				continue
			}
			if ptr.Value(p.Registry.ID) != registryID || ptr.Value(p.ID) != pluginID {
				continue
			}
			if p.Install.Version != nil && ptr.Value(p.Install.Version.ID) != versionID {
				continue
			}
			status := ptr.Value(p.Install.Status)
			if status == "running" {
				return p.Install, nil
			}
			if status == "failed" {
				return nil, fmt.Errorf("plugin install failed")
			}
		}
		time.Sleep(5 * time.Second)
	}
	return nil, fmt.Errorf("timeout waiting for plugin install to reach running status")
}

func (r *pluginResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data pluginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Middleware

	id := uint32(data.ID.ValueInt64())

	// m.GetPlugin deserializes models.Plugin JSON into *models.PluginInstall, mapping
	// Plugin.ID (registry plugin ID) → PluginInstall.ID. Use ListPlugins instead and
	// locate the install by its actual ID so we get the correctly-typed PluginInstall.
	plugins, _, err := m.ListPlugins(org)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list plugins in org %q", org), err.Error())
		return
	}
	var install *models.Plugin
	for _, p := range plugins {
		if p.Install != nil && ptr.Value(p.Install.ID) == id {
			install = p
			break
		}
	}
	if install == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// configuration and configuration_sensitive are RequiresReplace and never
	// readable back from the API as split maps — preserve their values from state.
	pluginInstallToModel(org, install.Install, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pluginResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan pluginResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// id is computed from state; read it separately so we don't lose it.
	var state pluginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, plan.Organization)
	m := r.provider.Middleware

	id := uint32(state.ID.ValueInt64())
	versionID := uint32(plan.PluginVersionID.ValueInt64())

	config, err := mergePluginConfiguration(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("invalid plugin configuration", err.Error())
		return
	}

	_, _, err = m.UpdatePlugin(org, id, versionID, config)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to update plugin install %d in org %q", id, org), err.Error())
		return
	}

	install, err := pollPluginInstall(m, org, uint32(plan.RegistryID.ValueInt64()), uint32(plan.PluginID.ValueInt64()), versionID, 5*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("plugin update did not reach running status in org %q", org), err.Error())
		return
	}

	plan.RegistryID = types.Int64Value(plan.RegistryID.ValueInt64())
	plan.PluginID = types.Int64Value(plan.PluginID.ValueInt64())
	pluginInstallToModel(org, install, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pluginResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data pluginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Middleware

	id := uint32(data.ID.ValueInt64())
	_, err := m.DeletePlugin(org, id)
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to delete plugin install %d in org %q", id, org), err.Error())
	}
}

// ImportState supports: terraform import cycloid_plugin.x <registry_id>:<plugin_id>:<install_id>
func (r *pluginResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("expected <registry_id>:<plugin_id>:<install_id>, got %q", req.ID),
		)
		return
	}
	registryID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid registry ID in import", err.Error())
		return
	}
	pluginID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid plugin ID in import", err.Error())
		return
	}
	installID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid install ID in import", err.Error())
		return
	}

	org := r.provider.DefaultOrganization
	m := r.provider.Middleware

	// m.GetPlugin deserializes models.Plugin into *models.PluginInstall (wrong type).
	// Use ListPlugins and locate by install ID for a correctly-typed result.
	plugins, _, err := m.ListPlugins(org)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list plugins in org %q for import", org), err.Error())
		return
	}
	var importPlugin *models.Plugin
	for _, p := range plugins {
		if p.Install != nil && ptr.Value(p.Install.ID) == uint32(installID) {
			importPlugin = p
			break
		}
	}
	if importPlugin == nil {
		resp.Diagnostics.AddError(fmt.Sprintf("plugin install %d not found in org %q", installID, org), "")
		return
	}

	var data pluginResourceModel
	data.RegistryID = types.Int64Value(registryID)
	data.PluginID = types.Int64Value(pluginID)
	// configuration and configuration_sensitive cannot be recovered from API;
	// they will be null in imported state — user must add them to config after import.
	data.Configuration = types.MapNull(types.StringType)
	data.ConfigurationSensitive = types.MapNull(types.StringType)
	pluginInstallToModel(org, importPlugin.Install, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// mergePluginConfiguration merges configuration and configuration_sensitive into a single map.
// Returns an error if the same key appears in both maps.
func mergePluginConfiguration(ctx context.Context, data pluginResourceModel) (map[string]string, error) {
	config := map[string]string{}
	if !data.Configuration.IsNull() && !data.Configuration.IsUnknown() {
		var visible map[string]string
		if diags := data.Configuration.ElementsAs(ctx, &visible, false); diags.HasError() {
			return nil, fmt.Errorf("reading configuration: %s", diags[0].Detail())
		}
		for k, v := range visible {
			config[k] = v
		}
	}
	if !data.ConfigurationSensitive.IsNull() && !data.ConfigurationSensitive.IsUnknown() {
		var sensitive map[string]string
		if diags := data.ConfigurationSensitive.ElementsAs(ctx, &sensitive, false); diags.HasError() {
			return nil, fmt.Errorf("reading configuration_sensitive: %s", diags[0].Detail())
		}
		for k, v := range sensitive {
			if _, exists := config[k]; exists {
				return nil, fmt.Errorf("key %q appears in both configuration and configuration_sensitive", k)
			}
			config[k] = v
		}
	}
	return config, nil
}

func pluginInstallToModel(org string, install *models.PluginInstall, data *pluginResourceModel) {
	data.Organization = types.StringValue(org)
	data.ID = types.Int64Value(int64(ptr.Value(install.ID)))
	if install.UUID != nil {
		data.UUID = types.StringValue(install.UUID.String())
	} else {
		data.UUID = types.StringValue("")
	}
	data.Status = types.StringPointerValue(install.Status)
	data.CreatedAt = types.Int64Value(int64(ptr.Value(install.CreatedAt)))
	data.UpdatedAt = types.Int64Value(int64(ptr.Value(install.UpdatedAt)))

	if install.Version != nil {
		data.PluginVersionID = types.Int64Value(int64(ptr.Value(install.Version.ID)))
	}
}
