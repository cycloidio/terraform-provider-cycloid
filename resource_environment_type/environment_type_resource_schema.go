package resource_environment_type

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func EnvironmentTypeResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Manage organization-scoped environment types in Cycloid. Environment types categorize environments (e.g. production, staging) and carry a color used by the UI. Built-in defaults (production, staging, development) are flagged as `is_default` and cannot be renamed or deleted.",
		MarkdownDescription: "Manage organization-scoped environment types in Cycloid. Environment types categorize environments (e.g. `production`, `staging`) and carry a color used by the UI. Built-in defaults (`production`, `staging`, `development`) are flagged as `is_default` and cannot be renamed or deleted.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical where the environment type lives. Defaults to the provider's `default_organization`.",
				MarkdownDescription: "The organization canonical where the environment type lives. Defaults to the provider's `default_organization`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description:         "Display name of the environment type. Either `name` or `canonical` must be set.",
				MarkdownDescription: "Display name of the environment type. Either `name` or `canonical` must be set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(
						path.MatchRoot("name"),
						path.MatchRoot("canonical"),
					),
				},
			},
			"canonical": schema.StringAttribute{
				Description:         "Stable identifier for the environment type. Lower-case alphanumerics with `-_.` separators, 1-100 chars. Inferred from `name` when omitted.",
				MarkdownDescription: "Stable identifier for the environment type. Lower-case alphanumerics with `-_.` separators, 1-100 chars. Inferred from `name` when omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[\da-z]+(?:[\da-z\-._]+[\da-z]|[\da-z])$`),
						"must match ^[\\da-z]+(?:[\\da-z\\-._]+[\\da-z]|[\\da-z])$",
					),
				},
			},
			"color": schema.StringAttribute{
				Description:         "Color used to render this environment type in the UI. Free-form string up to 64 chars (hex codes such as `#27ae60` or named colors are accepted).",
				MarkdownDescription: "Color used to render this environment type in the UI. Free-form string up to 64 chars (hex codes such as `#27ae60` or named colors are accepted).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(64),
				},
			},
			"is_default": schema.BoolAttribute{
				Description:         "True for built-in environment types (`production`, `staging`, `development`). Built-ins cannot be renamed or deleted.",
				MarkdownDescription: "True for built-in environment types (`production`, `staging`, `development`). Built-ins cannot be renamed or deleted.",
				Computed:            true,
			},
			"environments_count": schema.Int64Attribute{
				Description:         "Number of environments currently using this type.",
				MarkdownDescription: "Number of environments currently using this type.",
				Computed:            true,
			},
			"id": schema.Int64Attribute{
				Description:         "Internal numeric ID assigned by the Cycloid API.",
				MarkdownDescription: "Internal numeric ID assigned by the Cycloid API.",
				Computed:            true,
			},
			"label_selector": schema.SingleNestedAttribute{
				Description:         "Label selector constraining which stacks are offered in environments of this type. Omit to leave unconstrained.",
				MarkdownDescription: "Label selector constraining which stacks are offered in environments of this type. Omit to leave unconstrained.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"enforcement": schema.StringAttribute{
						Description:         "Enforcement mode: 'soft' only filters the stack list; 'hard' also rejects creating a component from a non-matching stack. Defaults to 'soft'.",
						MarkdownDescription: "Enforcement mode: `soft` only filters the stack list; `hard` also rejects creating a component from a non-matching stack. Defaults to `soft`.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("soft", "hard"),
						},
					},
					"requirements": schema.ListNestedAttribute{
						Description:         "Label requirements. A stack must satisfy ALL requirements to match (AND semantics).",
						MarkdownDescription: "Label requirements. A stack must satisfy ALL requirements to match (AND semantics).",
						Required:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									Description:         "Label key to match against.",
									MarkdownDescription: "Label key to match against.",
									Required:            true,
									Validators: []validator.String{
										stringvalidator.LengthBetween(1, 255),
									},
								},
								"operator": schema.StringAttribute{
									Description:         "Matching operator: 'eq' (exact single value) or 'in' (at least one value matches).",
									MarkdownDescription: "Matching operator: `eq` (exact single value) or `in` (at least one value matches).",
									Required:            true,
									Validators: []validator.String{
										stringvalidator.OneOf("eq", "in"),
									},
								},
								"values": schema.ListAttribute{
									Description:         "Values to match. For 'eq' operator, exactly one value must be provided.",
									MarkdownDescription: "Values to match. For `eq` operator, exactly one value must be provided.",
									Required:            true,
									ElementType:         types.StringType,
								},
							},
						},
					},
				},
			},
		},
	}
}

// LabelSelectorRequirementModel represents a single label requirement in TF state.
type LabelSelectorRequirementModel struct {
	Key      types.String `tfsdk:"key"`
	Operator types.String `tfsdk:"operator"`
	Values   types.List   `tfsdk:"values"`
}

// LabelSelectorModel represents the label selector in TF state.
type LabelSelectorModel struct {
	Enforcement  types.String `tfsdk:"enforcement"`
	Requirements types.List   `tfsdk:"requirements"`
}

type EnvironmentTypeModel struct {
	Organization      types.String `tfsdk:"organization"`
	Name              types.String `tfsdk:"name"`
	Canonical         types.String `tfsdk:"canonical"`
	Color             types.String `tfsdk:"color"`
	IsDefault         types.Bool   `tfsdk:"is_default"`
	EnvironmentsCount types.Int64  `tfsdk:"environments_count"`
	ID                types.Int64  `tfsdk:"id"`
	LabelSelector     types.Object `tfsdk:"label_selector"`
}
