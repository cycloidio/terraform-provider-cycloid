package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"reflect"

	"github.com/cycloidio/terraform-provider-cycloid/datasource_terraform_outputs"
	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &terraformOutputsDataSource{}

type terraformOutputsDatasourceModel = datasource_terraform_outputs.TerraformOutputsModel

type terraformOutputsDataSource struct {
	provider provider_cycloid.CycloidModel
}

func NewTerraformOutputsDataSource() datasource.DataSource {
	return &terraformOutputsDataSource{}
}

func (t terraformOutputsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_terraform_outputs"
}

func (t *terraformOutputsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	schema := datasource_terraform_outputs.TerraformOutputsDataSourceSchema(ctx)
	resp.Schema = schema
}

func (t *terraformOutputsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	pv, ok := req.ProviderData.(provider_cycloid.CycloidModel)
	if !ok {
		tflog.Error(ctx, "Unable to init client")
	}

	t.provider = pv
}

func (t *terraformOutputsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data terraformOutputsDatasourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if t.provider.Jwt.IsUnknown() || t.provider.Jwt.IsNull() {
		resp.Diagnostics.AddError("API token for cycloid is mising", "")
		return
	}

	var organization string
	if data.Organization.IsNull() || data.Organization.IsUnknown() {
		organization = t.provider.OrganizationCanonical.ValueString()
	} else {
		organization = data.Organization.ValueString()
	}

	if organization == "" {
		resp.Diagnostics.AddAttributeError(path.Root("organization"), "org should not be empty", "fill it by the provider or the datasource settings")
		return
	}

	// Fetch logic
	// We will not use the middleware because we need LHS filter that are undocumented
	apiUrl := fmt.Sprintf("%s/organizations/%s/inventory/outputs", t.provider.Url.ValueString(), organization)

	var filters []datasource_terraform_outputs.Filter = nil
	if !data.Filters.IsNull() && !data.Filters.IsUnknown() {
		elements, listDiags := data.Filters.ToListValue(ctx)
		resp.Diagnostics.Append(listDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		resp.Diagnostics.Append(
			elements.ElementsAs(ctx, &filters, false)...,
		)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	params := make(url.Values)
	for _, filter := range filters {
		params.Add(fmt.Sprintf("%s[%s]", filter.Attribute, filter.Condition), filter.Value)
	}

	url := apiUrl + "?" + params.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create http client, API url may be invalid", err.Error())
		return
	}
	request.Header.Add("Content-Type", "Application/json")
	request.Header.Add("Authorization", "Bearer "+t.provider.Jwt.ValueString())

	client := http.DefaultClient
	response, err := client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("failed to list credentials", err.Error())
		return
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		resp.Diagnostics.AddError("failed to read response body from API", err.Error())
		return
	}

	payloadJSON := struct {
		Data []datasource_terraform_outputs.TerraformOutput `json:"data"`
	}{}
	err = json.Unmarshal(payload, &payloadJSON)
	if err != nil {
		resp.Diagnostics.AddError("failed to read JSON from API", err.Error())
		return
	}
	terraformOutputs := payloadJSON.Data
	terraformOutputsValue, diags := AnyToDynamicValue(ctx, terraformOutputs)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	data.Outputs = terraformOutputsValue
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

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

	// case reflect.:
	// 	return types.DynamicNull()), nil

	case reflect.Array, reflect.Slice:
		_, attrValue, d := ValueAndType(ctx, v.Interface())
		diags.Append(d...)
		if diags.HasError() {
			return types.DynamicNull(), diags
		}
		return types.DynamicValue(attrValue), diags

	case reflect.Map:
		_, attrValue, d := ValueAndType(ctx, v.Interface())
		diags.Append(d...)
		return types.DynamicValue(attrValue), diags

	case reflect.Struct:
		_, attrValue, d := ValueAndType(ctx, v.Interface())
		diags.Append(d...)
		return types.DynamicValue(attrValue), diags

	default:
		return types.DynamicNull(), diag.Diagnostics{diag.NewErrorDiagnostic(
			"Unsupported type",
			fmt.Sprintf("Cannot convert type %T to a dynamic Terraform value", data),
		)}
	}
}

func ValueAndType(ctx context.Context, data any) (attr.Type, attr.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	tflog.Debug(ctx, "Evaluating value", map[string]any{
		"type": reflect.ValueOf(data).Kind().String(),
	})
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
			t, v, d := ValueAndType(ctx, v.Index(i).Interface())
			diags.Append(d...)
			if diags.HasError() {
				return types.TupleType{ElemTypes: ts}, types.TupleValueMust(ts, vs), diags
			}

			ts[i] = t
			vs[i] = v
		}

		// singleType := slices.Compact(cpy) // compact mutate the slice, so I copied it
		// if len(singleType) > 1 {
		tuple, d := types.TupleValue(ts, vs)
		diags.Append(d...)
		if diags.HasError() {
			return types.TupleType{ElemTypes: ts}, tuple, diags
		}

		return tuple.Type(ctx), tuple, diags
		// } else {
		// 	var singleType attr.Type
		// 	if len(slices.Compact(ts)) == 0 {
		// 		singleType = types.StringType
		// 	} else {
		// 		singleType = ts[0]
		// 	}
		//
		// 	list, d := types.ListValue(singleType, vs)
		// 	diags.Append(d...)
		// 	if diags.HasError() {
		// 		return nil, nil, diags
		// 	}
		//
		// 	return types.ListType{ElemType: singleType}, list, diags
		// }

	case reflect.Map:
		length := v.Len()
		attrValues := make(map[string]attr.Value, length)
		for _, key := range v.MapKeys() {
			attrValue, d := AnyToDynamicValue(ctx, v.MapIndex(key).Interface())
			diags.Append(d...)
			if diags.HasError() {
				return types.MapType{}, types.MapNull(types.DynamicType), diags
			}

			keyName := key.String()
			attrValues[keyName] = attrValue
		}

		mapValue, d := types.MapValue(types.DynamicType, attrValues)
		diags.Append(d...)
		if diags.HasError() {
			return types.MapType{}, types.MapNull(types.DynamicType), diags
		}

		return types.MapType{ElemType: types.DynamicType}, mapValue, diags

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

			attrType, attrValue, d := ValueAndType(ctx, v.Field(i).Interface())
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
