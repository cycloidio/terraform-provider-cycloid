// This schema has been hand coded
package datasource_terraform_output

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TerraformOutputDataSourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "This datasource allows you to fetch a single terraform output from the Cycloid inventory. Look up the documentation on output: https://docs.cycloid.io/reference/projects/concepts/components#output-from-terraform-output",
		MarkdownDescription: "This datasource allows you to fetch a single terraform output from the Cycloid inventory. Look up the [documentation on output](https://docs.cycloid.io/reference/projects/concepts/components#output-from-terraform-output).",
		Attributes: map[string]schema.Attribute{
			"filters": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"attribute": schema.StringAttribute{
							Required:            true,
							Description:         `The name of the attribute, you can filter using "output_key", "output_attribute", "project_canonical", "environment_canonical", "component_canonical" or "service_catalog_canonical"`,
							MarkdownDescription: `The name of the attribute, you can filter using "output_key", "output_attribute", "project_canonical", "environment_canonical", "component_canonical" or "service_catalog_canonical"`,
							Validators: []validator.String{
								stringvalidator.OneOf(
									"output_key", "output_attribute",
									"project_canonical", "environment_canonical",
									"component_canonical", "service_catalog_canonical",
								),
							},
						},
						"condition": schema.StringAttribute{
							Required:            true,
							Description:         `The condition to apply, one of "eq", "neq", "gt", "lt", "rlike" or "in".`,
							MarkdownDescription: `The condition to apply, one of "eq", "neq", "gt", "lt", "rlike" or "in".`,
							Validators: []validator.String{
								stringvalidator.OneOf("eq", "neq", "gt", "lt", "rlike", "in"),
							},
						},
						"value": schema.StringAttribute{
							Description:         "The value of the filter",
							MarkdownDescription: "The value of the filter",
							Required:            true,
						},
					},
					CustomType: FiltersType{
						ObjectType: types.ObjectType{
							AttrTypes: FiltersValue{}.AttributeTypes(ctx),
						},
					},
				},
				Optional:            true,
				Description:         "List of LHS filters to apply to the output. See the docs here: https://docs.cycloid.io/reference/api/LHS-filters",
				MarkdownDescription: "List of LHS filters to apply to the output. See the docs [here](https://docs.cycloid.io/reference/api/LHS-filters)",
			},
			"select_first": schema.BoolAttribute{
				Optional:            true,
				Description:         "If more than one output is listed, the data will fail, set this attribute to `true` to select the first result.",
				MarkdownDescription: "If more than one output is listed, the data will fail, set this attribute to `true` to select the first result.",
			},
			"organization": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The canonical of the organization where to fetch output, default to the provider's organization",
				MarkdownDescription: "The canonical of the organization where to fetch output, default to the provider's organization",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
				},
			},
			"output": schema.SingleNestedAttribute{
				Computed:            true,
				Description:         `A single terraform output, the output schema will match the result of a single listInventoryOutput data item from the API: https://docs.cycloid.io/api/#tag/Organization-Inventory/operation/listInventoryOutput. The 'value' and 'type' attributes in the terraform output will be dynamic depending on the terraform output itself.`,
				MarkdownDescription: `A single terraform output, the output schema will match the result of a single [listInventoryOutput data item from the API](https://docs.cycloid.io/api/#tag/Organization-Inventory/operation/listInventoryOutput). The 'value' and 'type' attributes in the terraform output will be dynamic depending on the terraform output itself.`,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description:         "the id of the output",
						MarkdownDescription: "the id of the output",
						Computed:            true,
					},
					"key": schema.StringAttribute{
						Description:         "the key of the output",
						MarkdownDescription: "the key of the output",
						Computed:            true,
					},
					"value": schema.DynamicAttribute{
						Description:         "the effective value of the output, can be anything a terraform output acepts",
						MarkdownDescription: "the effective value of the output, can be anything a terraform output acepts",
						Sensitive:           true,
						Computed:            true,
					},
					"type": schema.DynamicAttribute{
						Description:         "a description of the type",
						MarkdownDescription: "a description of the type",
						Computed:            true,
					},
					"sensitive": schema.BoolAttribute{
						Description:         "is the current attribute sensitive",
						MarkdownDescription: "is the current attribute sensitive",
						Computed:            true,
					},
					"description": schema.StringAttribute{
						Description:         "the output description",
						MarkdownDescription: "the output description",
						Computed:            true,
					},
				},
			},
		},
	}
}

type Filter struct {
	Attribute string `tfsdk:"attribute"`
	Condition string `tfsdk:"condition"`
	Value     string `tfsdk:"value"`
}

type TerraformOutput struct {
	ID          uint64  `json:"id"`
	Key         string  `json:"key"`
	Value       any     `json:"value"`
	Type        any     `json:"type"`
	Sensitive   bool    `json:"sensitive"`
	Description *string `json:"description,omitempty"`
}

type TerraformOutputModel struct {
	Filters      types.List   `tfsdk:"filters"`
	Organization types.String `tfsdk:"organization"`
	Output       types.Object `tfsdk:"output"`
	SelectFirst  types.Bool   `tfsdk:"select_first"`
}

