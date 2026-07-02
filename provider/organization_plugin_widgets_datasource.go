package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/datasource_organization_plugin_widgets"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &organizationPluginWidgetsDataSource{}

type organizationPluginWidgetsDatasourceModel = datasource_organization_plugin_widgets.OrganizationPluginWidgetsModel

type organizationPluginWidgetsDataSource struct {
	provider *CycloidProvider
}

func NewOrganizationPluginWidgetsDataSource() datasource.DataSource {
	return &organizationPluginWidgetsDataSource{}
}

func (s *organizationPluginWidgetsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_plugin_widgets"
}

func (s *organizationPluginWidgetsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_organization_plugin_widgets.OrganizationPluginWidgetsDataSourceSchema(ctx)
}

func (s *organizationPluginWidgetsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (s *organizationPluginWidgetsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationPluginWidgetsDatasourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*s.provider, data.Organization)
	placement := data.Placement.ValueString()

	widgets, _, err := s.provider.Middleware.ListPluginWidgets(org, placement)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to list plugin widgets for placement %q in org %q", placement, org),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(pluginWidgetsToData(ctx, org, widgets, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func pluginWidgetsToData(ctx context.Context, org string, widgets []*models.PluginWidget, data *organizationPluginWidgetsDatasourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	data.Organization = types.StringValue(org)

	widgetModels := make([]datasource_organization_plugin_widgets.PluginWidgetModel, 0, len(widgets))
	for _, widget := range widgets {
		if widget == nil {
			continue
		}

		placementValue, placementDiags := pluginWidgetJSONValue(widget.Placement)
		diags.Append(placementDiags...)
		widgetValue, widgetDiags := pluginWidgetJSONValue(widget.Widget)
		diags.Append(widgetDiags...)
		relationValue, relationDiags := pluginWidgetJSONValue(pluginWidgetRelationValue(widget.Relation))
		diags.Append(relationDiags...)
		if diags.HasError() {
			return diags
		}

		widgetModels = append(widgetModels, datasource_organization_plugin_widgets.PluginWidgetModel{
			ID:        types.Int64Value(int64(widget.ID)),
			Type:      types.StringPointerValue(widget.Type),
			Placement: placementValue,
			IsDefault: types.BoolPointerValue(widget.IsDefault),
			Widget:    widgetValue,
			Relation:  relationValue,
		})
	}

	widgetsList, listDiags := types.ListValueFrom(ctx, datasource_organization_plugin_widgets.PluginWidgetObjectType(), widgetModels)
	diags.Append(listDiags...)
	if diags.HasError() {
		return diags
	}
	data.Widgets = widgetsList

	return diags
}

func pluginWidgetJSONValue(value any) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value == nil {
		return types.StringNull(), diags
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		diags.AddError("Failed to encode plugin widget field", err.Error())
		return types.StringNull(), diags
	}
	return types.StringValue(string(encoded)), diags
}

func pluginWidgetRelationValue(relation *models.PluginRelation) any {
	if relation == nil {
		return nil
	}

	var enabled any
	if relation.Enabled != nil {
		enabled = *relation.Enabled
	}

	return map[string]any{
		"enabled":   enabled,
		"relations": relation.Relations,
	}
}
