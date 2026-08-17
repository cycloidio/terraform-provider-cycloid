package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cycloidio/cycloid-cli/gen/models"
	"github.com/cycloidio/terraform-provider-cycloid/resource_plugin_version"
	"github.com/cycloidio/cycloid-cli/utils/ptr"
)

const (
	pluginVersionPollInterval         = 5 * time.Second
	defaultPluginVersionCreateTimeout = 10 * time.Minute
)

var (
	_ resource.Resource                = &pluginVersionResource{}
	_ resource.ResourceWithImportState = &pluginVersionResource{}
)

type pluginVersionResourceModel resource_plugin_version.PluginVersionModel

func NewPluginVersionResource() resource.Resource {
	return &pluginVersionResource{}
}

type pluginVersionResource struct {
	provider *CycloidProvider
}

func (r *pluginVersionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin_version"
}

func (r *pluginVersionResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_plugin_version.PluginVersionResourceSchema(ctx)
}

func (r *pluginVersionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *pluginVersionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data pluginVersionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Client

	registryID := uint32(data.RegistryID.ValueInt64())
	pluginID := uint32(data.PluginID.ValueInt64())

	version, _, err := m.CreatePluginVersion(org, registryID, pluginID, data.URL.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to create plugin version in plugin %d registry %d org %q", pluginID, registryID, org),
			err.Error(),
		)
		return
	}

	versionID := ptr.Value(version.ID)

	createTimeout, diags := data.Timeouts.Create(ctx, defaultPluginVersionCreateTimeout)
	resp.Diagnostics.Append(diags...)

	// Save state early so the resource exists even if polling fails.
	// The state write happens before the diagnostics HasError check so that
	// an invalid timeout value does not orphan an already-created version.
	pluginVersionToModel(org, version, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Poll until success or failure.
	deadline := time.Now().Add(createTimeout)
	ticker := time.NewTicker(pluginVersionPollInterval)
	defer ticker.Stop()

	for {
		v, _, pollErr := m.GetPluginVersion(org, registryID, pluginID, versionID)
		if pollErr != nil {
			// Treat network/5xx errors as transient; log and retry on the next tick.
			tflog.Warn(ctx, "transient error polling plugin version status; will retry", map[string]any{
				"version_id": versionID,
				"error":      pollErr.Error(),
			})
		} else {
			version = v
			status := ptr.Value(version.Status)
			switch status {
			case models.PluginVersionStatusSuccess:
				pluginVersionToModel(org, version, &data)
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			case models.PluginVersionStatusFailed:
				// Taint: mark the resource as needing recreation by writing the
				// failed state and then adding an error so Terraform marks it tainted.
				pluginVersionToModel(org, version, &data)
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				resp.Diagnostics.AddError(
					fmt.Sprintf("plugin version %d processing failed", versionID),
					fmt.Sprintf("The plugin version entered status %q. Run `terraform apply` again to retry.", status),
				)
				return
			case models.PluginVersionStatusPending, models.PluginVersionStatusProcessing:
				// Expected intermediate states — keep polling.
			default:
				// Unknown intermediate status — log once per occurrence and continue.
				// Using tflog.Warn rather than resp.Diagnostics.AddWarning avoids
				// accumulating duplicate diagnostics on every tick.
				tflog.Warn(ctx, "unexpected plugin version status — continuing to poll", map[string]any{
					"version_id": versionID,
					"status":     status,
				})
			}
		}

		if time.Now().After(deadline) {
			// Persist the last-known status so state reflects the most recent poll.
			pluginVersionToModel(org, version, &data)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			resp.Diagnostics.AddError(
				fmt.Sprintf("timed out waiting for plugin version %d to finish processing", versionID),
				fmt.Sprintf("Last status: %q. The resource has been saved; run `terraform apply` again to continue.", ptr.Value(version.Status)),
			)
			return
		}

		select {
		case <-ctx.Done():
			// Persist the last-known status so state reflects the most recent poll.
			pluginVersionToModel(org, version, &data)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			resp.Diagnostics.AddError(
				fmt.Sprintf("context cancelled while waiting for plugin version %d to finish processing", versionID),
				fmt.Sprintf("Last status: %q. The resource has been saved; run `terraform apply` again to continue.", ptr.Value(version.Status)),
			)
			return
		case <-ticker.C:
		}
	}
}

