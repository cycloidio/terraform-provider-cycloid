package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/terraform-provider-cycloid/datasource_environment"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &environmentDataSource{}

type environmentDataSource struct {
	provider *CycloidProvider
}

func NewEnvironmentDataSource() datasource.DataSource {
	return &environmentDataSource{}
}

func (s *environmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (s *environmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_environment.EnvironmentDataSourceSchema(ctx)
}

func (s *environmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (s *environmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasource_environment.EnvironmentModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*s.provider, data.Organization)
	canonical := data.Canonical.ValueString()

	env, _, err := s.provider.Middleware.GetOrgEnv(org, canonical)
	if err != nil {
		resp.Diagnostics.AddError("failed to read environment '"+canonical+"'", err.Error())
		return
	}

	data.Organization = types.StringValue(org)
	data.Canonical = types.StringPointerValue(env.Canonical)
	data.Name = types.StringValue(env.Name)
	data.Description = types.StringValue(env.Description)
	data.ResourcesCount = ptrUint32ToInt64(env.ResourcesCount)
	data.ID = ptrUint32ToInt64(env.ID)
	data.CreatedAt = ptrUint64ToInt64(env.CreatedAt)
	data.UpdatedAt = ptrUint64ToInt64(env.UpdatedAt)

	if env.EnvironmentType != nil {
		data.Type = types.StringPointerValue(env.EnvironmentType.Canonical)
	} else {
		data.Type = types.StringNull()
	}

	if env.Owner != nil && env.Owner.Username != nil {
		data.Owner = types.StringPointerValue(env.Owner.Username)
	} else {
		data.Owner = types.StringValue("")
	}

	canonicals := make([]string, 0, len(env.CloudAccounts))
	for _, ca := range env.CloudAccounts {
		if ca.Canonical != nil {
			canonicals = append(canonicals, *ca.Canonical)
		}
	}
	cloudAccList, caDiags := types.ListValueFrom(ctx, types.StringType, canonicals)
	resp.Diagnostics.Append(caDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.CloudAccountCanonicals = cloudAccList

	varList, varDiags := buildVariableList(ctx, env.Variables)
	resp.Diagnostics.Append(varDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Variables = varList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
