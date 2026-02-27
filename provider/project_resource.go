package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/icons"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_project"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*projectResource)(nil)

func NewProjectResource() resource.Resource {
	return &projectResource{}
}

type projectResourceModel resource_project.ProjectModel

type projectResource struct {
	provider *CycloidProvider
}

func (p *projectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (p *projectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_project.ProjectResourceSchema(ctx)
}

func (p *projectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (p *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data projectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := p.provider.Middleware
	canonical := data.Canonical.ValueString()

	org := getOrganizationCanonical(*p.provider, data.Organization)
	projects, err := m.ListProjects(org)
	if err != nil {
		resp.Diagnostics.AddError("failed to fetch project from API", err.Error())
		return
	}

	if i := slices.IndexFunc(projects, func(p *models.Project) bool {
		return ptr.Value(p.Canonical) == canonical
	}); i == -1 {
		// Project doesn't exist, so empty state
		resp.Diagnostics.Append(projectToValue(ctx, org, &models.Project{}, &data)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		resp.Diagnostics.Append(projectToValue(ctx, org, projects[i], &data)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (p *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data projectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	canonical := data.Canonical.ValueString()
	org := getOrganizationCanonical(*p.provider, data.Organization)
	description := data.Description.ValueString()
	color := data.Color.ValueString()
	if color == "" {
		color = icons.RandomColor()
	}

	icon := data.Icon.ValueString()
	if icon == "" {
		icon = icons.RandomIcon()
	}
	owner := data.Owner.ValueString()
	configRepository := data.ConfigRepository.ValueString()

	data, d := p.createOrUpdateProject(ctx, org, name, canonical, description, configRepository, owner, owner, color, icon, false)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (p *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data projectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	canonical := data.Canonical.ValueString()
	org := getOrganizationCanonical(*p.provider, data.Organization)
	description := data.Description.ValueString()
	color := data.Color.ValueString()
	if color == "" {
		color = icons.RandomIcon()
	}

	icon := data.Icon.ValueString()
	if icon == "" {
		icon = icons.RandomIcon()
	}
	owner := data.Owner.ValueString()
	configRepository := data.ConfigRepository.ValueString()

	data, d := p.createOrUpdateProject(ctx, org, name, canonical, description, configRepository, owner, owner, color, icon, true)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (p *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data projectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := p.provider.Middleware
	org := getOrganizationCanonical(*p.provider, data.Organization)
	canonical := data.Canonical.ValueString()

	err := m.DeleteProject(org, canonical)
	if err != nil {
		resp.Diagnostics.AddError("failed to fetch project from API", err.Error())
		return
	}

	resp.Diagnostics.Append(projectToValue(ctx, org, &models.Project{}, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func projectToValue(ctx context.Context, org string, project *models.Project, data *projectResourceModel) diag.Diagnostics {
	if project == nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("project is nil for convertion", "This should not happend, please contact the plugin maintainer.")}
	}
	data.Canonical = types.StringPointerValue(project.Canonical)
	data.Name = types.StringPointerValue(project.Name)
	data.Organization = types.StringValue(org)
	data.Description = types.StringValue(project.Description)
	data.Color = types.StringPointerValue(project.Color)
	data.Icon = types.StringPointerValue(project.Icon)
	data.Owner = types.StringPointerValue(ptr.Value(project.Owner).Username)
	data.ConfigRepository = types.StringValue(project.ConfigRepositoryCanonical)
	return nil
}

func (p *projectResource) createOrUpdateProject(ctx context.Context, org, name, canonical, description, configRepository, owner, team, color, icon string, isUpdate bool) (projectResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var data projectResourceModel
	var err error

	name, canonical, err = NameOrCanonical(name, canonical)
	if err != nil {
		diags.AddError("failed to infer canonical", err.Error())
		return data, diags
	}

	if configRepository == "" {
		configRepositories, err := p.provider.Middleware.ListConfigRepositories(org)
		if err != nil {
			diags.AddError("failed to fetch list of current config repositories to infer default catalog", err.Error())
			return data, diags
		}

		if i := slices.IndexFunc(configRepositories, func(c *models.ConfigRepository) bool {
			return ptr.Value(c.Default)
		}); i != -1 {
			configRepository = ptr.Value(configRepositories[i].Canonical)
		} else {
			diags.AddError("no default catalog repository was found", "please add a default config repository to the org "+org+" or fill the `catalog_respository` attribute. Docs: https://docs.cycloid.io/reference/config-and-catalog-repository/")
			return data, diags
		}
	}

	projects, err := p.provider.Middleware.ListProjects(org)
	if err != nil {
		diags.AddError("failed to fetch projects from API", err.Error())
		return data, diags
	}

	var project *models.Project
	if i := slices.IndexFunc(projects, func(p *models.Project) bool {
		return ptr.Value(p.Canonical) == canonical
	}); i == -1 {
		if isUpdate {
			tflog.Info(ctx, "did not found current project, assuming it had been deleted outside the provider, re-creating...", nil)
		}

		project, err = p.provider.Middleware.CreateProject(org, name, canonical, description, configRepository, owner, owner, color, icon)
		if err != nil {
			diags.AddError("failed to create project from API", err.Error())
			return data, diags
		}
	} else {
		project, err = p.provider.Middleware.UpdateProject(org, name, canonical, description, configRepository, owner, owner, color, icon, "")
		if err != nil {
			diags.AddError("failed to update project from API", err.Error())
			return data, diags
		}
	}

	diags.Append(projectToValue(ctx, org, project, &data)...)
	return data, diags
}