func (r *pluginVersionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data pluginVersionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Client

	registryID := uint32(data.RegistryID.ValueInt64())
	pluginID := uint32(data.PluginID.ValueInt64())
	versionID := uint32(data.ID.ValueInt64())

	version, _, err := m.GetPluginVersion(org, registryID, pluginID, versionID)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to read plugin version %d in org %q", versionID, org),
			err.Error(),
		)
		return
	}

	pluginVersionToModel(org, version, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pluginVersionResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All fields use RequiresReplace — Update is never called.
}

func (r *pluginVersionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data pluginVersionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Client

	registryID := uint32(data.RegistryID.ValueInt64())
	pluginID := uint32(data.PluginID.ValueInt64())
	versionID := uint32(data.ID.ValueInt64())

	_, err := m.DeletePluginVersion(org, registryID, pluginID, versionID)
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to delete plugin version %d in org %q", versionID, org),
			err.Error(),
		)
	}
}

// ImportState supports: terraform import cycloid_plugin_version.x <registry_id>:<plugin_id>:<version_id>
func (r *pluginVersionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("expected <registry_id>:<plugin_id>:<version_id>, got %q", req.ID),
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
	versionID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid version ID in import", err.Error())
		return
	}

	org := r.provider.DefaultOrganization
	m := r.provider.Client

	version, _, err := m.GetPluginVersion(org, uint32(registryID), uint32(pluginID), uint32(versionID))
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to read plugin version %d for import", versionID), err.Error())
		return
	}

	var data pluginVersionResourceModel
	pluginVersionToModel(org, version, &data)
	// registry_id / plugin_id are path params, not returned on the version object,
	// so set them from the import ID — otherwise they default to 0 and the
	// post-import refresh reads /plugin_registries/0/... (API 422).
	data.RegistryID = types.Int64Value(registryID)
	data.PluginID = types.Int64Value(pluginID)
	// Seed Timeouts with a correctly-typed null so State.Set does not fail with a
	// Value Conversion Error. The zero timeouts.Value has no attribute types and
	// does not conform to the schema block.
	data.Timeouts = timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
		}),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func pluginVersionToModel(org string, v *models.PluginVersion, data *pluginVersionResourceModel) {
	data.Organization = types.StringValue(org)
	data.ID = types.Int64Value(int64(ptr.Value(v.ID)))
	data.Name = types.StringPointerValue(v.Name)
	data.URL = types.StringValue(v.URL.String())
	data.Status = types.StringPointerValue(v.Status)
	data.Description = types.StringValue(v.Description)
	data.Icon = types.StringValue(string(v.Icon))
	data.Error = types.StringValue(v.Error)

	images := make([]attr.Value, 0, len(v.Images))
	for _, img := range v.Images {
		images = append(images, types.StringValue(string(img)))
	}
	data.Images, _ = types.ListValue(types.StringType, images)

	scope := make([]attr.Value, 0, len(v.Scope))
	for _, s := range v.Scope {
		scope = append(scope, types.StringValue(s))
	}
	data.Scope, _ = types.ListValue(types.StringType, scope)

	if v.Configuration != nil {
		b, _ := json.Marshal(v.Configuration)
		data.ConfigurationSchema = types.StringValue(string(b))
	} else {
		data.ConfigurationSchema = types.StringValue("[]")
	}

	if v.Schema != nil {
		b, _ := json.Marshal(v.Schema)
		data.Schema = types.StringValue(string(b))
	} else {
		data.Schema = types.StringValue("{}")
	}

	if v.Widgets != nil {
		b, _ := json.Marshal(v.Widgets)
		data.Widgets = types.StringValue(string(b))
	} else {
		data.Widgets = types.StringValue("[]")
	}
}
