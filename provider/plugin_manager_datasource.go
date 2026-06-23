package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cycloidio/terraform-provider-cycloid/datasource_plugin_manager"
	"github.com/cycloidio/cycloid-cli/utils/ptr"
)

var _ datasource.DataSource = &pluginManagerDataSource{}

type pluginManagerDatasourceModel = datasource_plugin_manager.PluginManagerModel

type pluginManagerDataSource struct {
	provider *CycloidProvider
}

func NewPluginManagerDataSource() datasource.DataSource {
	return &pluginManagerDataSource{}
}

func (s *pluginManagerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin_manager"
}

func (s *pluginManagerDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_plugin_manager.PluginManagerDataSourceSchema(ctx)
}

func (s *pluginManagerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (s *pluginManagerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data pluginManagerDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*s.provider, data.Organization)
	m := s.provider.Middleware

	managers, _, err := m.ListPluginManagers(org)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list plugin managers in org %q", org), err.Error())
		return
	}

	name := data.Name.ValueString()
	for _, pm := range managers {
		if ptr.Value(pm.Name) != name {
			continue
		}
		data.Organization = types.StringValue(org)
		data.ID = types.Int64Value(int64(ptr.Value(pm.ID)))
		data.Name = types.StringPointerValue(pm.Name)
		data.URL = types.StringValue(pm.URL.String())
		data.Status = types.StringPointerValue(pm.Status)
		data.CreatedAt = types.Int64Value(int64(ptr.Value(pm.CreatedAt)))
		data.UpdatedAt = types.Int64Value(int64(ptr.Value(pm.UpdatedAt)))
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	resp.Diagnostics.AddError(
		"Plugin manager not found",
		fmt.Sprintf("no plugin manager named %q in org %q", name, org),
	)
}
