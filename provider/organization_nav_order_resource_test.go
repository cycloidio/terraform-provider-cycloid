package provider

import (
	"context"
	"testing"

	cycloidmiddleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/resource_organization_nav_order"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit tests — no TF_ACC required.

func TestNavItemsFromData_NullAndUnknownAlwaysExplicit(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name string
		val  types.List
	}{
		{name: "null", val: types.ListNull(itemObjectType())},
		{name: "unknown", val: types.ListUnknown(itemObjectType())},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			items, diags := navItemsFromData(ctx, tc.val)
			require.False(t, diags.HasError(), "diags: %v", diags)
			require.NotNil(t, items, "must never be nil — an explicit [] is required to reset the ordering")
			assert.Empty(t, items)
		})
	}
}

func TestNavItemsFromData_MapsFields(t *testing.T) {
	ctx := context.Background()

	itemsVal, diags := types.ListValueFrom(ctx, itemObjectType(), []resource_organization_nav_order.NavItemModel{
		{Type: types.StringValue("native"), Key: types.StringValue("dashboard"), Position: types.Int64Value(1)},
		{Type: types.StringValue("plugin_widget"), Key: types.StringValue("42"), Position: types.Int64Value(2)},
	})
	require.False(t, diags.HasError(), "ListValueFrom: %v", diags)

	items, diags := navItemsFromData(ctx, itemsVal)
	require.False(t, diags.HasError(), "diags: %v", diags)
	require.Len(t, items, 2)

	assert.Equal(t, "native", items[0].Type)
	assert.Equal(t, "dashboard", items[0].Key)
	assert.Equal(t, uint32(1), items[0].Position)

	assert.Equal(t, "plugin_widget", items[1].Type)
	assert.Equal(t, "42", items[1].Key)
	assert.Equal(t, uint32(2), items[1].Position)
}

func TestNavConfigToData(t *testing.T) {
	ctx := context.Background()

	config := &cycloidmiddleware.NavConfig{
		Items: []*cycloidmiddleware.NavItem{
			{Type: "native", Key: "dashboard", Position: 1},
			{Type: "plugin_widget", Key: "7", Position: 2},
		},
	}

	var data organizationNavOrderResourceModel
	diags := navConfigToData(ctx, "my-org", config, &data)
	require.False(t, diags.HasError(), "diags: %v", diags)

	assert.Equal(t, "my-org", data.Organization.ValueString())

	var itemModels []resource_organization_nav_order.NavItemModel
	diags = data.Items.ElementsAs(ctx, &itemModels, false)
	require.False(t, diags.HasError(), "ElementsAs: %v", diags)
	require.Len(t, itemModels, 2)

	assert.Equal(t, "native", itemModels[0].Type.ValueString())
	assert.Equal(t, "dashboard", itemModels[0].Key.ValueString())
	assert.Equal(t, int64(1), itemModels[0].Position.ValueInt64())
}

func TestNavConfigToData_EmptyItems(t *testing.T) {
	ctx := context.Background()

	config := &cycloidmiddleware.NavConfig{Items: []*cycloidmiddleware.NavItem{}}

	var data organizationNavOrderResourceModel
	diags := navConfigToData(ctx, "my-org", config, &data)
	require.False(t, diags.HasError(), "diags: %v", diags)

	assert.False(t, data.Items.IsNull(), "empty items from the API must map to a non-null empty list")
	assert.Len(t, data.Items.Elements(), 0)
}
