package provider

import (
	"context"
	"fmt"
	"net/http"

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

type oidcGroupMappingTeam struct {
	ID        uint32 `json:"id"`
	Canonical string `json:"canonical"`
}

type oidcGroupMapping struct {
	ID        uint32               `json:"id"`
	GroupName string               `json:"group_name"`
	Team      oidcGroupMappingTeam `json:"team"`
}

type newOIDCGroupMapping struct {
	GroupName     string `json:"group_name"`
	TeamCanonical string `json:"team_canonical"`
}

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

	mapping, err := r.createMapping(org, groupName, teamCanonical)
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

	mappings, err := r.listMappings(org)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("failed to list OIDC group mappings in org %q", org), err.Error())
		return
	}

	var found *oidcGroupMapping
	for i := range mappings {
		if mappings[i].ID == id {
			found = &mappings[i]
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

	err := r.deleteMapping(org, id)
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("failed to delete OIDC group mapping %d in org %q", id, org), err.Error())
	}
}

func oidcGroupMappingToData(org string, mapping *oidcGroupMapping, data *oidcGroupMappingResourceModel) {
	data.Organization = types.StringValue(org)
	data.GroupName = types.StringValue(mapping.GroupName)
	data.TeamCanonical = types.StringValue(mapping.Team.Canonical)
	data.ID = types.Int64Value(int64(mapping.ID))
}

func (r *oidcGroupMappingResource) createMapping(org, groupName, teamCanonical string) (*oidcGroupMapping, error) {
	body := &newOIDCGroupMapping{
		GroupName:     groupName,
		TeamCanonical: teamCanonical,
	}

	result := &oidcGroupMapping{}
	_, err := r.provider.Middleware.GenericRequest(cycloidmiddleware.Request{
		Method:       http.MethodPost,
		Organization: &org,
		Route:        []string{"organizations", org, "oidc-group-mappings"},
		Body:         body,
	}, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *oidcGroupMappingResource) listMappings(org string) ([]oidcGroupMapping, error) {
	var result []oidcGroupMapping
	_, err := r.provider.Middleware.GenericRequest(cycloidmiddleware.Request{
		Method:       http.MethodGet,
		Organization: &org,
		Route:        []string{"organizations", org, "oidc-group-mappings"},
	}, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *oidcGroupMappingResource) deleteMapping(org string, id uint32) error {
	_, err := r.provider.Middleware.GenericRequest(cycloidmiddleware.Request{
		Method:       http.MethodDelete,
		Organization: &org,
		Route:        []string{"organizations", org, "oidc-group-mappings", fmt.Sprintf("%d", id)},
	}, nil)
	return err
}
