package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_plugin_registry_plugin"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &pluginRegistryPluginResource{}

type pluginRegistryPluginResourceModel resource_plugin_registry_plugin.PluginRegistryPluginModel

func NewPluginRegistryPluginResource() resource.Resource {
	return &pluginRegistryPluginResource{}
}

type pluginRegistryPluginResource struct {
	provider *CycloidProvider
}

func (r *pluginRegistryPluginResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin_registry_plugin"
}

func (r *pluginRegistryPluginResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_plugin_registry_plugin.PluginRegistryPluginResourceSchema(ctx)
}

func (r *pluginRegistryPluginResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *pluginRegistryPluginResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data pluginRegistryPluginResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Middleware

	registryID := uint32(data.RegistryID.ValueInt64())
	plugin, _, err := m.CreateRegistryPlugin(org, registryID, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to create plugin in registry %d in org %q", registryID, org),
			err.Error(),
		)
		return
	}

	pluginRegistryPluginToModel(org, plugin, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pluginRegistryPluginResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data pluginRegistryPluginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Middleware

	registryID := uint32(data.RegistryID.ValueInt64())
	pluginID := uint32(data.ID.ValueInt64())
	plugin, _, err := m.GetRegistryPlugin(org, registryID, pluginID)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to read plugin %d in registry %d in org %q", pluginID, registryID, org),
			err.Error(),
		)
		return
	}

	pluginRegistryPluginToModel(org, plugin, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pluginRegistryPluginResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data pluginRegistryPluginResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read state to get the ID.
	var state pluginRegistryPluginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Middleware

	registryID := uint32(data.RegistryID.ValueInt64())
	pluginID := uint32(state.ID.ValueInt64())
	plugin, _, err := m.UpdateRegistryPlugin(org, registryID, pluginID, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to update plugin %d in registry %d in org %q", pluginID, registryID, org),
			err.Error(),
		)
		return
	}

	pluginRegistryPluginToModel(org, plugin, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pluginRegistryPluginResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data pluginRegistryPluginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Middleware

	registryID := uint32(data.RegistryID.ValueInt64())
	pluginID := uint32(data.ID.ValueInt64())
	_, err := m.DeleteRegistryPlugin(org, registryID, pluginID)
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to delete plugin %d in registry %d in org %q", pluginID, registryID, org),
			err.Error(),
		)
	}
}

func pluginRegistryPluginToModel(org string, p *models.Plugin, data *pluginRegistryPluginResourceModel) {
	data.Organization = types.StringValue(org)
	data.ID = types.Int64Value(int64(ptr.Value(p.ID)))
	data.Name = types.StringPointerValue(p.Name)
	data.Owned = types.BoolPointerValue(p.Owned)
}
