package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cycloidio/terraform-provider-cycloid/datasource_plugin_registry"
	"github.com/cycloidio/cycloid-cli/utils/ptr"
)

var (
	_ datasource.DataSource                     = &pluginRegistryDataSource{}
	_ datasource.DataSourceWithConfigValidators = &pluginRegistryDataSource{}
)

type pluginRegistryDatasourceModel = datasource_plugin_registry.PluginRegistryModel

type pluginRegistryDataSource struct {
	provider *CycloidProvider
}

func NewPluginRegistryDataSource() datasource.DataSource {
	return &pluginRegistryDataSource{}
}

func (s *pluginRegistryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin_registry"
}

func (s *pluginRegistryDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_plugin_registry.PluginRegistryDataSourceSchema(ctx)
}

func (s *pluginRegistryDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("name"),
			path.MatchRoot("url"),
		),
	}
}

func (s *pluginRegistryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (s *pluginRegistryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data pluginRegistryDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*s.provider, data.Organization)
	m := s.provider.Middleware

	registries, _, err := m.ListPluginRegistries(org)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list plugin registries in org %q", org), err.Error())
		return
	}

	name := data.Name.ValueString()
	url := data.URL.ValueString()

	for _, reg := range registries {
		if name != "" && ptr.Value(reg.Name) != name {
			continue
		}
		if url != "" && reg.URL.String() != url {
			continue
		}
		data.Organization = types.StringValue(org)
		data.ID = types.Int64Value(int64(ptr.Value(reg.ID)))
		data.Name = types.StringPointerValue(reg.Name)
		data.URL = types.StringValue(reg.URL.String())
		data.Status = types.StringPointerValue(reg.Status)
		data.Access = types.BoolPointerValue(reg.Access)
		data.CreatedAt = types.Int64Value(int64(ptr.Value(reg.CreatedAt)))
		data.UpdatedAt = types.Int64Value(int64(ptr.Value(reg.UpdatedAt)))
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	resp.Diagnostics.AddError(
		"Plugin registry not found",
		fmt.Sprintf("no plugin registry matching name=%q url=%q in org %q", name, url, org),
	)
}
