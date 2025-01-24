package provider

import (
	"context"
	"regexp"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/common"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
	"github.com/cycloidio/terraform-provider-cycloid/resource_catalog_repository"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*catalogRepositoryResource)(nil)

func NewCatalogRepositoryResource() resource.Resource {
	return &catalogRepositoryResource{}
}

type catalogRepositoryResource struct {
	provider provider_cycloid.CycloidModel
}

type catalogRepositoryResourceModel resource_catalog_repository.CatalogRepositoryModel

func (r *catalogRepositoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalog_repository"
}

func (r *catalogRepositoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_catalog_repository.CatalogRepositoryResourceSchema(ctx)
	resp.Schema.Attributes["on_create_visibility"] = schema.StringAttribute{
		Default:             stringdefault.StaticString("local"),
		Optional:            true,
		Computed:            true,
		Description:         "Team responsible for the maintenance of the underlying service catalogs\n",
		MarkdownDescription: "Team responsible for the maintenance of the underlying service catalogs\n",
		Validators: []validator.String{
			stringvalidator.LengthBetween(3, 100),
			stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9]+[a-z0-9\-_]+[a-z0-9]+$`), ""),
		},
	}
	resp.Schema.Attributes["on_create_team"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Description:         "The visibility setting allows to specify which visibility will be applied to stacks in this catalog repository.\nThis option is only applied during initial catalog repository creation, not for subsequent updates.\n",
		MarkdownDescription: "The visibility setting allows to specify which visibility will be applied to stacks in this catalog repository.\nThis option is only applied during initial catalog repository creation, not for subsequent updates.\n",
		Default:             stringdefault.StaticString(""),
	}
}

func (r *catalogRepositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	pv, ok := req.ProviderData.(provider_cycloid.CycloidModel)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}
	r.provider = pv
}

func (r *catalogRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data catalogRepositoryResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create API call logic
	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)
	name := data.Name.ValueString()
	url := data.Url.ValueString()
	branch := data.Branch.ValueString()
	credCan := data.CredentialCanonical.ValueString()
	visibility := data.OnCreateVisibility.ValueString()
	team := data.OnCreateTeam.ValueString()

	cr, err := mid.CreateCatalogRepository(orgCan, name, url, branch, credCan, visibility, team)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable create catalog repository",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(catalogRepositoryCYModelToData(orgCan, cr, &data)...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *catalogRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data catalogRepositoryResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	can := data.Canonical.ValueString()

	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)

	cr, err := mid.GetCatalogRepository(orgCan, can)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable read catalog repository",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(catalogRepositoryCYModelToData(orgCan, cr, &data)...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *catalogRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data catalogRepositoryResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update API call logic
	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)
	name := data.Name.ValueString()
	url := data.Url.ValueString()
	branch := data.Branch.ValueString()
	credCan := data.CredentialCanonical.ValueString()
	can := data.Canonical.ValueString()

	if can == "" {
		var plandata catalogRepositoryResourceModel
		// Read Terraform prior state data into the model
		resp.Diagnostics.Append(req.State.Get(ctx, &plandata)...)
		if resp.Diagnostics.HasError() {
			return
		}
		can = plandata.Canonical.ValueString()
	}

	cr, err := mid.UpdateCatalogRepository(orgCan, can, name, url, branch, credCan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable update catalog repository",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(catalogRepositoryCYModelToData(orgCan, cr, &data)...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *catalogRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data catalogRepositoryResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	can := data.Canonical.ValueString()
	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)

	err := mid.DeleteCatalogRepository(orgCan, can)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable delete catalog repository",
			err.Error(),
		)
		return
	}
}

func catalogRepositoryCYModelToData(org string, cr *models.ServiceCatalogSource, data *catalogRepositoryResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	ctx := context.Background()

	if cr.Owner != nil {
		data.Owner = types.StringPointerValue(cr.Owner.Username)
	}

	data.Name = types.StringPointerValue(cr.Name)
	data.Url = types.StringPointerValue(cr.URL)
	data.Branch = types.StringValue(cr.Branch)
	data.Canonical = types.StringPointerValue(cr.Canonical)
	data.OrganizationCanonical = types.StringValue(org)
	data.CredentialCanonical = types.StringValue(cr.CredentialCanonical)
	if cr.Owner != nil {
		data.Owner = types.StringValue(*cr.Owner.FamilyName)
	}

	stacksValue, diagErr := crStacksToListValue(ctx, cr.ServiceCatalogs)
	diags.Append(diagErr...)

	dataValue, diagErr := resource_catalog_repository.NewDataValue(
		// CrDataValue{}.AttrTypes(ctx),
		map[string]attr.Type{
			"branch":               basetypes.StringType{},
			"canonical":            basetypes.StringType{},
			"credential_canonical": basetypes.StringType{},
			"name":                 basetypes.StringType{},
			"stack_count":          basetypes.Int64Type{},
			"stacks": basetypes.ListType{
				ElemType: resource_catalog_repository.StacksValue{}.Type(ctx),
			},
			"url": basetypes.StringType{},
		},
		map[string]attr.Value{
			"name":                 types.StringValue(*cr.Canonical),
			"branch":               types.StringValue(cr.Branch),
			"canonical":            types.StringValue(*cr.Canonical),
			"credential_canonical": types.StringValue(*cr.Canonical),
			"stack_count":          types.Int64Value(int64(*cr.StackCount)),
			"url":                  types.StringValue(*cr.URL),
			"stacks":               stacksValue,
		})
	diags.Append(diagErr...)
	data.Data = dataValue

	return diags
}

func crStacksToListValue(ctx context.Context, stacks []*models.ServiceCatalog) (basetypes.ListValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	stackType := resource_catalog_repository.StacksValue{}.AttributeTypes(ctx)
	teamType := map[string]attr.Type{"canonical": basetypes.StringType{}}

	stackElements := make([]attr.Value, len(stacks))
	for index, s := range stacks {
		cloudProviderValues, errDiags := stackCloudProvidersToListValue(ctx, s.CloudProviders)
		if errDiags.HasError() {
			return basetypes.ListValue{}, errDiags
		}

		technologiesValues, errDiags := crStackTechnologiesToListValue(ctx, s.Technologies)
		if errDiags.HasError() {
			return basetypes.ListValue{}, errDiags
		}

		keywordsValues, errDiags := types.ListValueFrom(ctx, types.StringType, s.Keywords)
		if errDiags.HasError() {
			return basetypes.ListValue{}, errDiags
		}

		dependenciesValues, errDiags := crStackDependenciesToListValue(ctx, s.Dependencies)
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

		stackElements[index], errDiags = resource_catalog_repository.NewStacksValue(
			stackType,
			map[string]attr.Value{
				"author":                 types.StringValue(*s.Author),
				"blueprint":              types.BoolValue(s.Blueprint),
				"canonical":              types.StringValue(*s.Canonical),
				"cloud_providers":        cloudProviderValues,
				"dependencies":           dependenciesValues,
				"description":            types.StringValue(*s.Description),
				"directory":              types.StringValue(*s.Directory),
				"form_enabled":           types.BoolValue(*s.FormEnabled),
				"keywords":               keywordsValues,
				"name":                   types.StringValue(*s.Name),
				"organization_canonical": types.StringValue(*s.OrganizationCanonical),
				"quota_enabled":          types.BoolValue(*s.QuotaEnabled),
				"readme":                 types.StringValue(s.Readme),
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

	return types.ListValueFrom(ctx, resource_catalog_repository.StacksType{basetypes.ObjectType{
		AttrTypes: stackType,
	}}, stackElements)
}

func crStackTechnologiesToListValue(ctx context.Context, techs []*models.ServiceCatalogTechnology) (basetypes.ListValue, diag.Diagnostics) {
	techType := resource_catalog_repository.TechnologiesValue{}.AttributeTypes(ctx)
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

	return types.ListValueFrom(ctx, resource_catalog_repository.TechnologiesType{basetypes.ObjectType{
		AttrTypes: techType,
	}}, stackTechnologies)
}

func crStackDependenciesToListValue(ctx context.Context, dependencies []*models.ServiceCatalogDependency) (basetypes.ListValue, diag.Diagnostics) {
	dependencyType := resource_catalog_repository.DependenciesValue{}.AttributeTypes(ctx)
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

	return types.ListValueFrom(ctx, resource_catalog_repository.DependenciesType{basetypes.ObjectType{
		AttrTypes: dependencyType,
	}}, stackDependencies)
}

func stackCloudProvidersToListValue(ctx context.Context, cloudProviders []*models.CloudProvider) (basetypes.ListValue, diag.Diagnostics) {
	cloudProviderType := resource_catalog_repository.CloudProvidersValue{}.AttributeTypes(ctx)
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
		resource_catalog_repository.CloudProvidersType{
			basetypes.ObjectType{
				AttrTypes: cloudProviderType,
			},
		},
		stackCloudProviders)
}
