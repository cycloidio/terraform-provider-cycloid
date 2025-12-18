package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
	"github.com/cycloidio/terraform-provider-cycloid/resource_stack"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/sanity-io/litter"
)

var _ resource.Resource = (*stackResource)(nil)

type stackResource struct {
	provider provider_cycloid.CycloidModel
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

	pv, ok := req.ProviderData.(provider_cycloid.CycloidModel)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
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

	api, err := getDefaultApi(s.provider)
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")
		return
	}
	mid := middleware.NewMiddleware(api)

	orgCan := getOrganizationCanonical(s.provider, data.OrganizationCanonical)
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
	api, err := getDefaultApi(s.provider)
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")
		return
	}
	mid := middleware.NewMiddleware(api)

	orgCan := getOrganizationCanonical(s.provider, data.OrganizationCanonical)
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
	// Create API call logic
	api, err := getDefaultApi(s.provider)
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")
		return
	}
	mid := middleware.NewMiddleware(api)

	orgCan := getOrganizationCanonical(s.provider, data.OrganizationCanonical)
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
	url := fmt.Sprintf("%s/organizations/%s/service_catalogs/%s", normalizeURL(s.provider.Url.ValueString()), org, *stack.Ref)
	payload, err := json.Marshal(stackCyModel{
		Visibility: visibility,
		Team:       team,
	})
	if err != nil {
		diags.AddError("failed to send update to API", "json serialization failed")
		return diags
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payload))
	if err != nil {
		diags.AddError("failed to send update to API", "request creation failed")
		return diags
	}

	req.Header.Set("Content-Type", "application/vnd.cycloid.io.v1+json")
	req.Header.Set("Authorization", "Bearer "+s.provider.Jwt.ValueString())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		diags.AddError("failed to update the stack settings with canonical "+org, err.Error())
		return diags
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		diags.AddError("API responded with "+resp.Status, litter.Sdump(resp))
		return diags
	}

	var respData struct {
		Data stackCyModel `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		diags.AddError("failed to deserialize api response", err.Error()+litter.Sdump(respData))
		return diags
	}

	updatedStack := respData.Data
	if data.Team.IsNull() && updatedStack.Team == "" {
		data.Team = types.StringNull()
	} else {
		data.Team = types.StringValue(updatedStack.Team)
	}

	data.Canonical = types.StringValue(*stack.Canonical)
	data.Visibility = types.StringValue(updatedStack.Visibility)

	return diags
}
