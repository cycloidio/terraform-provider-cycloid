package dynamic

import (
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
		// {
		// 	Name:    "ListStringOK",
		// 	In:      []string{"one", "two", "three"},
		// 	OutType: types.ListType{ElemType: types.StringType},
		// 	OutValue: types.ListValueMust(types.StringType, []attr.Value{
		// 		types.StringValue("one"),
		// 		types.StringValue("two"),
		// 		types.StringValue("three"),
		// 	}),
		// },
		// {
		// 	Name:    "ListInt64Ok",
		// 	In:      []int{1, -1, 0},
		// 	OutType: types.ListType{ElemType: types.Int64Type},
		// 	OutValue: types.ListValueMust(types.Int64Type, []attr.Value{
		// 		types.Int64Value(1),
		// 		types.Int64Value(-1),
		// 		types.Int64Value(0),
		// 	}),
		// },
		// {
		// 	Name:    "ListUInt64Ok",
		// 	In:      []uint64{1, 9999, 0},
		// 	OutType: types.ListType{ElemType: types.NumberType},
		// 	OutValue: types.ListValueMust(types.NumberType, []attr.Value{
		// 		types.NumberValue(big.NewFloat(1)),
		// 		types.NumberValue(big.NewFloat(9999)),
		// 		types.NumberValue(big.NewFloat(0)),
		// 	}),
		// },
		// {
		// 	Name:    "ListFloatOk",
		// 	In:      []float64{1.1, -1.1, 0},
		// 	OutType: types.ListType{ElemType: types.Float64Type},
		// 	OutValue: types.ListValueMust(types.Float64Type, []attr.Value{
		// 		types.Float64Value(1.1),
		// 		types.Float64Value(-1.1),
		// 		types.Float64Value(0),
		// 	}),
		// },
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

func TestAttrValueToAny(t *testing.T) {
	testCases := []struct {
		Name      string
		Input     attr.Value
		Expected  any
		ExpectErr bool
	}{
		{
			Name:     "StringOK",
			Input:    types.StringValue("hello world"),
			Expected: "hello world",
		},
		{
			Name:     "StringNull",
			Input:    types.StringNull(),
			Expected: "",
		},
		{
			Name:     "Int32OK",
			Input:    types.Int32Value(42),
			Expected: int32(42),
		},
		{
			Name:     "Int32Null",
			Input:    types.Int32Null(),
			Expected: int32(0),
		},
		{
			Name:     "Int64OK",
			Input:    types.Int64Value(1234567890),
			Expected: int64(1234567890),
		},
		{
			Name:     "Int64Null",
			Input:    types.Int64Null(),
			Expected: int64(0),
		},
		{
			Name:     "NumberIntOK",
			Input:    types.NumberValue(big.NewFloat(42)),
			Expected: int64(42),
		},
		{
			Name:     "NumberFloatOK",
			Input:    types.NumberValue(big.NewFloat(3.14)),
			Expected: 3.14,
		},
		{
			Name:     "Float32OK",
			Input:    types.Float32Value(3.14),
			Expected: float32(3.14),
		},
		{
			Name:     "Float32Null",
			Input:    types.Float32Null(),
			Expected: float32(0),
		},
		{
			Name:     "Float64OK",
			Input:    types.Float64Value(2.718281828),
			Expected: 2.718281828,
		},
		{
			Name:     "Float64Null",
			Input:    types.Float64Null(),
			Expected: float64(0),
		},
		{
			Name:     "BoolTrue",
			Input:    types.BoolValue(true),
			Expected: true,
		},
		{
			Name:     "BoolFalse",
			Input:    types.BoolValue(false),
			Expected: false,
		},
		{
			Name:     "BoolNull",
			Input:    types.BoolNull(),
			Expected: false,
		},
		{
			Name: "ObjectOK",
			Input: types.ObjectValueMust(
				map[string]attr.Type{
					"name":   types.StringType,
					"age":    types.Int64Type,
					"active": types.BoolType,
				},
				map[string]attr.Value{
					"name":   types.StringValue("John Doe"),
					"age":    types.Int64Value(30),
					"active": types.BoolValue(true),
				},
			),
			Expected: map[string]any{
				"name":   "John Doe",
				"age":    int64(30),
				"active": true,
			},
		},
		{
			Name: "ObjectWithNullValues",
			Input: types.ObjectValueMust(
				map[string]attr.Type{
					"name":   types.StringType,
					"age":    types.Int64Type,
					"active": types.BoolType,
					"score":  types.Float64Type,
				},
				map[string]attr.Value{
					"name":   types.StringValue("Jane Doe"),
					"age":    types.Int64Null(),
					"active": types.BoolValue(false),
					"score":  types.Float64Value(95.5),
				},
			),
			Expected: map[string]any{
				"name":   "Jane Doe",
				"age":    int64(0),
				"active": false,
				"score":  95.5,
			},
		},
		{
			Name: "TupleOK",
			Input: types.TupleValueMust(
				[]attr.Type{types.StringType, types.Int64Type, types.BoolType, types.Float64Type},
				[]attr.Value{
					types.StringValue("test"),
					types.Int64Value(42),
					types.BoolValue(true),
					types.Float64Value(3.14),
				},
			),
			Expected: []any{
				"test",
				int64(42),
				true,
				3.14,
			},
		},
		{
			Name: "TupleWithNullValues",
			Input: types.TupleValueMust(
				[]attr.Type{types.StringType, types.Int64Type, types.BoolType},
				[]attr.Value{
					types.StringNull(),
					types.Int64Value(100),
					types.BoolNull(),
				},
			),
			Expected: []any{
				"",
				int64(100),
				false,
			},
		},
		{
			Name: "NestedObjectInTuple",
			Input: types.TupleValueMust(
				[]attr.Type{
					types.StringType,
					types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"field1": types.StringType,
							"field2": types.Int64Type,
						},
					},
				},
				[]attr.Value{
					types.StringValue("outer"),
					types.ObjectValueMust(
						map[string]attr.Type{
							"field1": types.StringType,
							"field2": types.Int64Type,
						},
						map[string]attr.Value{
							"field1": types.StringValue("inner"),
							"field2": types.Int64Value(200),
						},
					),
				},
			),
			Expected: []any{
				"outer",
				map[string]any{
					"field1": "inner",
					"field2": int64(200),
				},
			},
		},
		{
			Name: "NestedTupleInObject",
			Input: types.ObjectValueMust(
				map[string]attr.Type{
					"id": types.StringType,
					"values": types.TupleType{
						ElemTypes: []attr.Type{types.Int64Type, types.StringType},
					},
				},
				map[string]attr.Value{
					"id": types.StringValue("item1"),
					"values": types.TupleValueMust(
						[]attr.Type{types.Int64Type, types.StringType},
						[]attr.Value{
							types.Int64Value(123),
							types.StringValue("abc"),
						},
					),
				},
			),
			Expected: map[string]any{
				"id": "item1",
				"values": []any{
					int64(123),
					"abc",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run("Case"+tc.Name, func(t *testing.T) {
			result, diags := AttrValueToAny(t.Context(), tc.Input)

			if tc.ExpectErr {
				assert.True(t, diags.HasError(), "expected diagnostics to contain errors")
				return
			}

			assert.False(t, diags.HasError(), "expected no diagnostics errors, got: %v", diags)
			assert.Equal(t, tc.Expected, result, "converted value should match expected")
		})
	}
}
