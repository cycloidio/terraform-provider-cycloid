package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/cycloidio/cycloid-cli/client/models"
	middleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_plugin"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &pluginResource{}

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

	config := map[string]string{}
	resp.Diagnostics.Append(data.Configuration.ElementsAs(ctx, &config, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := m.InstallPluginVersion(org, registryID, pluginID, versionID, config)
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
	install, _, err := m.GetPlugin(org, id)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("failed to read plugin install %d in org %q", id, org), err.Error())
		return
	}

	pluginInstallToModel(org, install, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pluginResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All fields use RequiresReplace — Update is never called.
	// TODO: remove RequiresReplace on plugin_version_id and configuration once plugin upgrades work.
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

func pluginInstallToModel(org string, install *models.PluginInstall, data *pluginResourceModel) {
	data.Organization = types.StringValue(org)
	data.ID = types.Int64Value(int64(ptr.Value(install.ID)))
	data.UUID = types.StringValue(install.UUID.String())
	data.Status = types.StringPointerValue(install.Status)
	data.PmSecret = types.StringPointerValue(install.PmSecret)
	data.CreatedAt = types.Int64Value(int64(ptr.Value(install.CreatedAt)))
	data.UpdatedAt = types.Int64Value(int64(ptr.Value(install.UpdatedAt)))

	if install.Version != nil {
		data.PluginVersionID = types.Int64Value(int64(ptr.Value(install.Version.ID)))
	}
}
