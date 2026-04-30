package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/terraform-provider-cycloid/datasource_plugin_registry_plugin"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &pluginRegistryPluginDataSource{}

type pluginRegistryPluginDatasourceModel = datasource_plugin_registry_plugin.PluginRegistryPluginModel

type pluginRegistryPluginDataSource struct {
	provider *CycloidProvider
}

func NewPluginRegistryPluginDataSource() datasource.DataSource {
	return &pluginRegistryPluginDataSource{}
}

func (s *pluginRegistryPluginDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin_registry_plugin"
}

func (s *pluginRegistryPluginDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_plugin_registry_plugin.PluginRegistryPluginDataSourceSchema(ctx)
}

func (s *pluginRegistryPluginDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	s.provider = pv
}

func (s *pluginRegistryPluginDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data pluginRegistryPluginDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*s.provider, data.Organization)
	m := s.provider.Middleware

	registryID := uint32(data.RegistryID.ValueInt64())
	plugins, _, err := m.ListRegistryPlugins(org, registryID)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to list plugins in registry %d in org %q", registryID, org),
			err.Error(),
		)
		return
	}

	name := data.Name.ValueString()
	for _, p := range plugins {
		if ptr.Value(p.Name) != name {
			continue
		}
		data.Organization = types.StringValue(org)
		data.ID = types.Int64Value(int64(ptr.Value(p.ID)))
		data.Name = types.StringPointerValue(p.Name)
		data.Owned = types.BoolPointerValue(p.Owned)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	resp.Diagnostics.AddError(
		"Plugin not found",
		fmt.Sprintf("no plugin named %q in registry %d in org %q", name, registryID, org),
	)
}
