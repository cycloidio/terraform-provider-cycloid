package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cycloidio/cycloid-cli/cmd/apiclient"
	"github.com/cycloidio/terraform-provider-cycloid/resource_plugin_widget_view"
	"github.com/cycloidio/cycloid-cli/utils/ptr"
)

var (
	_ resource.Resource                = (*pluginWidgetViewResource)(nil)
	_ resource.ResourceWithImportState = (*pluginWidgetViewResource)(nil)
)

func NewPluginWidgetViewResource() resource.Resource {
	return &pluginWidgetViewResource{}
}

type pluginWidgetViewResource struct {
	provider *CycloidProvider
}

type pluginWidgetViewResourceModel resource_plugin_widget_view.PluginWidgetViewModel

func (r *pluginWidgetViewResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin_widget_view"
}

func (r *pluginWidgetViewResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_plugin_widget_view.PluginWidgetViewResourceSchema(ctx)
}

func (r *pluginWidgetViewResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *pluginWidgetViewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data pluginWidgetViewResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Client

	widgetViewID := uint32(data.WidgetViewID.ValueInt64())
	enabled := data.Enabled.ValueBool()
	urlSlug := data.URLSlug.ValueString()

	_, err := m.UpdatePluginWidgetView(org, widgetViewID, enabled, urlSlug)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to create widget view override %d in org %q", widgetViewID, org),
			err.Error(),
		)
		return
	}

	pluginInstallID := uint32(data.PluginInstallID.ValueInt64())

	diags := pluginWidgetViewRead(ctx, m, org, pluginInstallID, widgetViewID, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pluginWidgetViewResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data pluginWidgetViewResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Client

	pluginInstallID := uint32(data.PluginInstallID.ValueInt64())
	widgetViewID := uint32(data.WidgetViewID.ValueInt64())

	diags := pluginWidgetViewRead(ctx, m, org, pluginInstallID, widgetViewID, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pluginWidgetViewResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data pluginWidgetViewResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Client

	widgetViewID := uint32(data.WidgetViewID.ValueInt64())
	enabled := data.Enabled.ValueBool()
	urlSlug := data.URLSlug.ValueString()

	_, err := m.UpdatePluginWidgetView(org, widgetViewID, enabled, urlSlug)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to update widget view override %d in org %q", widgetViewID, org),
			err.Error(),
		)
		return
	}

	pluginInstallID := uint32(data.PluginInstallID.ValueInt64())

	diags := pluginWidgetViewRead(ctx, m, org, pluginInstallID, widgetViewID, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pluginWidgetViewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data pluginWidgetViewResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Client

	widgetViewID := uint32(data.WidgetViewID.ValueInt64())

	// No DELETE endpoint — disabling the widget view is the equivalent of deletion.
	_, err := m.UpdatePluginWidgetView(org, widgetViewID, false, data.URLSlug.ValueString())
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to disable widget view %d in org %q", widgetViewID, org),
			err.Error(),
		)
	}
}

func (r *pluginWidgetViewResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("expected <plugin_install_id>:<widget_view_id>, got %q", req.ID),
		)
		return
	}

	pluginInstallID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid plugin_install_id in import ID", err.Error())
		return
	}
	widgetViewID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid widget_view_id in import ID", err.Error())
		return
	}

	org := r.provider.DefaultOrganization
	m := r.provider.Client

	var data pluginWidgetViewResourceModel
	data.PluginInstallID = types.Int64Value(pluginInstallID)
	data.WidgetViewID = types.Int64Value(widgetViewID)

	diags := pluginWidgetViewRead(ctx, m, org, uint32(pluginInstallID), uint32(widgetViewID), &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// pluginWidgetViewRead fetches widget views for the given plugin install and finds the one
// matching widgetViewID, populating the model.
func pluginWidgetViewRead(_ context.Context, m apiclient.APIClient, org string, pluginInstallID, widgetViewID uint32, data *pluginWidgetViewResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	views, _, err := m.ListPluginWidgetViews(org, pluginInstallID)
	if err != nil {
		diags.AddError(
			fmt.Sprintf("failed to list widget views for install %d in org %q", pluginInstallID, org),
			err.Error(),
		)
		return diags
	}

	for _, v := range views {
		if ptr.Value(v.ID) == widgetViewID {
			data.Organization = types.StringValue(org)
			data.PluginInstallID = types.Int64Value(int64(pluginInstallID))
			data.WidgetViewID = types.Int64Value(int64(widgetViewID))
			data.Enabled = types.BoolPointerValue(v.Enabled)
			data.URLSlug = types.StringPointerValue(v.URLSlug)
			data.EffectiveEnabled = types.BoolPointerValue(v.EffectiveEnabled)
			data.EffectiveSlug = types.StringPointerValue(v.EffectiveSlug)
			data.IsInherited = types.BoolValue(v.IsInherited)
			data.HasOverride = types.BoolValue(v.HasOverride)
			return diags
		}
	}

	diags.AddError(
		fmt.Sprintf("widget view %d not found for install %d in org %q", widgetViewID, pluginInstallID, org),
		"The widget view may have been removed.",
	)
	return diags
}
