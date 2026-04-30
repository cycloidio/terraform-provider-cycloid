package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/terraform-provider-cycloid/datasource_plugin_version"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &pluginVersionDataSource{}
var _ datasource.DataSourceWithConfigValidators = &pluginVersionDataSource{}

type pluginVersionDatasourceModel = datasource_plugin_version.PluginVersionModel

type pluginVersionDataSource struct {
	provider *CycloidProvider
}

func NewPluginVersionDataSource() datasource.DataSource {
	return &pluginVersionDataSource{}
}

func (s *pluginVersionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin_version"
}

func (s *pluginVersionDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_plugin_version.PluginVersionDataSourceSchema(ctx)
}

func (s *pluginVersionDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("name"),
			path.MatchRoot("url"),
		),
	}
}

func (s *pluginVersionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (s *pluginVersionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data pluginVersionDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*s.provider, data.Organization)
	m := s.provider.Middleware

	registryID := uint32(data.RegistryID.ValueInt64())
	pluginID := uint32(data.PluginID.ValueInt64())

	versions, _, err := m.ListPluginVersions(org, registryID, pluginID)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to list versions for plugin %d in registry %d in org %q", pluginID, registryID, org),
			err.Error(),
		)
		return
	}

	name := data.Name.ValueString()
	url := data.URL.ValueString()

	for _, v := range versions {
		if name != "" && ptr.Value(v.Name) != name {
			continue
		}
		if url != "" && v.URL.String() != url {
			continue
		}
		data.Organization = types.StringValue(org)
		data.ID = types.Int64Value(int64(ptr.Value(v.ID)))
		data.Name = types.StringPointerValue(v.Name)
		data.URL = types.StringValue(v.URL.String())
		data.Status = types.StringPointerValue(v.Status)
		data.Description = types.StringValue(v.Description)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	resp.Diagnostics.AddError(
		"Plugin version not found",
		fmt.Sprintf("no version matching name=%q url=%q for plugin %d in registry %d in org %q", name, url, pluginID, registryID, org),
	)
}
