package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cycloidio/cycloid-cli/cmd/apiclient"
	"github.com/cycloidio/terraform-provider-cycloid/resource_plugin_sharing"
	"github.com/cycloidio/cycloid-cli/utils/ptr"
)

var (
	_ resource.Resource                = (*pluginSharingResource)(nil)
	_ resource.ResourceWithImportState = (*pluginSharingResource)(nil)
)

func NewPluginSharingResource() resource.Resource {
	return &pluginSharingResource{}
}

type pluginSharingResource struct {
	provider *CycloidProvider
}

type pluginSharingResourceModel resource_plugin_sharing.PluginSharingModel

func (r *pluginSharingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin_sharing"
}

func (r *pluginSharingResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_plugin_sharing.PluginSharingResourceSchema(ctx)
}

func (r *pluginSharingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *pluginSharingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data pluginSharingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Client

	pluginInstallID := uint32(data.PluginInstallID.ValueInt64())
	visibility := data.Visibility.ValueString()
	mode := data.Mode.ValueString()
	orgs := pluginSharingOrgsFromData(ctx, data)

	_, err := m.SetPluginInstallSharing(org, pluginInstallID, visibility, mode, orgs)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to set plugin sharing for install %d in org %q", pluginInstallID, org),
			err.Error(),
		)
		return
	}

	// Read back sharing config.
	_, diags := pluginSharingRead(ctx, m, org, pluginInstallID, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pluginSharingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data pluginSharingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Client

	pluginInstallID := uint32(data.PluginInstallID.ValueInt64())

	notFound, diags := pluginSharingRead(ctx, m, org, pluginInstallID, &data)
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

func (r *pluginSharingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data pluginSharingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Client

	pluginInstallID := uint32(data.PluginInstallID.ValueInt64())
	visibility := data.Visibility.ValueString()
	mode := data.Mode.ValueString()
	orgs := pluginSharingOrgsFromData(ctx, data)

	_, err := m.SetPluginInstallSharing(org, pluginInstallID, visibility, mode, orgs)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to update plugin sharing for install %d in org %q", pluginInstallID, org),
			err.Error(),
		)
		return
	}

	// Read back sharing config.
	_, diags := pluginSharingRead(ctx, m, org, pluginInstallID, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pluginSharingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data pluginSharingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Client

	pluginInstallID := uint32(data.PluginInstallID.ValueInt64())

	// Reset to local/include with no organizations.
	_, err := m.SetPluginInstallSharing(org, pluginInstallID, "local", "include", nil)
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to reset plugin sharing for install %d in org %q", pluginInstallID, org),
			err.Error(),
		)
	}
}

func (r *pluginSharingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	pluginInstallID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("expected <plugin_install_id>, got %q: %s", req.ID, err.Error()),
		)
		return
	}

	org := r.provider.DefaultOrganization
	m := r.provider.Client

	var data pluginSharingResourceModel
	data.PluginInstallID = types.Int64Value(pluginInstallID)

	_, diags := pluginSharingRead(ctx, m, org, uint32(pluginInstallID), &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// pluginSharingRead fetches the sharing config from the API and populates the model.
// Returns (notFound bool, diags Diagnostics). notFound is true when the org or install is gone.
func pluginSharingRead(ctx context.Context, m apiclient.APIClient, org string, pluginInstallID uint32, data *pluginSharingResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	sharing, _, err := m.GetPluginInstallSharing(org, pluginInstallID)
	if err != nil {
		if isNotFoundError(err) {
			return true, nil
		}
		diags.AddError(
			fmt.Sprintf("failed to read plugin sharing for install %d in org %q", pluginInstallID, org),
			err.Error(),
		)
		return false, diags
	}

	data.Organization = types.StringValue(org)
	data.PluginInstallID = types.Int64Value(int64(pluginInstallID))
	data.Visibility = types.StringValue(ptr.Value(sharing.Visibility))
	data.Mode = types.StringValue(ptr.Value(sharing.Mode))

	if len(sharing.Organizations) > 0 {
		listVal, listDiags := types.ListValueFrom(ctx, types.StringType, sharing.Organizations)
		diags.Append(listDiags...)
		if diags.HasError() {
			return false, diags
		}
		data.Organizations = listVal
	} else {
		data.Organizations = types.ListNull(types.StringType)
	}

	return false, diags
}

// pluginSharingOrgsFromData extracts the organizations list from the model.
func pluginSharingOrgsFromData(ctx context.Context, data pluginSharingResourceModel) []string {
	if data.Organizations.IsNull() || data.Organizations.IsUnknown() {
		return nil
	}
	var orgs []string
	data.Organizations.ElementsAs(ctx, &orgs, false)
	return orgs
}
