package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apiclient "github.com/cycloidio/cycloid-cli/cmd/apiclient"
	"github.com/cycloidio/cycloid-cli/gen/models"
	"github.com/cycloidio/terraform-provider-cycloid/resource_plugin_registry"
	"github.com/cycloidio/cycloid-cli/utils/ptr"
)

var (
	_ resource.Resource                = &pluginRegistryResource{}
	_ resource.ResourceWithImportState = &pluginRegistryResource{}
)

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

	id := uint32(ptr.Value(registry.ID))

	if data.WaitUntilConnected.ValueBool() {
		if err := pollPluginRegistryConnected(m, org, id, 5*time.Minute); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("plugin registry %d did not reach connected status in org %q", id, org),
				err.Error(),
			)
			return
		}
		// Refresh after polling.
		registries, _, listErr := m.ListPluginRegistries(org)
		if listErr == nil {
			for _, reg := range registries {
				if reg.ID != nil && uint32(*reg.ID) == id {
					registry = reg
					break
				}
			}
		}
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

	// GET /plugin_registries/{id} is not supported (405); use list + filter.
	id := uint32(data.ID.ValueInt64())
	registries, _, err := m.ListPluginRegistries(org)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list plugin registries in org %q", org), err.Error())
		return
	}
	var registry *models.PluginRegistry
	for _, reg := range registries {
		if reg.ID != nil && uint32(*reg.ID) == id {
			registry = reg
			break
		}
	}
	if registry == nil {
		resp.State.RemoveResource(ctx)
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

// ImportState supports: terraform import cycloid_plugin_registry.x <id>
func (r *pluginRegistryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("expected a numeric plugin registry ID, got %q: %v", req.ID, err))
		return
	}
	org := r.provider.DefaultOrganization
	m := r.provider.Middleware

	registries, _, err := m.ListPluginRegistries(org)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list plugin registries for import in org %q", org), err.Error())
		return
	}
	var registry *models.PluginRegistry
	for _, reg := range registries {
		if reg.ID != nil && uint32(*reg.ID) == uint32(id) {
			registry = reg
			break
		}
	}
	if registry == nil {
		resp.Diagnostics.AddError("Plugin registry not found", fmt.Sprintf("no registry with ID %d in org %q", id, org))
		return
	}

	var data pluginRegistryResourceModel
	pluginRegistryToModel(org, registry, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// pollPluginRegistryConnected polls list+filter until the registry status == "connected".
func pollPluginRegistryConnected(m apiclient.Middleware, org string, id uint32, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		registries, _, err := m.ListPluginRegistries(org)
		if err != nil {
			return err
		}
		for _, reg := range registries {
			if reg.ID != nil && uint32(*reg.ID) == id && ptr.Value(reg.Status) == "connected" {
				return nil
			}
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timeout waiting for plugin registry %d to reach connected status", id)
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
