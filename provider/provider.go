package provider

import (
	"context"

	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = (*cycloidProvider)(nil)

func New() func() provider.Provider {
	return func() provider.Provider {
		return &cycloidProvider{}
	}
}

type cycloidProvider struct{}

func (p *cycloidProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = provider_cycloid.CycloidProviderSchema(ctx)
}

func (p *cycloidProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data provider_cycloid.CycloidModel

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	//api := common.NewAPI(common.WithURL(data.Url))
	resp.ResourceData = data
	resp.DataSourceData = data
}

func (p *cycloidProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cycloid"
	//resp.Version = p.version
}

func (p *cycloidProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *cycloidProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewOrganizationResource,
		NewCredentialResource,
	}
}
