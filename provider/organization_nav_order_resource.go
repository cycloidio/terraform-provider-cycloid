package provider

import (
	"context"
	"fmt"

	cycloidmiddleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/resource_organization_nav_order"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*organizationNavOrderResource)(nil)
var _ resource.ResourceWithImportState = (*organizationNavOrderResource)(nil)

func NewOrganizationNavOrderResource() resource.Resource {
	return &organizationNavOrderResource{}
}

type organizationNavOrderResource struct {
	provider *CycloidProvider
}

type organizationNavOrderResourceModel resource_organization_nav_order.OrganizationNavOrderModel

func (r *organizationNavOrderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_nav_order"
}

func (r *organizationNavOrderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_organization_nav_order.OrganizationNavOrderResourceSchema(ctx)
}

func (r *organizationNavOrderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *organizationNavOrderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationNavOrderResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	items, diags := navItemsFromData(ctx, data.Items)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, _, err := r.provider.Middleware.UpdateOrgNav(org, items)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to create nav ordering in org %q", org), err.Error())
		return
	}

	resp.Diagnostics.Append(navConfigToData(ctx, org, config, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationNavOrderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationNavOrderResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)

	config, _, err := r.provider.Middleware.GetOrgNav(org)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("failed to read nav ordering in org %q", org), err.Error())
		return
	}

	resp.Diagnostics.Append(navConfigToData(ctx, org, config, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationNavOrderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data organizationNavOrderResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	items, diags := navItemsFromData(ctx, data.Items)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, _, err := r.provider.Middleware.UpdateOrgNav(org, items)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to update nav ordering in org %q", org), err.Error())
		return
	}

	resp.Diagnostics.Append(navConfigToData(ctx, org, config, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationNavOrderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationNavOrderResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)

	// The API has no delete endpoint for this config. Reset to the default
	// (empty) ordering so relinquishing Terraform management doesn't leave a
	// custom ordering silently active.
	_, _, err := r.provider.Middleware.UpdateOrgNav(org, nil)
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddWarning(
			"Unable to reset nav ordering",
			"The resource was removed from Terraform state, but the server-side ordering could not be reset to defaults. Error: "+err.Error(),
		)
	}
}

// ImportState supports: terraform import cycloid_organization_nav_order.x <organization>
func (r *organizationNavOrderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data organizationNavOrderResourceModel
	data.Organization = types.StringValue(req.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// navItemsFromData converts the Terraform items list into the middleware's
// NavItem slice. Always returns a non-nil slice (possibly empty) so the
// caller sends an explicit [] rather than a JSON null — an empty array is
// what resets the ordering to defaults; omitting the field or sending null
// is not the same thing.
func navItemsFromData(ctx context.Context, itemsVal types.List) ([]*cycloidmiddleware.NavItem, diag.Diagnostics) {
	var diags diag.Diagnostics
	var itemModels []resource_organization_nav_order.NavItemModel

	if itemsVal.IsNull() || itemsVal.IsUnknown() {
		return []*cycloidmiddleware.NavItem{}, diags
	}

	diags.Append(itemsVal.ElementsAs(ctx, &itemModels, false)...)
	if diags.HasError() {
		return nil, diags
	}

	items := make([]*cycloidmiddleware.NavItem, len(itemModels))
	for i, im := range itemModels {
		items[i] = &cycloidmiddleware.NavItem{
			Type:     im.Type.ValueString(),
			Key:      im.Key.ValueString(),
			Position: uint32(im.Position.ValueInt64()),
		}
	}
	return items, diags
}

// navConfigToData maps the middleware's NavConfig response into the
// Terraform state model.
func navConfigToData(ctx context.Context, org string, config *cycloidmiddleware.NavConfig, data *organizationNavOrderResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.Organization = types.StringValue(org)

	itemModels := make([]resource_organization_nav_order.NavItemModel, len(config.Items))
	for i, it := range config.Items {
		itemModels[i] = resource_organization_nav_order.NavItemModel{
			Type:     types.StringValue(it.Type),
			Key:      types.StringValue(it.Key),
			Position: types.Int64Value(int64(it.Position)),
		}
	}

	itemsList, listDiags := types.ListValueFrom(ctx, itemObjectType(), itemModels)
	diags.Append(listDiags...)
	if diags.HasError() {
		return diags
	}
	data.Items = itemsList

	return diags
}

func itemObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"type":     types.StringType,
			"key":      types.StringType,
			"position": types.Int64Type,
		},
	}
}
