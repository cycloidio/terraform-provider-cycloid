package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/icons"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_environment"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

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

	// Check that the project exists
	projects, err := m.ListProjects(org)
	if err != nil {
		resp.Diagnostics.AddError("failed to fetch project from API", err.Error())
		return
	}

	if i := slices.IndexFunc(projects, func(p *models.Project) bool {
		return ptr.Value(p.Canonical) == project
	}); i == -1 {
		// Project doesn't exist, so empty state
		// Environment doesn't exist, so empty state
		resp.Diagnostics.Append(environmentToValue(ctx, org, project, &models.Environment{}, &data)...)
		return
	}

	environments, err := m.ListProjectsEnv(org, project)
	if err != nil {
		resp.Diagnostics.AddError("failed to fetch environment from API while reading state", err.Error())
		return
	}

	if i := slices.IndexFunc(environments, func(e *models.Environment) bool {
		return ptr.Value(e.Canonical) == canonical
	}); i == -1 {
		// Environment doesn't exist, so empty state
		resp.Diagnostics.Append(environmentToValue(ctx, org, project, &models.Environment{}, &data)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		resp.Diagnostics.Append(environmentToValue(ctx, org, project, environments[i], &data)...)
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
		color = icons.RandomIcon()
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

	err := m.DeleteEnv(org, project, canonical)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete environment from API while deleting resource", err.Error())
		return
	}

	resp.Diagnostics.Append(environmentToValue(ctx, org, project, &models.Environment{}, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func environmentToValue(_ context.Context, org, project string, environment *models.Environment, data *environmentResourceModel) diag.Diagnostics {
	if environment == nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("environment is nil for convertion", "This should not happend, please contact the plugin maintainer.")}
	}
	data.Canonical = types.StringPointerValue(environment.Canonical)
	data.Name = types.StringValue(environment.Name)
	data.Organization = types.StringValue(org)
	data.Color = types.StringPointerValue(environment.Color)
	data.Project = types.StringValue(project)
	return nil
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

	environments, err := p.provider.Middleware.ListProjectsEnv(org, project)
	if err != nil {
		diags.AddError("failed to fetch environments from API while updating resource", err.Error())
		return data, diags
	}

	var environment *models.Environment
	if i := slices.IndexFunc(environments, func(p *models.Environment) bool {
		return ptr.Value(p.Canonical) == canonical
	}); i == -1 {
		if isUpdate {
			tflog.Info(ctx, "did not found current environment, assuming it had been deleted outside the provider, re-creating...", nil)
		}

		environment, err = p.provider.Middleware.CreateEnv(org, project, canonical, name, color)
		if err != nil {
			diags.AddError("failed to create environment from API", err.Error())
			return data, diags
		}
	} else {
		environment, err = p.provider.Middleware.UpdateEnv(org, project, canonical, name, color)
		if err != nil {
			diags.AddError("failed to update environment from API", err.Error())
			return data, diags
		}
	}

	diags.Append(environmentToValue(ctx, org, project, environment, &data)...)
	return data, diags
}
