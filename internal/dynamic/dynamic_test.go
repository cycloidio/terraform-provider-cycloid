package dynamic

import (
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestAnyToDynamicValue(t *testing.T) {
	testCases := []struct {
		Name     string
		In       any
		OutValue attr.Value
		Diags    diag.Diagnostics
	}{
		{
			Name: "ObjectOk",
			In: map[string]any{
				"string": "str",
				"int":    1,
				"struct": struct {
					SomeField string `json:"some_field"`
					SomeInt   int    `json:"some_int"`
				}{
					SomeField: "string",
					SomeInt:   -1,
				},
			},
			OutValue: types.DynamicValue(types.ObjectValueMust(
				map[string]attr.Type{
					"string": types.StringType,
					"int":    types.Int64Type,
					"struct": types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"some_field": types.StringType,
							"some_int":   types.Int64Type,
						},
					},
				},
				map[string]attr.Value{
					"string": types.StringValue("str"),
					"int":    types.Int64Value(1),
					"struct": types.ObjectValueMust(
						map[string]attr.Type{
							"some_field": types.StringType,
							"some_int":   types.Int64Type,
						},
						map[string]attr.Value{
							"some_field": types.StringValue("string"),
							"some_int":   types.Int64Value(-1),
						},
					),
				},
			),
			),
		},
	}

	for _, tc := range testCases {
		t.Run("Case"+tc.Name, func(t *testing.T) {
			dynamicValue, d := AnyToDynamicValue(t.Context(), tc.In)
			if d.HasError() {
				t.Fatal(d.Errors())
			}

			assert.Equal(t, tc.OutValue, dynamicValue, "values should match")
		})
	}
}

func TestAnyToAttributeTypeAndValue(t *testing.T) {
	testCases := []struct {
		Name     string
		In       any
		OutType  attr.Type
		OutValue attr.Value
		Diags    diag.Diagnostics
	}{
		{
			Name:     "StringOK",
			In:       "someString",
			OutType:  types.StringType,
			OutValue: types.StringValue("someString"),
		},
		{
			Name:    "ListStringOK",
			In:      []string{"one", "two", "three"},
			OutType: types.ListType{ElemType: types.StringType},
			OutValue: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("one"),
				types.StringValue("two"),
				types.StringValue("three"),
			}),
		},
		{
			Name:    "ListInt64Ok",
			In:      []int{1, -1, 0},
			OutType: types.ListType{ElemType: types.Int64Type},
			OutValue: types.ListValueMust(types.Int64Type, []attr.Value{
				types.Int64Value(1),
				types.Int64Value(-1),
				types.Int64Value(0),
			}),
		},
		{
			Name:    "ListUInt64Ok",
			In:      []uint64{1, 9999, 0},
			OutType: types.ListType{ElemType: types.NumberType},
			OutValue: types.ListValueMust(types.NumberType, []attr.Value{
				types.NumberValue(big.NewFloat(1)),
				types.NumberValue(big.NewFloat(9999)),
				types.NumberValue(big.NewFloat(0)),
			}),
		},
		{
			Name:    "ListFloatOk",
			In:      []float64{1.1, -1.1, 0},
			OutType: types.ListType{ElemType: types.Float64Type},
			OutValue: types.ListValueMust(types.Float64Type, []attr.Value{
				types.Float64Value(1.1),
				types.Float64Value(-1.1),
				types.Float64Value(0),
			}),
		},
		{
			Name:    "TupleOK",
			In:      []any{1.1, -1.1, "coucou", true},
			OutType: types.TupleType{ElemTypes: []attr.Type{types.Float64Type, types.Float64Type, types.StringType, types.BoolType}},
			OutValue: types.TupleValueMust(
				[]attr.Type{types.Float64Type, types.Float64Type, types.StringType, types.BoolType},
				[]attr.Value{
					types.Float64Value(1.1),
					types.Float64Value(-1.1),
					types.StringValue("coucou"),
					types.BoolValue(true),
				}),
		},
		{
			Name: "StructOk",
			In: struct {
				SomeStr  string `json:"some_str"`
				someBool bool
				SomeInt  int
			}{
				SomeStr:  "toto",
				someBool: false,
				SomeInt:  177,
			},
			OutType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"some_str": types.StringType,
					"SomeInt":  types.Int64Type,
				},
			},
			OutValue: types.ObjectValueMust(
				map[string]attr.Type{
					"some_str": types.StringType,
					"SomeInt":  types.Int64Type,
				},
				map[string]attr.Value{
					"some_str": types.StringValue("toto"),
					"SomeInt":  types.Int64Value(177),
				},
			),
		},
		{
			Name: "MapAnyOk",
			In: map[string]any{
				"string": "str",
				"int":    1,
				"struct": struct {
					SomeField string `json:"some_field"`
					SomeInt   int    `json:"some_int"`
				}{
					SomeField: "string",
					SomeInt:   -1,
				},
			},
			OutType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"string": types.StringType,
					"int":    types.Int64Type,
					"struct": types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"some_field": types.StringType,
							"some_int":   types.Int64Type,
						},
					},
				},
			},
			OutValue: types.ObjectValueMust(
				map[string]attr.Type{
					"string": types.StringType,
					"int":    types.Int64Type,
					"struct": types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"some_field": types.StringType,
							"some_int":   types.Int64Type,
						},
					},
				},
				map[string]attr.Value{
					"string": types.StringValue("str"),
					"int":    types.Int64Value(1),
					"struct": types.ObjectValueMust(
						map[string]attr.Type{
							"some_field": types.StringType,
							"some_int":   types.Int64Type,
						},
						map[string]attr.Value{
							"some_field": types.StringValue("string"),
							"some_int":   types.Int64Value(-1),
						},
					),
				},
			),
		},
	}

	for _, tc := range testCases {
		t.Run("Case"+tc.Name, func(t *testing.T) {
			attrType, attrValue, d := AnyToAttributeTypeAndValue(t.Context(), tc.In)
			if d.HasError() {
				t.Fatal(d.Errors())
			}

			assert.Equal(t, tc.OutType, attrType, "type should match")
			assert.Equal(t, tc.OutValue, attrValue, "values should match")
		})
	}
}
