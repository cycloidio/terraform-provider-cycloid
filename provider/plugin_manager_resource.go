package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_plugin_manager"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &pluginManagerResource{}

type pluginManagerResourceModel resource_plugin_manager.PluginManagerModel

func NewPluginManagerResource() resource.Resource {
	return &pluginManagerResource{}
}

type pluginManagerResource struct {
	provider *CycloidProvider
}

func (r *pluginManagerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin_manager"
}

func (r *pluginManagerResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_plugin_manager.PluginManagerResourceSchema(ctx)
}

func (r *pluginManagerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *pluginManagerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data pluginManagerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Middleware

	pm, _, err := m.CreatePluginManager(org, data.Name.ValueString(), data.URL.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to create plugin manager in org %q", org), err.Error())
		return
	}

	// Accept the invite immediately so the resource is in a fully declared state.
	pm, _, err = m.UpdatePluginManager(org, uint32(ptr.Value(pm.ID)), "accepted")
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("created plugin manager %d but failed to accept the invite in org %q", ptr.Value(pm.ID), org),
			err.Error(),
		)
		return
	}

	pluginManagerToModel(org, pm, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pluginManagerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data pluginManagerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Middleware

	id := uint32(data.ID.ValueInt64())
	pm, _, err := m.GetPluginManager(org, id)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("failed to read plugin manager %d in org %q", id, org), err.Error())
		return
	}

	pluginManagerToModel(org, pm, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pluginManagerResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All fields use RequiresReplace — Update is never called.
}

func (r *pluginManagerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data pluginManagerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Middleware

	id := uint32(data.ID.ValueInt64())
	_, err := m.DeletePluginManager(org, id)
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to delete plugin manager %d in org %q", id, org), err.Error())
	}
}

func pluginManagerToModel(org string, pm *models.PluginManager, data *pluginManagerResourceModel) {
	data.Organization = types.StringValue(org)
	data.ID = types.Int64Value(int64(ptr.Value(pm.ID)))
	data.Name = types.StringPointerValue(pm.Name)
	data.URL = types.StringValue(pm.URL.String())
	data.Status = types.StringPointerValue(pm.Status)
	data.InviteStatus = types.StringPointerValue(pm.InviteStatus)
	data.CreatedAt = types.Int64Value(int64(ptr.Value(pm.CreatedAt)))
	data.UpdatedAt = types.Int64Value(int64(ptr.Value(pm.UpdatedAt)))
}