type OutputType struct {
	basetypes.ObjectType
}

var _ basetypes.ObjectTypable = OutputType{}

var _ basetypes.ObjectTypable = FiltersType{}

type FiltersType struct {
	basetypes.ObjectType
}

func (t FiltersType) Equal(o attr.Type) bool {
	other, ok := o.(FiltersType)

	if !ok {
		return false
	}

	return t.ObjectType.Equal(other.ObjectType)
}

func (t FiltersType) String() string {
	return "FiltersType"
}

func (t FiltersType) ValueFromObject(ctx context.Context, in basetypes.ObjectValue) (basetypes.ObjectValuable, diag.Diagnostics) {
	var diags diag.Diagnostics

	attributes := in.Attributes()

	attributeAttribute, ok := attributes["attribute"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`attribute is missing from object`)

		return nil, diags
	}

	attributeVal, ok := attributeAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`attribute expected to be basetypes.StringValue, was: %T`, attributeAttribute))
	}

	conditionAttribute, ok := attributes["condition"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`condition is missing from object`)

		return nil, diags
	}

	conditionVal, ok := conditionAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`condition expected to be basetypes.StringValue, was: %T`, conditionAttribute))
	}

	valueAttribute, ok := attributes["value"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`value is missing from object`)

		return nil, diags
	}

	valueVal, ok := valueAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`value expected to be basetypes.StringValue, was: %T`, valueAttribute))
	}

	if diags.HasError() {
		return nil, diags
	}

	return FiltersValue{
		Attribute: attributeVal,
		Condition: conditionVal,
		Value:     valueVal,
		state:     attr.ValueStateKnown,
	}, diags
}

func NewFiltersValueNull() FiltersValue {
	return FiltersValue{
		state: attr.ValueStateNull,
	}
}

func NewFiltersValueUnknown() FiltersValue {
	return FiltersValue{
		state: attr.ValueStateUnknown,
	}
}

func NewFiltersValue(attributeTypes map[string]attr.Type, attributes map[string]attr.Value) (FiltersValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Reference: https://github.com/hashicorp/terraform-plugin-framework/issues/521
	ctx := context.Background()

	for name, attributeType := range attributeTypes {
		attribute, ok := attributes[name]

		if !ok {
			diags.AddError(
				"Missing FiltersValue Attribute Value",
				"While creating a FiltersValue value, a missing attribute value was detected. "+
					"A FiltersValue must contain values for all attributes, even if null or unknown. "+
					"This is always an issue with the provider and should be reported to the provider developers.\n\n"+
					fmt.Sprintf("FiltersValue Attribute Name (%s) Expected Type: %s", name, attributeType.String()),
			)

			continue
		}

		if !attributeType.Equal(attribute.Type(ctx)) {
			diags.AddError(
				"Invalid FiltersValue Attribute Type",
				"While creating a FiltersValue value, an invalid attribute value was detected. "+
					"A FiltersValue must use a matching attribute type for the value. "+
					"This is always an issue with the provider and should be reported to the provider developers.\n\n"+
					fmt.Sprintf("FiltersValue Attribute Name (%s) Expected Type: %s\n", name, attributeType.String())+
					fmt.Sprintf("FiltersValue Attribute Name (%s) Given Type: %s", name, attribute.Type(ctx)),
			)
		}
	}

	for name := range attributes {
		_, ok := attributeTypes[name]

		if !ok {
			diags.AddError(
				"Extra FiltersValue Attribute Value",
				"While creating a FiltersValue value, an extra attribute value was detected. "+
					"A FiltersValue must not contain values beyond the expected attribute types. "+
					"This is always an issue with the provider and should be reported to the provider developers.\n\n"+
					fmt.Sprintf("Extra FiltersValue Attribute Name: %s", name),
			)
		}
	}

	if diags.HasError() {
		return NewFiltersValueUnknown(), diags
	}

	attributeAttribute, ok := attributes["attribute"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`attribute is missing from object`)

		return NewFiltersValueUnknown(), diags
	}

	attributeVal, ok := attributeAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`attribute expected to be basetypes.StringValue, was: %T`, attributeAttribute))
	}

	conditionAttribute, ok := attributes["condition"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`condition is missing from object`)

		return NewFiltersValueUnknown(), diags
	}

	conditionVal, ok := conditionAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`condition expected to be basetypes.StringValue, was: %T`, conditionAttribute))
	}

	valueAttribute, ok := attributes["value"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`value is missing from object`)

		return NewFiltersValueUnknown(), diags
	}

	valueVal, ok := valueAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`value expected to be basetypes.StringValue, was: %T`, valueAttribute))
	}

	if diags.HasError() {
		return NewFiltersValueUnknown(), diags
	}

	return FiltersValue{
		Attribute: attributeVal,
		Condition: conditionVal,
		Value:     valueVal,
		state:     attr.ValueStateKnown,
	}, diags
}

