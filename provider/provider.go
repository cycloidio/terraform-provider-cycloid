package provider

import (
	"context"
	"os"

	"github.com/cycloidio/cycloid-cli/cmd/cycloid/common"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = (*CycloidProvider)(nil)

func New() func() provider.Provider {
	return func() provider.Provider {
		return &CycloidProvider{}
	}
}

type CycloidProvider struct {
	Test                types.String `tfsdk:"test"`
	APIKey              string
	APIUrl              string
	DefaultOrganization string
	Insecure            bool
	APIClient           *common.APIClient
	Middleware          middleware.Middleware
}

func (p *CycloidProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cycloid"
}

func (p *CycloidProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = provider_cycloid.CycloidProviderSchema(ctx)
}

func (p *CycloidProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data provider_cycloid.CycloidModel

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	p.APIKey = os.Getenv("CY_API_KEY")
	p.APIUrl = os.Getenv("CY_API_URL")
	p.DefaultOrganization = os.Getenv("CY_ORG")

	// Deprecated attributes, delete after next release
	if !data.OrganizationCanonical.IsUnknown() && !data.OrganizationCanonical.IsNull() {
		p.DefaultOrganization = data.OrganizationCanonical.ValueString()
	}

	if !data.Url.IsUnknown() && !data.Url.IsNull() {
		p.APIUrl = data.Jwt.ValueString()
	}

	if !data.Jwt.IsUnknown() && !data.Jwt.IsNull() {
		p.APIKey = data.APIKey.ValueString()
	}

	// New attributes takes precedence
	if !data.DefaultOrganization.IsUnknown() && !data.DefaultOrganization.IsNull() {
		p.DefaultOrganization = data.DefaultOrganization.ValueString()
	}

	if !data.APIUrl.IsUnknown() && !data.APIUrl.IsNull() {
		p.APIUrl = data.APIUrl.ValueString()
	}

	if !data.APIKey.IsUnknown() && !data.APIKey.IsNull() {
		p.APIKey = data.APIKey.ValueString()
	}

	if p.DefaultOrganization == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("default_organization"),
			"organization parameter is empty",
			"please fill it using `default_organization` attribute in the provider or `CY_ORG` environment variable.",
		)
		return
	}

	if p.APIKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"api_key parameter is empty",
			"please fill it using `api_key` attribute in the provider or `CY_API_KEY` environment variable.",
		)
		return
	}

	if p.APIUrl == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"api_url parameter is empty",
			"please fill it using `api_url` attribute in the provider or `CY_API_URL` environment variable.",
		)
		return
	}

	p.Insecure = data.Insecure.ValueBool()
	p.APIClient = common.NewAPI(
		common.WithURL(p.APIUrl),
		common.WithToken(p.APIKey),
		common.WithInsecure(p.Insecure),
	)
	p.Middleware = middleware.NewMiddleware(p.APIClient)

	p.Test = types.StringValue("test")

	resp.ResourceData = p
	resp.DataSourceData = p
}

func (p *CycloidProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewStacksDataSource,
		NewCredentialsDataSource,
		NewCredentialDataSource,
		NewTerraformOutputDataSource,
		NewTerraformOutputsDataSource,
	}
}

func (p *CycloidProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewOrganizationResource,
		NewCredentialResource,
		NewCatalogRepositoryResource,
		NewConfigRepositoryResource,
		NewExternalBackendResource,
		NewOrganizationMemberResource,
		NewStackResource,
		NewProjectResource,
		NewEnvironmentResource,
	}
}
