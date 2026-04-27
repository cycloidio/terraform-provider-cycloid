package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_plugin_registry"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &pluginRegistryResource{}

type pluginRegistryResourceModel resource_plugin_registry.PluginRegistryModel

func NewPluginRegistryResource() resource.Resource {
	return &pluginRegistryResource{}
}

type pluginRegistryResource struct {
	provider *CycloidProvider
}

func (r *pluginRegistryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin_registry"
}

func (r *pluginRegistryResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_plugin_registry.PluginRegistryResourceSchema(ctx)
}

func (r *pluginRegistryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *pluginRegistryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data pluginRegistryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Middleware

	registry, _, err := m.CreatePluginRegistry(org, data.Name.ValueString(), data.URL.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to create plugin registry in org %q", org), err.Error())
		return
	}

	pluginRegistryToModel(org, registry, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pluginRegistryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data pluginRegistryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Middleware

	id := uint32(data.ID.ValueInt64())
	registry, _, err := m.GetPluginRegistry(org, id)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("failed to read plugin registry %d in org %q", id, org), err.Error())
		return
	}

	pluginRegistryToModel(org, registry, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pluginRegistryResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All fields use RequiresReplace — Update is never called.
}

func (r *pluginRegistryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data pluginRegistryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Middleware

	id := uint32(data.ID.ValueInt64())
	_, err := m.DeletePluginRegistry(org, id)
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to delete plugin registry %d in org %q", id, org), err.Error())
	}
}

func pluginRegistryToModel(org string, r *models.PluginRegistry, data *pluginRegistryResourceModel) {
	data.Organization = types.StringValue(org)
	data.ID = types.Int64Value(int64(ptr.Value(r.ID)))
	data.Name = types.StringPointerValue(r.Name)
	data.URL = types.StringValue(r.URL.String())
	data.Status = types.StringPointerValue(r.Status)
	data.Access = types.BoolPointerValue(r.Access)
	data.CreatedAt = types.Int64Value(int64(ptr.Value(r.CreatedAt)))
	data.UpdatedAt = types.Int64Value(int64(ptr.Value(r.UpdatedAt)))
}