func NewFiltersValueMust(attributeTypes map[string]attr.Type, attributes map[string]attr.Value) FiltersValue {
	object, diags := NewFiltersValue(attributeTypes, attributes)

	if diags.HasError() {
		// This could potentially be added to the diag package.
		diagsStrings := make([]string, 0, len(diags))

		for _, diagnostic := range diags {
			diagsStrings = append(diagsStrings, fmt.Sprintf(
				"%s | %s | %s",
				diagnostic.Severity(),
				diagnostic.Summary(),
				diagnostic.Detail()))
		}

		panic("NewFiltersValueMust received error(s): " + strings.Join(diagsStrings, "\n"))
	}

	return object
}

func (t FiltersType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	if in.Type() == nil {
		return NewFiltersValueNull(), nil
	}

	if !in.Type().Equal(t.TerraformType(ctx)) {
		return nil, fmt.Errorf("expected %s, got %s", t.TerraformType(ctx), in.Type())
	}

	if !in.IsKnown() {
		return NewFiltersValueUnknown(), nil
	}

	if in.IsNull() {
		return NewFiltersValueNull(), nil
	}

	attributes := map[string]attr.Value{}

	val := map[string]tftypes.Value{}

	err := in.As(&val)

	if err != nil {
		return nil, err
	}

	for k, v := range val {
		a, err := t.AttrTypes[k].ValueFromTerraform(ctx, v)

		if err != nil {
			return nil, err
		}

		attributes[k] = a
	}

	return NewFiltersValueMust(FiltersValue{}.AttributeTypes(ctx), attributes), nil
}

func (t FiltersType) ValueType(ctx context.Context) attr.Value {
	return FiltersValue{}
}

var _ basetypes.ObjectValuable = FiltersValue{}

type FiltersValue struct {
	Attribute basetypes.StringValue `tfsdk:"attribute"`
	Condition basetypes.StringValue `tfsdk:"condition"`
	Value     basetypes.StringValue `tfsdk:"value"`
	state     attr.ValueState
}

func (v FiltersValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	attrTypes := make(map[string]tftypes.Type, 3)

	var val tftypes.Value
	var err error

	attrTypes["attribute"] = basetypes.StringType{}.TerraformType(ctx)
	attrTypes["condition"] = basetypes.StringType{}.TerraformType(ctx)
	attrTypes["value"] = basetypes.StringType{}.TerraformType(ctx)

	objectType := tftypes.Object{AttributeTypes: attrTypes}

	switch v.state {
	case attr.ValueStateKnown:
		vals := make(map[string]tftypes.Value, 3)

		val, err = v.Attribute.ToTerraformValue(ctx)

		if err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		vals["attribute"] = val

		val, err = v.Condition.ToTerraformValue(ctx)

		if err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		vals["condition"] = val

		val, err = v.Value.ToTerraformValue(ctx)

		if err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		vals["value"] = val

		if err := tftypes.ValidateValue(objectType, vals); err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		return tftypes.NewValue(objectType, vals), nil
	case attr.ValueStateNull:
		return tftypes.NewValue(objectType, nil), nil
	case attr.ValueStateUnknown:
		return tftypes.NewValue(objectType, tftypes.UnknownValue), nil
	default:
		panic(fmt.Sprintf("unhandled Object state in ToTerraformValue: %s", v.state))
	}
}

func (v FiltersValue) IsNull() bool {
	return v.state == attr.ValueStateNull
}

func (v FiltersValue) IsUnknown() bool {
	return v.state == attr.ValueStateUnknown
}

func (v FiltersValue) String() string {
	return "FiltersValue"
}

func (v FiltersValue) ToObjectValue(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	attributeTypes := map[string]attr.Type{
		"attribute": basetypes.StringType{},
		"condition": basetypes.StringType{},
		"value":     basetypes.StringType{},
	}

	if v.IsNull() {
		return types.ObjectNull(attributeTypes), diags
	}

	if v.IsUnknown() {
		return types.ObjectUnknown(attributeTypes), diags
	}

	objVal, diags := types.ObjectValue(
		attributeTypes,
		map[string]attr.Value{
			"attribute": v.Attribute,
			"condition": v.Condition,
			"value":     v.Value,
		})

	return objVal, diags
}

func (v FiltersValue) Equal(o attr.Value) bool {
	other, ok := o.(FiltersValue)

	if !ok {
		return false
	}

	if v.state != other.state {
		return false
	}

	if v.state != attr.ValueStateKnown {
		return true
	}

	if !v.Attribute.Equal(other.Attribute) {
		return false
	}

	if !v.Condition.Equal(other.Condition) {
		return false
	}

	if !v.Value.Equal(other.Value) {
		return false
	}

	return true
}

func (v FiltersValue) Type(ctx context.Context) attr.Type {
	return FiltersType{
		basetypes.ObjectType{
			AttrTypes: v.AttributeTypes(ctx),
		},
	}
}

func (v FiltersValue) AttributeTypes(ctx context.Context) map[string]attr.Type {
	return map[string]attr.Type{
		"attribute": basetypes.StringType{},
		"condition": basetypes.StringType{},
		"value":     basetypes.StringType{},
	}
}
