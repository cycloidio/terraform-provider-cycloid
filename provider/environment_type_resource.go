package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cycloidio/cycloid-cli/gen/models"
	"github.com/cycloidio/terraform-provider-cycloid/resource_environment_type"
)

var (
	_ resource.Resource                = (*environmentTypeResource)(nil)
	_ resource.ResourceWithImportState = (*environmentTypeResource)(nil)
)

func NewEnvironmentTypeResource() resource.Resource {
	return &environmentTypeResource{}
}

type environmentTypeResource struct {
	provider *CycloidProvider
}

type environmentTypeResourceModel resource_environment_type.EnvironmentTypeModel

func (r *environmentTypeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_type"
}

func (r *environmentTypeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_environment_type.EnvironmentTypeResourceSchema(ctx)
}

func (r *environmentTypeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *environmentTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data environmentTypeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	name := data.Name.ValueString()
	canonical := data.Canonical.ValueString()
	color := data.Color.ValueString()

	var err error
	name, canonical, err = NameOrCanonical(name, canonical)
	if err != nil {
		resp.Diagnostics.AddError("failed to infer environment type canonical", err.Error())
		return
	}

	body := &models.NewEnvironmentType{
		Name:      &name,
		Canonical: canonical,
		Color:     &color,
	}

	et, _, err := r.provider.Middleware.CreateEnvironmentType(org, body)
	if err != nil {
		resp.Diagnostics.AddError("failed to create environment type", err.Error())
		return
	}

	environmentTypeCYModelToData(org, et, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *environmentTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data environmentTypeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	canonical := data.Canonical.ValueString()

	et, _, err := r.provider.Middleware.GetEnvironmentType(org, canonical)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("failed to read environment type", err.Error())
		return
	}

	environmentTypeCYModelToData(org, et, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *environmentTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data environmentTypeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	canonical := data.Canonical.ValueString()
	name := data.Name.ValueString()
	color := data.Color.ValueString()

	body := &models.UpdateEnvironmentType{
		Name:  &name,
		Color: &color,
	}

	et, _, err := r.provider.Middleware.UpdateEnvironmentType(org, canonical, body)
	if err != nil {
		resp.Diagnostics.AddError("failed to update environment type", err.Error())
		return
	}

	environmentTypeCYModelToData(org, et, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *environmentTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data environmentTypeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	canonical := data.Canonical.ValueString()

	_, err := r.provider.Middleware.DeleteEnvironmentType(org, canonical)
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError("failed to delete environment type", err.Error())
	}
}

func (r *environmentTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("canonical"), req, resp)
}

func environmentTypeCYModelToData(org string, et *models.EnvironmentType, data *environmentTypeResourceModel) {
	data.Organization = types.StringValue(org)
	data.Canonical = types.StringPointerValue(et.Canonical)
	data.Name = types.StringPointerValue(et.Name)
	data.Color = types.StringPointerValue(et.Color)
	data.IsDefault = types.BoolPointerValue(et.IsDefault)
	data.EnvironmentsCount = ptrUint32ToInt64(et.EnvironmentsCount)
	data.ID = ptrUint32ToInt64(et.ID)
}
