package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cycloidio/cycloid-cli/gen/models"
	"github.com/cycloidio/terraform-provider-cycloid/datasource_plugin_widget_views"
	"github.com/cycloidio/cycloid-cli/utils/ptr"
)

var _ datasource.DataSource = &pluginWidgetViewsDataSource{}

type pluginWidgetViewsDatasourceModel = datasource_plugin_widget_views.PluginWidgetViewsModel

type pluginWidgetViewsDataSource struct {
	provider *CycloidProvider
}

func NewPluginWidgetViewsDataSource() datasource.DataSource {
	return &pluginWidgetViewsDataSource{}
}

func (s *pluginWidgetViewsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin_widget_views"
}

func (s *pluginWidgetViewsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_plugin_widget_views.PluginWidgetViewsDataSourceSchema(ctx)
}

func (s *pluginWidgetViewsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (s *pluginWidgetViewsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data pluginWidgetViewsDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*s.provider, data.Organization)
	m := s.provider.Client

	pluginInstallID := uint32(data.PluginInstallID.ValueInt64())

	views, _, err := m.ListPluginWidgetViews(org, pluginInstallID)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to list widget views for install %d in org %q", pluginInstallID, org),
			err.Error(),
		)
		return
	}

	listVal, diags := pluginWidgetViewsToListValue(ctx, views)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Organization = types.StringValue(org)
	data.WidgetViews = listVal
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func pluginWidgetViewsToListValue(ctx context.Context, views []*models.PluginWidgetView) (types.List, diag.Diagnostics) {
	items := make([]datasource_plugin_widget_views.WidgetViewItem, 0, len(views))
	for _, v := range views {
		items = append(items, datasource_plugin_widget_views.WidgetViewItem{
			ID:               types.Int64Value(int64(ptr.Value(v.ID))),
			Enabled:          types.BoolPointerValue(v.Enabled),
			EffectiveEnabled: types.BoolPointerValue(v.EffectiveEnabled),
			URLSlug:          types.StringPointerValue(v.URLSlug),
			EffectiveSlug:    types.StringPointerValue(v.EffectiveSlug),
			IsInherited:      types.BoolValue(v.IsInherited),
			HasOverride:      types.BoolValue(v.HasOverride),
		})
	}
	return types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: datasource_plugin_widget_views.WidgetViewAttrTypes(),
	}, items)
}
