package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var envVarObjAttrTypes = map[string]attr.Type{
	"key":         types.StringType,
	"type":        types.StringType,
	"value":       types.StringType,
	"description": types.StringType,
	"sensitive":   types.BoolType,
}

// buildVariableList converts API variable items to a types.List suitable for TF state.
func buildVariableList(ctx context.Context, vars []*models.EnvironmentVariableItem) (types.List, diag.Diagnostics) {
	elemType := types.ObjectType{AttrTypes: envVarObjAttrTypes}

	if len(vars) == 0 {
		return types.ListValueMust(elemType, []attr.Value{}), nil
	}

	items := make([]attr.Value, len(vars))
	for i, v := range vars {
		obj, objDiags := types.ObjectValue(envVarObjAttrTypes, map[string]attr.Value{
			"key":         types.StringPointerValue(v.Key),
			"type":        types.StringPointerValue(v.Type),
			"value":       anyToString(v.Value),
			"description": types.StringValue(v.Description),
			"sensitive":   types.BoolPointerValue(v.Sensitive),
		})
		if objDiags.HasError() {
			return types.ListNull(elemType), objDiags
		}
		items[i] = obj
	}

	return types.ListValue(elemType, items)
}

// anyToString converts an API variable value (any) to a TF string attribute.
func anyToString(v any) types.String {
	switch val := v.(type) {
	case string:
		return types.StringValue(val)
	case float64:
		if val == float64(int64(val)) {
			return types.StringValue(strconv.FormatInt(int64(val), 10))
		}
		return types.StringValue(strconv.FormatFloat(val, 'f', -1, 64))
	case bool:
		return types.StringValue(strconv.FormatBool(val))
	case nil:
		return types.StringNull()
	default:
		return types.StringValue(fmt.Sprintf("%v", val))
	}
}

// stringToAny converts a TF string value to an API-compatible value, coercing by declared type.
func stringToAny(s types.String, varType string) any {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	raw := s.ValueString()
	switch varType {
	case "boolean":
		if b, err := strconv.ParseBool(raw); err == nil {
			return b
		}
	case "integer":
		if i, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return float64(i)
		}
	case "float":
		if f, err := strconv.ParseFloat(raw, 64); err == nil {
			return f
		}
	}
	return raw
}
