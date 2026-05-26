package provider

import (
	"context"
	"fmt"
	"strings"

	middleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/resource_environment_link"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*environmentLinkResource)(nil)
var _ resource.ResourceWithImportState = (*environmentLinkResource)(nil)

func NewEnvironmentLinkResource() resource.Resource {
	return &environmentLinkResource{}
}

type environmentLinkResource struct {
	provider *CycloidProvider
}

type environmentLinkResourceModel resource_environment_link.EnvironmentLinkModel

func (r *environmentLinkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_link"
}

func (r *environmentLinkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_environment_link.EnvironmentLinkResourceSchema(ctx)
}

func (r *environmentLinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *environmentLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data environmentLinkResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	project := data.Project.ValueString()
	env := data.Environment.ValueString()

	_, err := r.provider.Middleware.LinkEnvToProject(org, project, env)
	if err != nil {
		resp.Diagnostics.AddError("failed to link environment to project", err.Error())
		return
	}

	data.Organization = types.StringValue(org)
	data.ID = types.StringValue(org + "/" + project + "/" + env)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *environmentLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data environmentLinkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	project := data.Project.ValueString()
	env := data.Environment.ValueString()

	envs, _, err := r.provider.Middleware.ListProjectEnvs(org, project)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("failed to list project environments", err.Error())
		return
	}

	for _, e := range envs {
		if e.Canonical != nil && *e.Canonical == env {
			data.Organization = types.StringValue(org)
			data.ID = types.StringValue(org + "/" + project + "/" + env)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *environmentLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// all attributes force replacement; Update is never called
}

func (r *environmentLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data environmentLinkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	project := data.Project.ValueString()
	env := data.Environment.ValueString()

	_, err := r.provider.Middleware.UnlinkEnvFromProject(org, project, env, middleware.DeleteOptions{})
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError("failed to unlink environment from project", err.Error())
	}
}

func (r *environmentLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"invalid import ID format",
			"expected format org/project/environment, got: "+req.ID,
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
