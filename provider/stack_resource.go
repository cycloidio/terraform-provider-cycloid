package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_stack"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*stackResource)(nil)

type stackResource struct {
	provider *CycloidProvider
}

type stackResourceModel resource_stack.StackModel

func NewStackResource() resource.Resource {
	return &stackResource{}
}

func (s *stackResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_stack.StackResourceSchema(ctx)
}
func (s *stackResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stack"
}

func (s *stackResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (s *stackResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data stackResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mid := s.provider.Middleware

	orgCan := getOrganizationCanonical(*s.provider, data.OrganizationCanonical)
	stack, err := mid.GetStack(orgCan, fmt.Sprintf("%s:%s", orgCan, data.Canonical.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Cannot edit stack with canonical '%s', stack must exist to be edited.", data.Canonical.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(s.UpdateStack(orgCan, stack, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (s *stackResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data stackResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Create API call logic
	mid := s.provider.Middleware

	orgCan := getOrganizationCanonical(*s.provider, data.OrganizationCanonical)
	stackRef := fmt.Sprintf("%s:%s", orgCan, data.Canonical.ValueString())
	stack, err := mid.GetStack(orgCan, stackRef)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to get stack informations with ref '%s'.", stackRef), err.Error())
		return
	}

	if stack.Team == nil {
		data.Team = types.StringNull()
	} else {
		data.Team = types.StringValue(*stack.Team.Canonical)
	}

	data.Canonical = types.StringValue(*stack.Canonical)
	data.Visibility = types.StringValue(*stack.Visibility)
}

func (s *stackResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data stackResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mid := s.provider.Middleware

	orgCan := getOrganizationCanonical(*s.provider, data.OrganizationCanonical)
	stack, err := mid.GetStack(orgCan, fmt.Sprintf("%s:%s", orgCan, data.Canonical.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Cannot edit stack with canonical '%s', stack must exist to be edited.", data.Canonical.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(s.UpdateStack(orgCan, stack, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (s *stackResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data stackResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

type stackCyModel struct {
	Visibility string `json:"visibility" tfsdk:"visibility"`
	Team       string `json:"team_canonical" tfsdk:"team"`
}

// UpdateStack will update the stack and merge the state in `data`
func (s *stackResource) UpdateStack(org string, stack *models.ServiceCatalog, data *stackResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	var visibility, team string
	if data.Visibility.IsNull() {
		visibility = *stack.Visibility
	} else {
		visibility = data.Visibility.ValueString()
	}

	if data.Team.IsNull() {
		if stack.Team != nil {
			team = *stack.Team.Canonical
		} else {
			team = ""
		}
	}

	// call api
	updatedStack, err := s.provider.Middleware.UpdateStack(org, ptr.Value(stack.Ref), team, &visibility)
	if err != nil {
		diags.AddError(fmt.Sprintf("Failed to update stack %s, API call failed", ptr.Value(stack.Ref)), err.Error())
		return diags
	}

	if data.Team.IsNull() && ptr.Value(ptr.Value(updatedStack.Team).Canonical) == "" {
		data.Team = types.StringNull()
	} else {
		data.Team = types.StringValue(*updatedStack.Team.Canonical)
	}

	data.Canonical = types.StringValue(*stack.Canonical)
	data.Visibility = types.StringValue(ptr.Value(updatedStack.Visibility))

	return diags
}
