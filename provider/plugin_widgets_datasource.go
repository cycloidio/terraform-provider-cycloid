package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cycloidio/cycloid-cli/gen/models"
	"github.com/cycloidio/terraform-provider-cycloid/datasource_plugin_widgets"
)

var _ datasource.DataSource = &pluginWidgetsDataSource{}

type pluginWidgetsDatasourceModel = datasource_plugin_widgets.PluginWidgetsModel

type pluginWidgetsDataSource struct {
	provider *CycloidProvider
}

func NewPluginWidgetsDataSource() datasource.DataSource {
	return &pluginWidgetsDataSource{}
}

func (s *pluginWidgetsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin_widgets"
}

func (s *pluginWidgetsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_plugin_widgets.PluginWidgetsDataSourceSchema(ctx)
}

func (s *pluginWidgetsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (s *pluginWidgetsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data pluginWidgetsDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*s.provider, data.Organization)
	m := s.provider.Client

	placement := data.Placement.ValueString()

	widgets, _, err := m.ListPluginWidgets(org, placement)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to list plugin widgets in org %q with placement %q", org, placement),
			err.Error(),
		)
		return
	}

	if widgets == nil {
		widgets = []*models.PluginWidget{}
	}
	b, err := json.Marshal(widgets)
	if err != nil {
		resp.Diagnostics.AddError("failed to marshal plugin widgets to JSON", err.Error())
		return
	}

	data.Organization = types.StringValue(org)
	data.WidgetsJSON = types.StringValue(string(b))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
