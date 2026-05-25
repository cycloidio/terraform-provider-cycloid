package provider

import (
	"context"
	"fmt"
	"slices"
	stderrors "errors"
	"net/http"

	"github.com/cycloidio/cycloid-cli/client/models"
	middleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/internal/icons"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_environment"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const defaultEnvType = "production" // TODO(meta-gov-env): remove once backend infers type from canonical

var _ resource.Resource = (*environmentResource)(nil)

func NewEnvironmentResource() resource.Resource {
	return &environmentResource{}
}

type environmentResourceModel resource_environment.EnvironmentModel

type environmentResource struct {
	provider *CycloidProvider
}

func (p *environmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (p *environmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_environment.EnvironmentResourceSchema(ctx)
}

func (p *environmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	p.provider = pv
}

func (p *environmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data environmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := p.provider.Middleware
	canonical := data.Canonical.ValueString()
	org := getOrganizationCanonical(*p.provider, data.Organization)
	project := data.Project.ValueString()

	projects, _, err := m.ListProjects(org)
	if err != nil {
		resp.Diagnostics.AddError("failed to fetch project from API", err.Error())
		return
	}

	if i := slices.IndexFunc(projects, func(p *models.Project) bool {
		return ptr.Value(p.Canonical) == project
	}); i == -1 {
		resp.Diagnostics.Append(environmentToValue(ctx, org, project, &models.Environment{}, &data)...)
		return
	}

	projectEnvs, _, err := m.ListProjectEnvs(org, project)
	if err != nil {
		resp.Diagnostics.AddError("failed to fetch environment from API while reading state", err.Error())
		return
	}

	if i := slices.IndexFunc(projectEnvs, func(e *models.ProjectEnvironment) bool {
		return ptr.Value(e.Canonical) == canonical
	}); i == -1 {
		resp.Diagnostics.Append(environmentToValue(ctx, org, project, &models.Environment{}, &data)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		environment, _, err := m.GetOrgEnv(org, canonical)
		if err != nil {
			resp.Diagnostics.AddError("failed to fetch org environment from API while reading state", err.Error())
			return
		}
		resp.Diagnostics.Append(environmentToValue(ctx, org, project, environment, &data)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (p *environmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data environmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	canonical := data.Canonical.ValueString()
	org := getOrganizationCanonical(*p.provider, data.Organization)
	project := data.Project.ValueString()
	color := data.Color.ValueString()
	if color == "" {
		color = icons.RandomColor()
	}

	data, d := p.createOrUpdateEnvironment(ctx, org, name, canonical, project, color, false)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (p *environmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data environmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	canonical := data.Canonical.ValueString()
	org := getOrganizationCanonical(*p.provider, data.Organization)
	project := data.Project.ValueString()
	color := data.Color.ValueString()
	if color == "" {
		color = icons.RandomColor()
	}

	data, d := p.createOrUpdateEnvironment(ctx, org, name, canonical, project, color, true)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (p *environmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data environmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := p.provider.Middleware
	org := getOrganizationCanonical(*p.provider, data.Organization)
	project := data.Project.ValueString()
	canonical := data.Canonical.ValueString()

	_, err := m.UnlinkEnvFromProject(org, project, canonical, middleware.DeleteOptions{})
	if err != nil {
		resp.Diagnostics.AddError("failed to unlink environment from project while deleting resource", err.Error())
		return
	}

	resp.Diagnostics.Append(environmentToValue(ctx, org, project, &models.Environment{}, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func environmentColor(environment *models.Environment) *string {
	if environment == nil || environment.EnvironmentType == nil {
		return nil
	}
	return environment.EnvironmentType.Color
}

func environmentToValue(_ context.Context, org, project string, environment *models.Environment, data *environmentResourceModel) diag.Diagnostics {
	if environment == nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("environment is nil for convertion", "This should not happend, please contact the plugin maintainer.")}
	}
	data.Canonical = types.StringPointerValue(environment.Canonical)
	data.Name = types.StringValue(environment.Name)
	data.Organization = types.StringValue(org)
	data.Color = types.StringPointerValue(environmentColor(environment))
	data.Project = types.StringValue(project)
	return nil
}

func envTypeFromCurrent(environment *models.Environment) string {
	if environment != nil && environment.EnvironmentType != nil && environment.EnvironmentType.Canonical != nil {
		return *environment.EnvironmentType.Canonical
	}
	return defaultEnvType
}

func (p *environmentResource) createOrUpdateEnvironment(ctx context.Context, org, name, canonical, project, color string, isUpdate bool) (environmentResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var data environmentResourceModel
	var err error

	name, canonical, err = NameOrCanonical(name, canonical)
	if err != nil {
		diags.AddError("failed to infer canonical", err.Error())
		return data, diags
	}

	m := p.provider.Middleware
	current, _, err := m.GetOrgEnv(org, canonical)
	if err == nil {
		updateBody := &models.UpdateEnvironment{
			Name: ptr.Ptr(name),
			Type: ptr.Ptr(envTypeFromCurrent(current)),
		}
		current, _, err = m.UpdateOrgEnv(org, canonical, updateBody)
		if err != nil {
			diags.AddError("failed to update environment from API", err.Error())
			return data, diags
		}
	} else {
		var apiErr *middleware.APIResponseError
		if !stderrors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
			diags.AddError("failed to fetch environment from API while updating resource", err.Error())
			return data, diags
		}
		if isUpdate {
			tflog.Info(ctx, "did not find current environment, assuming it had been deleted outside the provider, re-creating...", nil)
		}

		createBody := &models.NewEnvironment{
			Canonical: canonical,
			Name:      ptr.Ptr(name),
			Type:      ptr.Ptr(defaultEnvType),
		}
		current, _, err = m.CreateOrgEnv(org, createBody)
		if err != nil {
			diags.AddError("failed to create environment from API", err.Error())
			return data, diags
		}
	}

	if _, err = m.LinkEnvToProject(org, project, canonical); err != nil {
		diags.AddError("failed to link environment to project from API", err.Error())
		return data, diags
	}

	environment, _, err := m.GetOrgEnv(org, canonical)
	if err != nil {
		diags.AddError("failed to fetch environment from API after create/update", err.Error())
		return data, diags
	}

	_ = color // color is deprecated on environments; kept in schema for backward compatibility
	diags.Append(environmentToValue(ctx, org, project, environment, &data)...)
	return data, diags
}
