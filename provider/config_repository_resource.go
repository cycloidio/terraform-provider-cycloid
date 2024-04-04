package provider

import (
	"context"
	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/common"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
	"github.com/cycloidio/terraform-provider-cycloid/resource_config_repository"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ resource.Resource = (*configRepositoryResource)(nil)

func NewConfigRepositoryResource() resource.Resource {
	return &configRepositoryResource{}
}

type configRepositoryResource struct {
	provider provider_cycloid.CycloidModel
}

type configRepositoryResourceModel resource_config_repository.ConfigRepositoryModel

func (r *configRepositoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_repository"
}

func (r *configRepositoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_config_repository.ConfigRepositoryResourceSchema(ctx)
}

func (r *configRepositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
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

func (r *configRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data configRepositoryResourceModel

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
	def := data.Default.ValueBool()

	cr, err := mid.CreateConfigRepository(orgCan, name, url, branch, credCan, def)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable create config repository",
			err.Error(),
		)
		return
	}

	configRepositoryCYModelToData(orgCan, cr, &data)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *configRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data configRepositoryResourceModel

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

	cr, err := mid.GetConfigRepository(orgCan, can)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable read config repository",
			err.Error(),
		)
		return
	}

	configRepositoryCYModelToData(orgCan, cr, &data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *configRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data configRepositoryResourceModel

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
	def := data.Default.ValueBool()

	if can == "" {
		var plandata configRepositoryResourceModel
		// Read Terraform prior state data into the model
		resp.Diagnostics.Append(req.State.Get(ctx, &plandata)...)
		if resp.Diagnostics.HasError() {
			return
		}
		can = plandata.Canonical.ValueString()
	}

	cr, err := mid.UpdateConfigRepository(orgCan, can, credCan, name, url, branch, def)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update config repository",
			err.Error(),
		)
		return
	}

	configRepositoryCYModelToData(orgCan, cr, &data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *configRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data configRepositoryResourceModel

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

	err := mid.DeleteConfigRepository(orgCan, can)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable delete config repository",
			err.Error(),
		)
		return
	}
}

func configRepositoryCYModelToData(org string, cr *models.ConfigRepository, data *configRepositoryResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.Name = types.StringPointerValue(cr.Name)
	data.Url = types.StringPointerValue(cr.URL)
	data.Branch = types.StringValue(cr.Branch)
	data.Canonical = types.StringPointerValue(cr.Canonical)
	data.OrganizationCanonical = types.StringValue(org)
	data.Default = types.BoolPointerValue(cr.Default)
	data.CredentialCanonical = types.StringValue(cr.CredentialCanonical)

	return diags
}
