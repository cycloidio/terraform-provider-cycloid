package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/cycloidio/cycloid-cli/client/client/organization_service_catalog_sources"
	"github.com/cycloidio/cycloid-cli/client/models"
	cycloidmiddleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/resource_catalog_repository"
	strfmt "github.com/go-openapi/strfmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/pkg/errors"
)

var _ resource.Resource = (*catalogRepositoryResource)(nil)

func NewCatalogRepositoryResource() resource.Resource {
	return &catalogRepositoryResource{}
}

type catalogRepositoryResource struct {
	provider *CycloidProvider
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

func (r *catalogRepositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.provider = pv
}

func (r *catalogRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data catalogRepositoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configData catalogRepositoryResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &configData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgCan := getOrganizationCanonical(*r.provider, data.OrganizationCanonical)
	name := data.Name.ValueString()
	url := data.Url.ValueString()
	branch := data.Branch.ValueString()
	credCan := data.CredentialCanonical.ValueString()
	visibility := data.OnCreateVisibility.ValueString()
	team := data.OnCreateTeam.ValueString()
	owner, ownerConfigured := configuredCatalogRepositoryOwner(configData.Owner)

	cr, err := r.createCatalogRepository(orgCan, name, url, branch, credCan, visibility, team, owner)
	if err != nil {
		if ownerConfigured {
			resp.Diagnostics.AddError(
				"Unable create catalog repository with requested owner",
				fmt.Sprintf("Unable to assign owner %q to catalog repository %q in organization %q: %s. A common cause is that this user is not invited in the target organization.", owner, name, orgCan, err.Error()),
			)
			return
		}
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
	mid := r.provider.Middleware
	can := data.Canonical.ValueString()
	orgCan := getOrganizationCanonical(*r.provider, data.OrganizationCanonical)

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

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configData catalogRepositoryResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &configData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgCan := getOrganizationCanonical(*r.provider, data.OrganizationCanonical)
	name := data.Name.ValueString()
	url := data.Url.ValueString()
	branch := data.Branch.ValueString()
	credCan := data.CredentialCanonical.ValueString()
	owner, ownerConfigured := configuredCatalogRepositoryOwner(configData.Owner)
	can := data.Canonical.ValueString()

	if can == "" {
		var plandata catalogRepositoryResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &plandata)...)
		if resp.Diagnostics.HasError() {
			return
		}
		can = plandata.Canonical.ValueString()
	}

	cr, err := r.updateCatalogRepository(orgCan, can, name, url, branch, credCan, owner)
	if err != nil {
		if ownerConfigured {
			resp.Diagnostics.AddError(
				"Unable update catalog repository with requested owner",
				fmt.Sprintf("Unable to assign owner %q to catalog repository %q in organization %q: %s. A common cause is that this user is not invited in the target organization.", owner, can, orgCan, err.Error()),
			)
			return
		}
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
	mid := r.provider.Middleware

	can := data.Canonical.ValueString()
	orgCan := getOrganizationCanonical(*r.provider, data.OrganizationCanonical)

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

	if cr.Owner != nil && cr.Owner.Username != nil {
		data.Owner = types.StringPointerValue(cr.Owner.Username)
	} else {
		data.Owner = types.StringValue("")
	}

	stackCount := int64(0)
	if cr.StackCount != nil {
		stackCount = int64(*cr.StackCount)
	}

	data.Name = types.StringPointerValue(cr.Name)
	data.Url = types.StringPointerValue(cr.URL)
	data.Branch = types.StringValue(cr.Branch)
	data.Canonical = types.StringPointerValue(cr.Canonical)
	data.OrganizationCanonical = types.StringValue(org)
	data.CredentialCanonical = types.StringValue(cr.CredentialCanonical)

	stacksValue, diagErr := crStacksToListValue(ctx, cr.ServiceCatalogs)
	diags.Append(diagErr...)

	dataValue, diagErr := resource_catalog_repository.NewDataValue(
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
			"name":                 types.StringPointerValue(cr.Name),
			"branch":               types.StringValue(cr.Branch),
			"canonical":            types.StringPointerValue(cr.Canonical),
			"credential_canonical": types.StringValue(cr.CredentialCanonical),
			"stack_count":          types.Int64Value(stackCount),
			"url":                  types.StringPointerValue(cr.URL),
			"stacks":               stacksValue,
		})
	diags.Append(diagErr...)
	data.Data = dataValue

	return diags
}

