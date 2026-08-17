package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cycloidio/cycloid-cli/cmd/apiclient"
	"github.com/cycloidio/cycloid-cli/gen/models"
	"github.com/cycloidio/terraform-provider-cycloid/resource_plugin"
	"github.com/cycloidio/cycloid-cli/utils/ptr"
)

const (
	defaultPluginInstallTimeout = 5 * time.Minute
	pluginInstallPollInterval   = 5 * time.Second
)

var (
	_ resource.Resource                = &pluginResource{}
	_ resource.ResourceWithImportState = &pluginResource{}
)

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
	m := r.provider.Client

	registryID := uint32(data.RegistryID.ValueInt64())
	pluginID := uint32(data.PluginID.ValueInt64())
	versionID := uint32(data.PluginVersionID.ValueInt64())

	config, err := mergePluginConfiguration(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("invalid plugin configuration", err.Error())
		return
	}

	var installID uint32

	install, _, err := m.InstallPluginVersion(org, registryID, pluginID, versionID, config)
	if err != nil {
		if !isConflictError(err) {
			resp.Diagnostics.AddError(
				fmt.Sprintf("failed to install plugin version %d in org %q", versionID, org),
				err.Error(),
			)
			return
		}
		// 409 Conflict: plugin already installed (e.g. after a transient 502/504
		// on a previous apply). Retry the install so the plugin manager
		// re-deploys it and we can adopt it into state.
		_, err = m.RetryPluginVersion(org, registryID, pluginID, versionID)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("failed to retry plugin version %d in org %q", versionID, org),
				err.Error(),
			)
			return
		}
		// Find the existing install ID via ListPlugins.
		plugins, _, listErr := m.ListPlugins(org)
		if listErr != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to list plugins after conflict in org %q", org), listErr.Error())
			return
		}
		installID = findInstallIDInPluginList(plugins, registryID, pluginID)
		if installID == 0 {
			resp.Diagnostics.AddError(fmt.Sprintf("plugin install not found after conflict retry in org %q", org), "")
			return
		}
	} else {
		if install == nil || install.ID == nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("install response for plugin version %d in org %q missing ID", versionID, org),
				"The API returned a successful install response but no install ID. This is unexpected; please report this issue.",
			)
			return
		}
		installID = ptr.Value(install.ID)
	}

	createTimeout, diags := data.Timeouts.Create(ctx, defaultPluginInstallTimeout)
	resp.Diagnostics.Append(diags...)

	// Save the install ID to state before polling so the resource survives a
	// timeout or context cancellation. Without this, a failed poll would leave
	// the backend install orphaned with no Terraform state entry, causing a
	// 409 conflict on the next apply with no recovery path via import.
	// This mirrors the pattern used in pluginVersionResource.Create.
	// The state write happens before the diagnostics HasError check so that
	// an invalid timeout value does not orphan an already-created install.
	data.Organization = types.StringValue(org)
	data.RegistryID = types.Int64Value(int64(registryID))
	data.PluginID = types.Int64Value(int64(pluginID))
	data.ID = types.Int64Value(int64(installID))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// InstallPluginVersion is async (pending → running). Poll using RefreshPluginInstallStatus
	// until the install reaches a terminal status.
	polledInstall, err := pollPluginInstall(ctx, m, org, installID, createTimeout)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("plugin install did not reach running status in org %q", org), err.Error())
		return
	}

	data.RegistryID = types.Int64Value(int64(registryID))
	data.PluginID = types.Int64Value(int64(pluginID))
	pluginInstallToModel(org, polledInstall, &data)
	if data.EnableAllWidgets.ValueBool() {
		enableAllPluginWidgetViews(ctx, m, org, installID, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// pollPluginInstall polls RefreshPluginInstallStatus until the install reaches
// status "running", then returns the PluginInstall. Returns an error on timeout,
// context cancellation, or when the install status is "failed".
//
// The first poll happens immediately on entry (before waiting for a tick) because
// installs occasionally complete quickly and skipping the initial delay gives a
// faster happy-path. Subsequent polls are spaced by pluginInstallPollInterval.
//
// Transient API errors (network glitches, 5xx) are logged and retried; only
// terminal status values ("failed") or expiry cause a permanent error.
//
// Note: ctx cancellation is checked between polls (in the select), but an
// in-flight RefreshPluginInstallStatus HTTP call cannot be interrupted because
// the apiclient interface does not accept a context. Cancellation takes effect
// at the next inter-poll sleep, not mid-request.
func pollPluginInstall(ctx context.Context, m apiclient.APIClient, org string, installID uint32, timeout time.Duration) (*models.PluginInstall, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(pluginInstallPollInterval)
	defer ticker.Stop()

	for {
		pi, _, err := m.RefreshPluginInstallStatus(org, installID)
		if err != nil {
			tflog.Warn(ctx, "transient error refreshing plugin install status; will retry", map[string]any{
				"install_id": installID,
				"org":        org,
				"error":      err.Error(),
			})
		} else if pi != nil {
			switch ptr.Value(pi.Status) {
			case models.PluginInstallStatusRunning:
				return pi, nil
			case models.PluginInstallStatusFailed:
				return nil, fmt.Errorf("plugin install %d in org %q failed", installID, org)
			case models.PluginInstallStatusPending:
				// Expected intermediate state — keep polling.
			default:
				// Unknown intermediate status — continue polling rather than failing.
				// The overall timeout still applies.
				tflog.Warn(ctx, "unexpected plugin install status — continuing to poll", map[string]any{
					"install_id": installID,
					"org":        org,
					"status":     ptr.Value(pi.Status),
				})
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for plugin install %d in org %q to reach running status", installID, org)
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled while waiting for plugin install %d in org %q", installID, org)
		case <-ticker.C:
		}
	}
}

func (r *pluginResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data pluginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Client

	notFound, diags := pluginRead(ctx, m, org, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if notFound {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// pluginRead fetches a plugin install by ID and populates data.
// Returns (notFound bool, diags). notFound=true means the install is gone.
func pluginRead(ctx context.Context, m apiclient.APIClient, org string, data *pluginResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	id := uint32(data.ID.ValueInt64())

	p, _, err := m.GetPlugin(org, id)
	if err != nil {
		if isNotFoundError(err) {
			return true, nil
		}
		diags.AddError(fmt.Sprintf("failed to get plugin install %d in org %q", id, org), err.Error())
		return false, diags
	}
	if p == nil || p.Install == nil {
		return true, nil
	}

	pluginInstallToModel(org, p.Install, data)

	// Recover visible configuration from the API's merged map.
	// The API returns all config keys in one map — we subtract the sensitive
	// key set (preserved from state) to reconstruct the visible portion.
	if p.Install.Configuration != nil {
		sensitiveKeys := map[string]struct{}{}
		if !data.ConfigurationSensitive.IsNull() && !data.ConfigurationSensitive.IsUnknown() {
			var sensitive map[string]string
			if sensitiveDialgs := data.ConfigurationSensitive.ElementsAs(ctx, &sensitive, false); !sensitiveDialgs.HasError() {
				for k := range sensitive {
					sensitiveKeys[k] = struct{}{}
				}
			}
		}
		visible := map[string]string{}
		for k, v := range p.Install.Configuration {
			if _, isSensitive := sensitiveKeys[k]; !isSensitive {
				visible[k] = v
			}
		}
		if len(visible) > 0 {
			visibleMap, mapDiags := types.MapValueFrom(ctx, types.StringType, visible)
			if !mapDiags.HasError() {
				data.Configuration = visibleMap
			}
		} else if !data.Configuration.IsNull() {
			data.Configuration = types.MapNull(types.StringType)
		}
	} else if !data.Configuration.IsNull() {
		data.Configuration = types.MapNull(types.StringType)
	}

	return false, diags
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
	m := r.provider.Client

	id := uint32(state.ID.ValueInt64())
	versionID := uint32(plan.PluginVersionID.ValueInt64())

	config, err := mergePluginConfiguration(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("invalid plugin configuration", err.Error())
		return
	}

	// UpdatePlugin is async — the returned install reflects the pre-update state.
	// We discard it and poll RefreshPluginInstallStatus below to get the
	// post-update running state.
	_, _, err = m.UpdatePlugin(org, id, versionID, config)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to update plugin install %d in org %q", id, org), err.Error())
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, defaultPluginInstallTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	install, err := pollPluginInstall(ctx, m, org, id, updateTimeout)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("plugin update did not reach running status in org %q", org), err.Error())
		return
	}

	plan.RegistryID = types.Int64Value(plan.RegistryID.ValueInt64())
	plan.PluginID = types.Int64Value(plan.PluginID.ValueInt64())
	pluginInstallToModel(org, install, &plan)
	if plan.EnableAllWidgets.ValueBool() {
		enableAllPluginWidgetViews(ctx, m, org, id, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pluginResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data pluginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Client

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
	m := r.provider.Client

	p, _, err := m.GetPlugin(org, uint32(installID))
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to get plugin install %d for import in org %q", installID, org), err.Error())
		return
	}
	if p == nil || p.Install == nil {
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
	// Seed Timeouts with a correctly-typed null so State.Set does not fail with a
	// Value Conversion Error. The zero timeouts.Value has no attribute types and
	// does not conform to the schema block.
	data.Timeouts = timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"update": types.StringType,
		}),
	}
	data.EnableAllWidgets = types.BoolNull()
	pluginInstallToModel(org, p.Install, &data)
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

// findInstallIDInPluginList searches for the install whose registry matches registryID
// and whose plugin (catalog entry) matches pluginID, returning the install ID, or 0 if not found.
func findInstallIDInPluginList(plugins []*models.Plugin, registryID, pluginID uint32) uint32 {
	for _, p := range plugins {
		if p.Install != nil && p.Registry != nil && ptr.Value(p.Registry.ID) == registryID && ptr.Value(p.ID) == pluginID {
			return ptr.Value(p.Install.ID)
		}
	}
	return 0
}

func enableAllPluginWidgetViews(ctx context.Context, m apiclient.APIClient, org string, installID uint32, diags *diag.Diagnostics) {
	views, _, err := m.ListPluginWidgetViews(org, installID)
	if err != nil {
		diags.AddError(fmt.Sprintf("failed to list widget views for install %d in org %q", installID, org), err.Error())
		return
	}
	if len(views) == 0 {
		tflog.Warn(ctx, "enable_all_widgets is true but plugin install has no widget views", map[string]any{
			"install_id": installID,
			"org":        org,
		})
	}
	for _, v := range views {
		_, err := m.UpdatePluginWidgetView(org, ptr.Value(v.ID), true, ptr.Value(v.URLSlug))
		if err != nil {
			diags.AddError(fmt.Sprintf("failed to enable widget view %d in org %q", ptr.Value(v.ID), org), err.Error())
			return
		}
	}
}

func pluginInstallToModel(org string, install *models.PluginInstall, data *pluginResourceModel) {
	data.Organization = types.StringValue(org)
	data.ID = types.Int64Value(int64(ptr.Value(install.ID)))
	if install.UUID != nil {
		data.UUID = types.StringValue(install.UUID.String())
	}
	// nil UUID: preserve existing state value (API omits uuid in List response;
	// overwriting with "" would cause persistent drift — mirrors ENG-183 / PR #109).
	data.Status = types.StringPointerValue(install.Status)
	data.CreatedAt = types.Int64Value(int64(ptr.Value(install.CreatedAt)))
	data.UpdatedAt = types.Int64Value(int64(ptr.Value(install.UpdatedAt)))

	data.PmSecret = types.StringPointerValue(install.PmSecret)

	if install.Version != nil {
		data.PluginVersionID = types.Int64Value(int64(ptr.Value(install.Version.ID)))
		data.VersionName = types.StringPointerValue(install.Version.Name)
		data.VersionStatus = types.StringPointerValue(install.Version.Status)
	}
}
