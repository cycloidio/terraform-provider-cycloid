package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/datasource_environments"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &environmentsDataSource{}

type environmentsDataSource struct {
	provider *CycloidProvider
}

func NewEnvironmentsDataSource() datasource.DataSource {
	return &environmentsDataSource{}
}

func (s *environmentsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environments"
}

func (s *environmentsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_environments.EnvironmentsDataSourceSchema(ctx)
}

func (s *environmentsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

var environmentListObjAttrTypes = map[string]attr.Type{
	"canonical":       types.StringType,
	"name":            types.StringType,
	"description":     types.StringType,
	"type":            types.StringType,
	"owner":           types.StringType,
	"resources_count": types.Int64Type,
	"id":              types.Int64Type,
	"created_at":      types.Int64Type,
	"updated_at":      types.Int64Type,
}

func (s *environmentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasource_environments.EnvironmentsModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*s.provider, data.Organization)
	project := data.Project.ValueString()

	envs, _, err := s.provider.Middleware.ListOrgEnvs(org)
	if err != nil {
		resp.Diagnostics.AddError("failed to list environments", err.Error())
		return
	}

	// When project filter is set, restrict to envs linked to that project.
	var allowedCanonicals []string
	if project != "" {
		projEnvs, _, err := s.provider.Middleware.ListProjectEnvs(org, project)
		if err != nil {
			resp.Diagnostics.AddError("failed to list project environments for filter", err.Error())
			return
		}
		for _, pe := range projEnvs {
			if pe.Canonical != nil {
				allowedCanonicals = append(allowedCanonicals, *pe.Canonical)
			}
		}
	}

	items := make([]attr.Value, 0, len(envs))
	for _, env := range envs {
		if project != "" && !slices.Contains(allowedCanonicals, envCanonical(env)) {
			continue
		}

		obj, objDiags := types.ObjectValue(environmentListObjAttrTypes, envToListObj(env))
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		items = append(items, obj)
	}

	listVal, listDiags := types.ListValue(types.ObjectType{AttrTypes: environmentListObjAttrTypes}, items)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Organization = types.StringValue(org)
	data.Environments = listVal

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func envCanonical(env *models.Environment) string {
	if env.Canonical == nil {
		return ""
	}
	return *env.Canonical
}

func envToListObj(env *models.Environment) map[string]attr.Value {
	envType := types.StringNull()
	if env.EnvironmentType != nil {
		envType = types.StringPointerValue(env.EnvironmentType.Canonical)
	}

	owner := ""
	if env.Owner != nil && env.Owner.Username != nil {
		owner = *env.Owner.Username
	}

	return map[string]attr.Value{
		"canonical":       types.StringPointerValue(env.Canonical),
		"name":            types.StringValue(env.Name),
		"description":     types.StringValue(env.Description),
		"type":            envType,
		"owner":           types.StringValue(owner),
		"resources_count": ptrUint32ToInt64(env.ResourcesCount),
		"id":              ptrUint32ToInt64(env.ID),
		"created_at":      ptrUint64ToInt64(env.CreatedAt),
		"updated_at":      ptrUint64ToInt64(env.UpdatedAt),
	}
}
