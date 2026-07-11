package resource_organization

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// conflictsWhenBothTrueValidator errors when this bool attribute and the attribute
// at `other` are BOTH true.
//
// It replaces boolvalidator.ConflictsWith on allow_destroy / soft_destroy: that
// validator is presence-based and false-positives on these Computed attributes,
// which always carry a (default false) value, so it rejected even the default
// state. This one only triggers on two concrete true values, so false/false (the
// default) and "exactly one true" are always valid — only the contradictory
// "both true" is rejected.
type conflictsWhenBothTrueValidator struct {
	other path.Expression
}

// ConflictsWhenBothTrue returns a Bool validator that rejects a config in which
// this attribute and the attribute at `other` are both set to true.
func ConflictsWhenBothTrue(other path.Expression) validator.Bool {
	return conflictsWhenBothTrueValidator{other: other}
}

func (v conflictsWhenBothTrueValidator) Description(_ context.Context) string {
	return fmt.Sprintf("must not be true at the same time as %s", v.other)
}

func (v conflictsWhenBothTrueValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v conflictsWhenBothTrueValidator) ValidateBool(ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	// Only a concrete true value can conflict; null / unknown / false are fine.
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || !req.ConfigValue.ValueBool() {
		return
	}

	matchedPaths, diags := req.Config.PathMatches(ctx, v.other)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, mp := range matchedPaths {
		if mp.Equal(req.Path) {
			continue
		}

		var otherVal types.Bool
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, mp, &otherVal)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if !otherVal.IsNull() && !otherVal.IsUnknown() && otherVal.ValueBool() {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Attribute Combination",
				fmt.Sprintf("%q and %q cannot both be true.", req.Path.String(), mp.String()),
			)
		}
	}
}
