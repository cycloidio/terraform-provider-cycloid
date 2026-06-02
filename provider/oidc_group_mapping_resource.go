package provider

import (
	"context"
	"fmt"

	cycloidmiddleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/resource_oidc_group_mapping"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*oidcGroupMappingResource)(nil)

func NewOIDCGroupMappingResource() resource.Resource {
	return &oidcGroupMappingResource{}
}

type oidcGroupMappingResource struct {
	provider *CycloidProvider
}

type oidcGroupMappingResourceModel resource_oidc_group_mapping.OidcGroupMappingModel

func (r *oidcGroupMappingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_group_mapping"
}

func (r *oidcGroupMappingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_oidc_group_mapping.OidcGroupMappingResourceSchema(ctx)
}

func (r *oidcGroupMappingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *oidcGroupMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data oidcGroupMappingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	groupName := data.GroupName.ValueString()
	teamCanonical := data.TeamCanonical.ValueString()

	mapping, _, err := r.provider.Middleware.CreateOIDCGroupMapping(org, groupName, teamCanonical)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to create OIDC group mapping for group %q in org %q", groupName, org), err.Error())
		return
	}

	oidcGroupMappingToData(org, mapping, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *oidcGroupMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data oidcGroupMappingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	id := uint32(data.ID.ValueInt64())

	mappings, _, err := r.provider.Middleware.ListOIDCGroupMappings(org)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list OIDC group mappings in org %q", org), err.Error())
		return
	}

	var found *cycloidmiddleware.OIDCGroupMapping
	for _, m := range mappings {
		if m.ID == id {
			found = m
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	oidcGroupMappingToData(org, found, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *oidcGroupMappingResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All fields use RequiresReplace — Update is never called.
}

func (r *oidcGroupMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data oidcGroupMappingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.Organization)
	id := uint32(data.ID.ValueInt64())

	_, err := r.provider.Middleware.DeleteOIDCGroupMapping(org, id)
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("failed to delete OIDC group mapping %d in org %q", id, org), err.Error())
	}
}

func oidcGroupMappingToData(org string, mapping *cycloidmiddleware.OIDCGroupMapping, data *oidcGroupMappingResourceModel) {
	data.Organization = types.StringValue(org)
	data.GroupName = types.StringValue(mapping.GroupName)
	data.ID = types.Int64Value(int64(mapping.ID))
	// Team is a pointer in the API response; guard against a mapping whose team
	// was deleted server-side to avoid a provider panic.
	if mapping.Team != nil {
		data.TeamCanonical = types.StringValue(mapping.Team.Canonical)
	}
}
