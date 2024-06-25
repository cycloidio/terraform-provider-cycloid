package provider

import (
	"context"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/common"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
	"github.com/cycloidio/terraform-provider-cycloid/resource_catalog_repository"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

	cr, err := mid.CreateCatalogRepository(orgCan, name, url, branch, credCan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable create catalog repository",
			err.Error(),
		)
		return
	}

	catalogRepositoryCYModelToData(orgCan, cr, &data)

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

	catalogRepositoryCYModelToData(orgCan, cr, &data)

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

	catalogRepositoryCYModelToData(orgCan, cr, &data)

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

	data.Name = types.StringPointerValue(cr.Name)
	data.Url = types.StringPointerValue(cr.URL)
	data.Branch = types.StringValue(cr.Branch)
	data.Canonical = types.StringPointerValue(cr.Canonical)
	data.Owner = types.StringPointerValue(cr.Owner.Username)
	data.OrganizationCanonical = types.StringValue(org)
	data.CredentialCanonical = types.StringValue(cr.CredentialCanonical)

	return diags
}