func configuredCatalogRepositoryOwner(owner types.String) (string, bool) {
	if owner.IsNull() || owner.IsUnknown() {
		return "", false
	}

	ownerValue := owner.ValueString()
	if ownerValue == "" {
		return "", false
	}

	return ownerValue, true
}

func (r *catalogRepositoryResource) createCatalogRepository(org, name, url, branch, cred, visibility, teamCanonical, owner string) (*models.ServiceCatalogSource, error) {
	params := organization_service_catalog_sources.NewCreateServiceCatalogSourceParams()
	params.SetOrganizationCanonical(org)

	body := &models.NewServiceCatalogSource{
		Branch: &branch,
		Name:   &name,
		URL:    &url,
	}
	if cred != "" {
		body.CredentialCanonical = cred
	}
	if owner != "" {
		body.Owner = owner
	}
	switch visibility {
	case "shared", "local", "hidden":
		body.Visibility = visibility
	case "":
	default:
		return nil, errors.New("invalid visibility parameter for CreateCatalogRepository, accepted values are 'local', 'shared' or 'hidden'")
	}
	if teamCanonical != "" {
		body.TeamCanonical = teamCanonical
	}

	params.SetBody(body)
	if err := body.Validate(strfmt.Default); err != nil {
		return nil, err
	}

	resp, err := r.provider.APIClient.OrganizationServiceCatalogSources.CreateServiceCatalogSource(params, r.provider.APIClient.Credentials(&org))
	if err != nil {
		return nil, cycloidmiddleware.NewAPIError(err)
	}

	return resp.GetPayload().Data, nil
}

func (r *catalogRepositoryResource) updateCatalogRepository(org, catalogRepo, name, url, branch, cred, owner string) (*models.ServiceCatalogSource, error) {
	params := organization_service_catalog_sources.NewUpdateServiceCatalogSourceParams()
	params.SetOrganizationCanonical(org)
	params.SetServiceCatalogSourceCanonical(catalogRepo)

	body := &models.UpdateServiceCatalogSource{
		Branch:              branch,
		CredentialCanonical: cred,
		Name:                &name,
		URL:                 &url,
	}
	if owner != "" {
		body.Owner = owner
	}

	params.SetBody(body)
	if err := body.Validate(strfmt.Default); err != nil {
		return nil, err
	}

	resp, err := r.provider.APIClient.OrganizationServiceCatalogSources.UpdateServiceCatalogSource(params, r.provider.APIClient.Credentials(&org))
	if err != nil {
		return nil, cycloidmiddleware.NewAPIError(err)
	}

	return resp.GetPayload().Data, nil
}

func crStacksToListValue(ctx context.Context, stacks []*models.ServiceCatalog) (basetypes.ListValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	stackType := resource_catalog_repository.StacksValue{}.AttributeTypes(ctx)

	stackElements := make([]attr.Value, len(stacks))
	for index, s := range stacks {
		var errDiags diag.Diagnostics
		stackElements[index], errDiags = resource_catalog_repository.NewStacksValue(
			stackType,
			map[string]attr.Value{
				"canonical": types.StringPointerValue(s.Canonical),
				"ref":       types.StringPointerValue(s.Ref),
			},
		)
		diags.Append(errDiags...)
		if diags.HasError() {
			return basetypes.ListValue{}, diags
		}
	}

	return types.ListValueFrom(ctx, resource_catalog_repository.StacksType{
		ObjectType: types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"canonical": types.StringType,
				"ref":       types.StringType,
			}},
	}, stackElements)
}
