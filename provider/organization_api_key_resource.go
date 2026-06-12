package provider

import (
	"context"
	"fmt"

	"github.com/cycloidio/cycloid-cli/client/models"
	cycloidmiddleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/resource_organization_api_key"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*organizationAPIKeyResource)(nil)
var _ resource.ResourceWithImportState = (*organizationAPIKeyResource)(nil)

func NewOrganizationAPIKeyResource() resource.Resource {
	return &organizationAPIKeyResource{}
}

type organizationAPIKeyResource struct {
	provider *CycloidProvider
}

type organizationAPIKeyResourceModel resource_organization_api_key.OrganizationAPIKeyModel

func (r *organizationAPIKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_api_key"
}

func (r *organizationAPIKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_organization_api_key.OrganizationAPIKeyResourceSchema(ctx)
}

func (r *organizationAPIKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *organizationAPIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationAPIKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.OrganizationCanonical)
	name := data.Name.ValueString()
	canonical := data.Canonical.ValueString()
	description := data.Description.ValueString()
	owner := data.Owner.ValueString()

	rules, diags := dataToNewRules(ctx, data.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey, _, err := r.provider.Middleware.CreateAPIKey(org, canonical, description, owner, &name, rules)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create organization API key", err.Error())
		return
	}

	resp.Diagnostics.Append(apiKeyCYModelToData(ctx, org, apiKey, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationAPIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationAPIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.OrganizationCanonical)
	canonical := data.Canonical.ValueString()

	apiKey, _, err := r.provider.Middleware.GetAPIKey(org, canonical)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read organization API key", err.Error())
		return
	}

	resp.Diagnostics.Append(apiKeyCYModelToData(ctx, org, apiKey, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationAPIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data organizationAPIKeyResourceModel
	var stateData organizationAPIKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.OrganizationCanonical)
	// Use state canonical in case plan canonical differs (canonical is immutable via UseStateForUnknown).
	canonical := stateData.Canonical.ValueString()
	name := data.Name.ValueString()
	description := data.Description.ValueString()
	owner := data.Owner.ValueString()

	body := &models.UpdateAPIKey{
		Name:        &name,
		Description: description,
		Owner:       owner,
	}

	var apiKey *models.APIKey
	_, err := r.provider.Middleware.GenericRequest(cycloidmiddleware.Request{
		Method:       "PUT",
		Organization: &org,
		Route:        []string{"organizations", org, "api_keys", canonical},
		Body:         body,
	}, &apiKey)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update organization API key", err.Error())
		return
	}

	resp.Diagnostics.Append(apiKeyCYModelToData(ctx, org, apiKey, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *organizationAPIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationAPIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, data.OrganizationCanonical)
	canonical := data.Canonical.ValueString()

	_, err := r.provider.Middleware.DeleteAPIKey(org, canonical)
	if err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError("Unable to delete organization API key", err.Error())
	}
}

func (r *organizationAPIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("canonical"), req, resp)
}

// apiKeyCYModelToData maps an APIKey model from the API into the Terraform state model.
// The token field is only populated on creation; subsequent reads leave it unchanged (UseStateForUnknown).
func apiKeyCYModelToData(ctx context.Context, org string, apiKey *models.APIKey, data *organizationAPIKeyResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.OrganizationCanonical = types.StringValue(org)
	data.Canonical = types.StringPointerValue(apiKey.Canonical)
	data.Name = types.StringPointerValue(apiKey.Name)
	data.Description = types.StringValue(apiKey.Description)
	data.LastSeven = types.StringPointerValue(apiKey.LastSeven)
	data.ID = types.Int64Value(int64(*apiKey.ID))

	if apiKey.Owner != nil && apiKey.Owner.Username != nil {
		data.Owner = types.StringPointerValue(apiKey.Owner.Username)
	} else if data.Owner.IsNull() || data.Owner.IsUnknown() {
		data.Owner = types.StringValue("")
	}

	// Token is only returned on creation. Preserve the state value on subsequent reads.
	if apiKey.Token != "" {
		data.Token = types.StringValue(apiKey.Token)
	}

	rules, rDiags := rulesToTFList(ctx, apiKey.Rules)
	diags.Append(rDiags...)
	if !diags.HasError() {
		data.Rules = rules
	}

	return diags
}

// dataToNewRules converts the Terraform rules list into the API NewRule slice.
func dataToNewRules(ctx context.Context, rulesVal types.List) ([]*models.NewRule, diag.Diagnostics) {
	var diags diag.Diagnostics
	var ruleModels []resource_organization_api_key.RuleModel

	diags.Append(rulesVal.ElementsAs(ctx, &ruleModels, false)...)
	if diags.HasError() {
		return nil, diags
	}

	rules := make([]*models.NewRule, len(ruleModels))
	for i, rm := range ruleModels {
		action := rm.Action.ValueString()
		effect := rm.Effect.ValueString()
		var resources []string
		if !rm.Resources.IsNull() && !rm.Resources.IsUnknown() {
			diags.Append(rm.Resources.ElementsAs(ctx, &resources, false)...)
			if diags.HasError() {
				return nil, diags
			}
		}
		rules[i] = &models.NewRule{
			Action:    &action,
			Effect:    &effect,
			Resources: resources,
		}
	}

	return rules, diags
}

// rulesToTFList converts the API Rule slice to a Terraform types.List.
func rulesToTFList(ctx context.Context, rules []*models.Rule) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	ruleModels := make([]resource_organization_api_key.RuleModel, len(rules))
	for i, r := range rules {
		action := ""
		if r.Action != nil {
			action = *r.Action
		}
		effect := ""
		if r.Effect != nil {
			effect = *r.Effect
		}

		var resourcesList types.List
		if len(r.Resources) > 0 {
			var lDiags diag.Diagnostics
			resourcesList, lDiags = types.ListValueFrom(ctx, types.StringType, r.Resources)
			diags.Append(lDiags...)
			if diags.HasError() {
				return types.ListNull(apiKeyRuleObjectType()), diags
			}
		} else {
			resourcesList = types.ListValueMust(types.StringType, []attr.Value{})
		}

		ruleModels[i] = resource_organization_api_key.RuleModel{
			Action:    types.StringValue(action),
			Effect:    types.StringValue(effect),
			Resources: resourcesList,
		}
	}

	result, lDiags := types.ListValueFrom(ctx, apiKeyRuleObjectType(), ruleModels)
	diags.Append(lDiags...)
	return result, diags
}

func apiKeyRuleObjectType() attr.Type {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"action":    types.StringType,
			"effect":    types.StringType,
			"resources": types.ListType{ElemType: types.StringType},
		},
	}
}
