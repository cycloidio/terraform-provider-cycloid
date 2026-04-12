package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/cycloid-cli/client/models"
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

	var versionID *uint32
	if !data.PluginVersionID.IsNull() && !data.PluginVersionID.IsUnknown() {
		v := uint32(data.PluginVersionID.ValueInt64())
		versionID = &v
	}

	config := map[string]string{}
	resp.Diagnostics.Append(data.Configuration.ElementsAs(ctx, &config, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	install, _, err := m.CreatePlugin(org, versionID, config)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to install plugin in org %q", org), err.Error())
		return
	}

	pluginInstallToModel(org, install, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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
