package dynamic

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// AnyToDynamicValue will convert any data to a terraform DynamicValue.
// Please, only use dynamic values if really necessary, preferably only
// on datasource to avoid weird state related bugs.
func AnyToDynamicValue(ctx context.Context, data any) (basetypes.DynamicValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	switch v := reflect.ValueOf(data); v.Kind() {
	case reflect.String:
		return types.DynamicValue(types.StringValue(v.String())), nil
	case reflect.Bool:
		return types.DynamicValue(types.BoolValue(v.Bool())), nil
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		return types.DynamicValue(types.Int64Value(v.Int())), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
		return types.DynamicValue(types.NumberValue(big.NewFloat(float64(v.Uint())))), nil
	case reflect.Float32, reflect.Float64:
		return types.DynamicValue(types.Float64Value(v.Float())), nil

	case reflect.Array, reflect.Slice:
		_, attrValue, d := AnyToAttributeTypeAndValue(ctx, v.Interface())
		diags.Append(d...)
		if diags.HasError() {
			return types.DynamicNull(), diags
		}
		return types.DynamicValue(attrValue), diags

	case reflect.Map:
		_, attrValue, d := AnyToAttributeTypeAndValue(ctx, v.Interface())
		diags.Append(d...)
		return types.DynamicValue(attrValue), diags

	case reflect.Struct:
		_, attrValue, d := AnyToAttributeTypeAndValue(ctx, v.Interface())
		diags.Append(d...)
		return types.DynamicValue(attrValue), diags

	default:
		return types.DynamicNull(), diag.Diagnostics{diag.NewErrorDiagnostic(
			"Unsupported type",
			fmt.Sprintf("Cannot convert type %T to a dynamic Terraform value", data),
		)}
	}
}

// AnyToAttributeTypeAndValue will take any data and return its terraform type and value.
// Int and Float will all use 64 versions, UInt will be converted to numbers
func AnyToAttributeTypeAndValue(ctx context.Context, data any) (attr.Type, attr.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch v := reflect.ValueOf(data); v.Kind() {
	case reflect.String:
		return types.StringType, types.StringValue(v.String()), nil
	case reflect.Bool:
		return types.BoolType, types.BoolValue(v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		return types.Int64Type, types.Int64Value(v.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
		return types.NumberType, types.NumberValue(big.NewFloat(float64(v.Uint()))), nil
	case reflect.Float32, reflect.Float64:
		return types.Float64Type, types.Float64Value(v.Float()), nil

	case reflect.Array, reflect.Slice:
		var length = v.Len()
		var ts = make([]attr.Type, length)
		var vs = make([]attr.Value, length)
		for i := range length {
			t, v, d := AnyToAttributeTypeAndValue(ctx, v.Index(i).Interface())
			diags.Append(d...)
			if diags.HasError() {
				return types.TupleType{ElemTypes: ts}, types.TupleValueMust(ts, vs), diags
			}

			ts[i] = t
			vs[i] = v
		}

		// Terraform lists don't support multiple types, so if
		// the slice has more than one type, we use a tuple instead.
		singleType := true
		if len(ts) > 1 {
			for _, t := range ts[1:] {
				if ts[0].String() != t.String() {
					singleType = false
					break
				}
			}
		}

		if singleType {
			var singleType attr.Type
			if len(slices.Compact(ts)) == 0 {
				singleType = types.StringType
			} else {
				singleType = ts[0]
			}

			list, d := types.ListValue(singleType, vs)
			diags.Append(d...)
			if diags.HasError() {
				return nil, nil, diags
			}

			return types.ListType{ElemType: singleType}, list, diags
		} else {
			tuple, d := types.TupleValue(ts, vs)
			diags.Append(d...)
			if diags.HasError() {
				return types.TupleType{ElemTypes: ts}, tuple, diags
			}

			return tuple.Type(ctx), tuple, diags
		}

	case reflect.Map:
		// Maps don't support dynamic type attributes nor attributes of different types.
		// To make the logic simpler, we return an object everytime.
		length := v.Len()
		attrValues := make(map[string]attr.Value, length)
		attrTypes := make(map[string]attr.Type, length)
		for _, key := range v.MapKeys() {
			attrType, attrValue, d := AnyToAttributeTypeAndValue(ctx, v.MapIndex(key).Interface())
			diags.Append(d...)
			if diags.HasError() {
				return types.ObjectType{AttrTypes: map[string]attr.Type{}}, types.ObjectNull(map[string]attr.Type{}), diags
			}

			keyName := key.String()
			attrValues[keyName] = attrValue
			attrTypes[keyName] = attrType
		}

		objectValue, d := types.ObjectValue(attrTypes, attrValues)
		diags.Append(d...)
		if diags.HasError() {
			return types.ObjectType{AttrTypes: map[string]attr.Type{}}, types.ObjectNull(map[string]attr.Type{}), diags
		}

		return objectValue.Type(ctx), objectValue, diags

	case reflect.Struct:
		attrTypes := make(map[string]attr.Type)
		attrValues := make(map[string]attr.Value)
		structType := reflect.TypeOf(data)
		structFields := reflect.VisibleFields(structType)
		structValue := reflect.ValueOf(data)
		for i, structField := range structFields {
			value := structValue.FieldByName(structField.Name)
			fieldName := structField.Name

			if !value.CanInterface() {
				continue
			}

			attrType, attrValue, d := AnyToAttributeTypeAndValue(ctx, v.Field(i).Interface())
			diags.Append(d...)
			if diags.HasError() {
				return types.ObjectType{AttrTypes: map[string]attr.Type{}}, types.ObjectNull(map[string]attr.Type{}), diags
			}

			if fieldJSONTag := structField.Tag.Get("json"); fieldJSONTag != "" {
				fieldName = fieldJSONTag
			}
			attrValues[fieldName] = attrValue
			attrTypes[fieldName] = attrType
		}

		objValue, d := types.ObjectValue(attrTypes, attrValues)
		diags.Append(d...)
		if diags.HasError() {
			return types.ObjectType{AttrTypes: attrTypes}, types.ObjectNull(attrTypes), diags
		}

		return types.ObjectType{AttrTypes: attrTypes}, objValue, diags

	default:
		diags.AddError("Unsuported type "+v.Kind().String()+" for value", "This is an error from the provider, please reach out to the developper")
		return types.StringType, types.StringNull(), diags
	}
}
