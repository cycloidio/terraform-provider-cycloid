package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_plugin_version"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	pluginVersionPollInterval = 5 * time.Second
	pluginVersionPollTimeout  = 10 * time.Minute
)

var _ resource.Resource = &pluginVersionResource{}

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
	m := r.provider.Middleware

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

	versionID := uint32(ptr.Value(version.ID))

	// Save state early so the resource exists even if polling fails.
	pluginVersionToModel(org, version, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Poll until success or failure.
	deadline := time.Now().Add(pluginVersionPollTimeout)
	for time.Now().Before(deadline) {
		version, _, err = m.GetPluginVersion(org, registryID, pluginID, versionID)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("failed to poll plugin version %d status", versionID),
				err.Error(),
			)
			return
		}

		status := ptr.Value(version.Status)
		switch status {
		case "success":
			pluginVersionToModel(org, version, &data)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		case "failed":
			// Taint: mark the resource as needing recreation by writing the
			// failed state and then adding an error so Terraform marks it tainted.
			pluginVersionToModel(org, version, &data)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			resp.Diagnostics.AddError(
				fmt.Sprintf("plugin version %d processing failed", versionID),
				fmt.Sprintf("The plugin version entered status %q. Run `terraform apply` again to retry.", status),
			)
			return
		}

		time.Sleep(pluginVersionPollInterval)
	}

	resp.Diagnostics.AddError(
		fmt.Sprintf("timed out waiting for plugin version %d to finish processing", versionID),
		fmt.Sprintf("Last status: %q. The resource has been saved; run `terraform apply` again to continue.", ptr.Value(version.Status)),
	)
}

func (r *pluginVersionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data pluginVersionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	m := r.provider.Middleware

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
	m := r.provider.Middleware

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

func pluginVersionToModel(org string, v *models.PluginVersion, data *pluginVersionResourceModel) {
	data.Organization = types.StringValue(org)
	data.ID = types.Int64Value(int64(ptr.Value(v.ID)))
	data.Name = types.StringPointerValue(v.Name)
	data.URL = types.StringValue(v.URL.String())
	data.Status = types.StringPointerValue(v.Status)
	data.Description = types.StringValue(v.Description)
}
