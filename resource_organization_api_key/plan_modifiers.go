package resource_organization_api_key

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// listForceNewPlanModifier forces resource replacement when the list value changes.
type listForceNewPlanModifier struct{}

func (m listForceNewPlanModifier) Description(ctx context.Context) string {
	return "Forces replacement when the value changes."
}

func (m listForceNewPlanModifier) MarkdownDescription(ctx context.Context) string {
	return "Forces replacement when the value changes."
}

func (m listForceNewPlanModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.StateValue.IsNull() {
		return
	}
	if req.PlanValue.Equal(req.StateValue) {
		return
	}
	resp.RequiresReplace = true
}
