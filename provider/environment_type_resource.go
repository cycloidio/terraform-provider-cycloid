package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/cycloidio/cycloid-cli/gen/models"
	"github.com/cycloidio/terraform-provider-cycloid/resource_environment_type"
	"github.com/cycloidio/cycloid-cli/utils/ptr"
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

	et, _, err := r.provider.Client.CreateEnvironmentType(org, body)
	if err != nil {
		resp.Diagnostics.AddError("failed to create environment type", err.Error())
		return
	}

	// Set label selector if configured
	if !data.LabelSelector.IsNull() && !data.LabelSelector.IsUnknown() {
		labelSelector, diags := labelSelectorFromState(ctx, data.LabelSelector)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		et, _, err = r.provider.Client.SetEnvironmentTypeLabelSelector(org, canonical, labelSelector)
		if err != nil {
			resp.Diagnostics.AddError("failed to set label selector", err.Error())
			return
		}
	}

	environmentTypeCYModelToData(ctx, org, et, &data, &resp.Diagnostics)
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

	et, _, err := r.provider.Client.GetEnvironmentType(org, canonical)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("failed to read environment type", err.Error())
		return
	}

	environmentTypeCYModelToData(ctx, org, et, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *environmentTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data environmentTypeResourceModel
	var state environmentTypeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := getOrganizationCanonical(*r.provider, state.Organization)
	canonical := state.Canonical.ValueString()
	name := data.Name.ValueString()
	if name == "" {
		name = state.Name.ValueString()
	}
	color := data.Color.ValueString()

	body := &models.UpdateEnvironmentType{
		Name:  &name,
		Color: &color,
	}

	et, _, err := r.provider.Client.UpdateEnvironmentType(org, canonical, body)
	if err != nil {
		resp.Diagnostics.AddError("failed to update environment type", err.Error())
		return
	}

	// Handle label selector changes
	planHasSelector := !data.LabelSelector.IsNull() && !data.LabelSelector.IsUnknown()
	stateHasSelector := !state.LabelSelector.IsNull() && !state.LabelSelector.IsUnknown()

	if planHasSelector {
		// Set or update the label selector
		labelSelector, diags := labelSelectorFromState(ctx, data.LabelSelector)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		et, _, err = r.provider.Client.SetEnvironmentTypeLabelSelector(org, canonical, labelSelector)
		if err != nil {
			resp.Diagnostics.AddError("failed to set label selector", err.Error())
			return
		}
	} else if stateHasSelector && !planHasSelector {
		// Label selector was removed from config — delete it
		_, err = r.provider.Client.DeleteEnvironmentTypeLabelSelector(org, canonical)
		if err != nil {
			resp.Diagnostics.AddError("failed to delete label selector", err.Error())
			return
		}
		// Re-read to get updated state
		et, _, err = r.provider.Client.GetEnvironmentType(org, canonical)
		if err != nil {
			resp.Diagnostics.AddError("failed to read environment type after deleting label selector", err.Error())
			return
		}
	}

	environmentTypeCYModelToData(ctx, org, et, &data, &resp.Diagnostics)
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

	_, err := r.provider.Client.DeleteEnvironmentType(org, canonical)
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError("failed to delete environment type", err.Error())
	}
}

func (r *environmentTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("canonical"), req, resp)
}

func environmentTypeCYModelToData(ctx context.Context, org string, et *models.EnvironmentType, data *environmentTypeResourceModel, diags *diag.Diagnostics) {
	data.Organization = types.StringValue(org)
	data.Canonical = types.StringPointerValue(et.Canonical)
	data.Name = types.StringPointerValue(et.Name)
	data.Color = types.StringPointerValue(et.Color)
	data.IsDefault = types.BoolPointerValue(et.IsDefault)
	data.EnvironmentsCount = ptrUint32ToInt64(et.EnvironmentsCount)
	data.ID = ptrUint32ToInt64(et.ID)

	if et.LabelSelector != nil {
		lsObj, d := labelSelectorToState(ctx, et.LabelSelector)
		diags.Append(d...)
		data.LabelSelector = lsObj
	} else {
		data.LabelSelector = types.ObjectNull(labelSelectorAttrTypes())
	}
}

// labelSelectorAttrTypes returns the attribute types for the label_selector object.
func labelSelectorAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enforcement": types.StringType,
		"requirements": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
			"key":      types.StringType,
			"operator": types.StringType,
			"values":   types.ListType{ElemType: types.StringType},
		}}},
	}
}

// labelSelectorRequirementAttrTypes returns the attribute types for a single requirement.
func labelSelectorRequirementAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"key":      types.StringType,
		"operator": types.StringType,
		"values":   types.ListType{ElemType: types.StringType},
	}
}

// labelSelectorToState converts an API LabelSelector model to a TF state object.
func labelSelectorToState(ctx context.Context, ls *models.LabelSelector) (basetypes.ObjectValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	reqObjs := make([]attr.Value, 0, len(ls.Requirements))
	for _, req := range ls.Requirements {
		values := make([]attr.Value, 0, len(req.Values))
		for _, v := range req.Values {
			values = append(values, types.StringValue(v))
		}
		valList, d := types.ListValue(types.StringType, values)
		diags.Append(d...)

		reqObj, d := types.ObjectValue(labelSelectorRequirementAttrTypes(), map[string]attr.Value{
			"key":      types.StringPointerValue(req.Key),
			"operator": types.StringPointerValue(req.Operator),
			"values":   valList,
		})
		diags.Append(d...)
		reqObjs = append(reqObjs, reqObj)
	}

	reqList, d := types.ListValue(types.ObjectType{AttrTypes: labelSelectorRequirementAttrTypes()}, reqObjs)
	diags.Append(d...)

	enforcement := types.StringValue("soft")
	if ls.Enforcement != nil {
		enforcement = types.StringValue(*ls.Enforcement)
	}

	obj, d := types.ObjectValue(labelSelectorAttrTypes(), map[string]attr.Value{
		"enforcement":  enforcement,
		"requirements": reqList,
	})
	diags.Append(d...)

	return obj, diags
}

// labelSelectorFromState converts TF state object back to an API LabelSelector model.
func labelSelectorFromState(ctx context.Context, obj basetypes.ObjectValue) (*models.LabelSelector, diag.Diagnostics) {
	var diags diag.Diagnostics

	var lsModel resource_environment_type.LabelSelectorModel
	diags.Append(obj.As(ctx, &lsModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	var reqModels []resource_environment_type.LabelSelectorRequirementModel
	diags.Append(lsModel.Requirements.ElementsAs(ctx, &reqModels, false)...)
	if diags.HasError() {
		return nil, diags
	}

	requirements := make([]*models.LabelSelectorRequirement, 0, len(reqModels))
	for _, rm := range reqModels {
		var values []string
		diags.Append(rm.Values.ElementsAs(ctx, &values, false)...)
		if diags.HasError() {
			return nil, diags
		}

		key := rm.Key.ValueString()
		op := rm.Operator.ValueString()
		requirements = append(requirements, &models.LabelSelectorRequirement{
			Key:      &key,
			Operator: &op,
			Values:   values,
		})
	}

	enforcement := lsModel.Enforcement.ValueString()
	if enforcement == "" {
		enforcement = "soft"
	}

	return &models.LabelSelector{
		Enforcement:  ptr.Ptr(enforcement),
		Requirements: requirements,
	}, diags
}
