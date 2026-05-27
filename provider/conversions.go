package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

func ptrUint32ToInt64(v *uint32) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}

func ptrUint64ToInt64(v *uint64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}
