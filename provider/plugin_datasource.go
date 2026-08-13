package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cycloidio/terraform-provider-cycloid/datasource_plugin"
	"github.com/cycloidio/cycloid-cli/utils/ptr"
)

var _ datasource.DataSource = &pluginDataSource{}

type pluginDatasourceModel = datasource_plugin.PluginModel

type pluginDataSource struct {
	provider *CycloidProvider
}

func NewPluginDataSource() datasource.DataSource {
	return &pluginDataSource{}
}

func (s *pluginDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin"
}

func (s *pluginDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_plugin.PluginDataSourceSchema(ctx)
}

func (s *pluginDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (s *pluginDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data pluginDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*s.provider, data.Organization)
	m := s.provider.Client

	plugins, _, err := m.ListPlugins(org)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list plugins in org %q", org), err.Error())
		return
	}

	name := data.Name.ValueString()
	for _, p := range plugins {
		if p.Install == nil || ptr.Value(p.Name) != name {
			continue
		}
		data.Organization = types.StringValue(org)
		data.ID = types.Int64Value(int64(ptr.Value(p.Install.ID)))
		if p.Install.UUID != nil {
			data.UUID = types.StringValue(p.Install.UUID.String())
		}
		data.Status = types.StringPointerValue(p.Install.Status)
		data.CreatedAt = types.Int64Value(int64(ptr.Value(p.Install.CreatedAt)))
		data.UpdatedAt = types.Int64Value(int64(ptr.Value(p.Install.UpdatedAt)))
		data.PmSecret = types.StringPointerValue(p.Install.PmSecret)

		if p.Registry != nil {
			data.RegistryID = types.Int64Value(int64(ptr.Value(p.Registry.ID)))
		}
		data.PluginID = types.Int64Value(int64(ptr.Value(p.ID)))

		if p.Install.Version != nil {
			data.PluginVersionID = types.Int64Value(int64(ptr.Value(p.Install.Version.ID)))
			data.VersionName = types.StringPointerValue(p.Install.Version.Name)
			data.VersionStatus = types.StringPointerValue(p.Install.Version.Status)
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	resp.Diagnostics.AddError(
		"Installed plugin not found",
		fmt.Sprintf("no installed plugin named %q in org %q", name, org),
	)
}
