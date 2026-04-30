package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/cycloidio/cycloid-cli/client/models"
	middleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_plugin_manager"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &pluginManagerResource{}
var _ resource.ResourceWithImportState = &pluginManagerResource{}

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
	pmID := uint32(ptr.Value(pm.ID))
	pm, _, err = m.UpdatePluginManager(org, pmID, "accepted")
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("created plugin manager %d but failed to accept the invite in org %q", pmID, org),
			err.Error(),
		)
		return
	}

	if data.WaitUntilConnected.ValueBool() {
		if err := pollPluginManagerConnected(m, org, pmID, 5*time.Minute); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("plugin manager %d did not reach connected status in org %q", pmID, org),
				err.Error(),
			)
			return
		}
		// Refresh after polling.
		pm, _, err = m.GetPluginManager(org, pmID)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to read plugin manager %d after connected poll", pmID), err.Error())
			return
		}
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

// ImportState supports: terraform import cycloid_plugin_manager.x <id>
func (r *pluginManagerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("expected a numeric plugin manager ID, got %q: %v", req.ID, err))
		return
	}
	org := r.provider.DefaultOrganization
	m := r.provider.Middleware
	pm, _, err := m.GetPluginManager(org, uint32(id))
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to read plugin manager %d for import", id), err.Error())
		return
	}
	var data pluginManagerResourceModel
	pluginManagerToModel(org, pm, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// pollPluginManagerConnected polls GetPluginManager until status == "connected".
func pollPluginManagerConnected(m middleware.Middleware, org string, id uint32, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pm, _, err := m.GetPluginManager(org, id)
		if err != nil {
			return err
		}
		if ptr.Value(pm.Status) == "connected" {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timeout waiting for plugin manager %d to reach connected status", id)
}

func pluginManagerToModel(org string, pm *models.PluginManager, data *pluginManagerResourceModel) {
	data.Organization = types.StringValue(org)
	data.ID = types.Int64Value(int64(ptr.Value(pm.ID)))
	data.Name = types.StringPointerValue(pm.Name)
	data.URL = types.StringValue(pm.URL.String())
	data.Status = types.StringPointerValue(pm.Status)
	data.CreatedAt = types.Int64Value(int64(ptr.Value(pm.CreatedAt)))
	data.UpdatedAt = types.Int64Value(int64(ptr.Value(pm.UpdatedAt)))
}
