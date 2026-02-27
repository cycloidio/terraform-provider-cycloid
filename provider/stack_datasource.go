package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/datasource_stacks"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ datasource.DataSource = &stackDataSource{}

type stackDataSource struct {
	provider *CycloidProvider
}

type stackDatasourceModel = datasource_stacks.StacksModel

func NewStacksDataSource() datasource.DataSource {
	return &stackDataSource{}
}

func (s stackDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stacks"
}

func (s *stackDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_stacks.StacksDataSourceSchema(ctx)
}

func (s *stackDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (s *stackDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data stackDatasourceModel

	// Read terraform configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if s.provider.Middleware == nil {
		return
	}

	mid := s.provider.Middleware

	org := getOrganizationCanonical(*s.provider, data.OrganizationCanonical)
	stacks, err := mid.ListStacks(org)
	if err != nil {
		resp.Diagnostics.AddError("failed to list stacks from api", err.Error())
		return
	}

	stacksValues, errDiags := dataStacksToListValue(ctx, stacks)
	resp.Diagnostics.Append(errDiags...)
	attrType := basetypes.ObjectType{
		AttrTypes: datasource_stacks.StacksValue{}.AttributeTypes(ctx),
	}
	listValue, errDiags := types.ListValueFrom(ctx, attrType, stacksValues)
	if errDiags.HasError() {
		resp.Diagnostics.Append(errDiags...)
		return
	}

	data.Stacks = listValue

	// Save data to terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func dataStacksToListValue(ctx context.Context, stacks []*models.ServiceCatalog) (basetypes.ListValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	stackType := datasource_stacks.StacksValue{}.AttributeTypes(ctx)
	teamType := map[string]attr.Type{"canonical": basetypes.StringType{}}

	stackElements := make([]attr.Value, len(stacks))
	for index, s := range stacks {
		cloudProviderValues, errDiags := dataStackCloudProvidersToListValue(ctx, s.CloudProviders)
		if errDiags.HasError() {
			return basetypes.ListValue{}, errDiags
		}

		technologiesValues, errDiags := dataStackTechnologiesToListValue(ctx, s.Technologies)
		if errDiags.HasError() {
			return basetypes.ListValue{}, errDiags
		}

		keywordsValues, errDiags := types.ListValueFrom(ctx, types.StringType, s.Keywords)
		if errDiags.HasError() {
			return basetypes.ListValue{}, errDiags
		}

		dependenciesValues, errDiags := dataStackDependenciesToListValue(ctx, s.Dependencies)
		if errDiags.HasError() {
			return basetypes.ListValue{}, errDiags
		}

		teamCan := ""
		if s.Team != nil {
			teamCan = *s.Team.Canonical
		}

		teamValue, errDiags := types.ObjectValue(
			teamType,
			map[string]attr.Value{
				"canonical": types.StringValue(teamCan),
			},
		)
		if errDiags.HasError() {
			return basetypes.ListValue{}, errDiags
		}

		stackElements[index], errDiags = datasource_stacks.NewStacksValue(
			stackType,
			map[string]attr.Value{
				"author":                 types.StringValue(*s.Author),
				"blueprint":              types.BoolValue(s.Blueprint),
				"canonical":              types.StringValue(*s.Canonical),
				"cloud_providers":        cloudProviderValues,
				"dependencies":           dependenciesValues,
				"description":            types.StringValue(s.Description),
				"directory":              types.StringValue(*s.Directory),
				"form_enabled":           types.BoolValue(*s.FormEnabled),
				"keywords":               keywordsValues,
				"name":                   types.StringValue(*s.Name),
				"organization_canonical": types.StringValue(*s.OrganizationCanonical),
				"quota_enabled":          types.BoolValue(*s.QuotaEnabled),
				"ref":                    types.StringValue(*s.Ref),
				"team":                   teamValue,
				"trusted":                types.BoolValue(*s.Trusted),
				"technologies":           technologiesValues,
				"visibility":             types.StringValue(*s.Visibility),
			},
		)
		diags.Append(errDiags...)
		if diags.HasError() {
			return basetypes.ListValue{}, diags
		}
	}

	return types.ListValueFrom(ctx, basetypes.ObjectType{
		AttrTypes: stackType,
	}, stackElements)
}

func dataStackTechnologiesToListValue(ctx context.Context, techs []*models.ServiceCatalogTechnology) (basetypes.ListValue, diag.Diagnostics) {
	techType := datasource_stacks.TechnologiesValue{}.AttributeTypes(ctx)
	stackTechnologies := make([]attr.Value, len(techs))
	for index, tech := range techs {
		tech, diag := types.ObjectValue(
			techType,
			map[string]attr.Value{
				"technology": types.StringValue(tech.Technology),
				"version":    types.StringValue(tech.Version),
			},
		)
		if diag.HasError() {
			return basetypes.ListValue{}, diag
		}

		stackTechnologies[index] = tech
	}

	return types.ListValueFrom(ctx, basetypes.ObjectType{
		AttrTypes: techType,
	}, stackTechnologies)
}

func dataStackDependenciesToListValue(ctx context.Context, dependencies []*models.ServiceCatalogDependency) (basetypes.ListValue, diag.Diagnostics) {
	dependencyType := datasource_stacks.DependenciesValue{}.AttributeTypes(ctx)
	stackDependencies := make([]attr.Value, len(dependencies))
	for index, dependency := range dependencies {
		tech, diag := types.ObjectValue(
			dependencyType,
			map[string]attr.Value{
				"ref":      types.StringValue(dependency.Ref),
				"required": types.BoolValue(dependency.Required),
			},
		)
		if diag.HasError() {
			return basetypes.ListValue{}, diag
		}

		stackDependencies[index] = tech
	}

	return types.ListValueFrom(ctx, basetypes.ObjectType{
		AttrTypes: dependencyType,
	}, stackDependencies)
}

func dataStackCloudProvidersToListValue(ctx context.Context, cloudProviders []*models.CloudProvider) (basetypes.ListValue, diag.Diagnostics) {
	cloudProviderType := datasource_stacks.CloudProvidersValue{}.AttributeTypes(ctx)
	stackCloudProviders := make([]attr.Value, len(cloudProviders))
	for index, cloudProvider := range cloudProviders {
		regions, diag := types.ListValueFrom(ctx, types.StringType, cloudProvider.Regions)
		if diag.HasError() {
			return basetypes.ListValue{}, diag
		}

		tech, diag := types.ObjectValue(
			cloudProviderType,
			map[string]attr.Value{
				"abbreviation": types.StringValue(cloudProvider.Abbreviation),
				"canonical":    types.StringValue(*cloudProvider.Canonical),
				"name":         types.StringValue(*cloudProvider.Name),
				"regions":      regions,
			},
		)
		if diag.HasError() {
			return basetypes.ListValue{}, diag
		}

		stackCloudProviders[index] = tech
	}

	return types.ListValueFrom(
		ctx,
		basetypes.ObjectType{
			AttrTypes: cloudProviderType,
		},
		stackCloudProviders)
}
