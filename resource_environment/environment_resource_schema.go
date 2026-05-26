package resource_environment

import (
	"context"
	"regexp"

	"github.com/cycloidio/terraform-provider-cycloid/internal/icons"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func EnvironmentResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "This resource manages Cycloid environments. With the org-scoped meta-gov-env API an environment is a first-class organization entity that can be linked to one or more projects. This resource owns one such link via the required `project` attribute; use `cycloid_environment_link` to attach the same environment to additional projects. Docs: https://docs.cycloid.io/reference/core-concepts/",
		MarkdownDescription: "This resource manages Cycloid environments. With the org-scoped meta-gov-env API an environment is a first-class organization entity that can be linked to one or more projects. This resource owns one such link via the required `project` attribute; use [`cycloid_environment_link`](./environment_link.md) to attach the same environment to additional projects. [Docs](https://docs.cycloid.io/reference/core-concepts/).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				Description:         "Internal numeric ID of the environment assigned by the Cycloid API.",
				MarkdownDescription: "Internal numeric ID of the environment assigned by the Cycloid API.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Display name of the environment, for the UI. Either `name` or `canonical` must be set.",
				MarkdownDescription: "Display name of the environment, for the UI. Either `name` or `canonical` must be set.",
			},
			"canonical": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Stable identifier of the environment. Either `name` or `canonical` must be set.",
				MarkdownDescription: "Stable identifier of the environment. Either `name` or `canonical` must be set.",
			},
			"project": schema.StringAttribute{
				Required:            true,
				Description:         "Project canonical that this resource will link the environment to at creation. The environment itself remains an organization-scoped entity; deleting this resource only unlinks it from this project. Use `cycloid_environment_link` for additional projects.",
				MarkdownDescription: "Project canonical that this resource will link the environment to at creation. The environment itself remains an organization-scoped entity; deleting this resource only unlinks it from this project. Use [`cycloid_environment_link`](./environment_link.md) for additional projects.",
			},
			"organization": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The organization where to create the environment. Defaults to the provider's `default_organization`.",
				MarkdownDescription: "The organization where to create the environment. Defaults to the provider's `default_organization`.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9]+[a-z0-9\-_]+[a-z0-9]+$`), ""),
				},
			},
			"color": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				DeprecationMessage:  "color is no longer carried by an environment; it now lives on the linked `cycloid_environment_type`. This attribute is accepted for backward compatibility and ignored on write.",
				Description:         "Deprecated. Color now lives on the linked `cycloid_environment_type` and is exposed read-only via `type`. Setting this attribute has no effect.",
				MarkdownDescription: "**Deprecated.** Color now lives on the linked [`cycloid_environment_type`](./environment_type.md) and is exposed read-only via `type`. Setting this attribute has no effect.",
				Validators: []validator.String{
					stringvalidator.OneOf(icons.ValidColors...),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Free-form description of the environment.",
				MarkdownDescription: "Free-form description of the environment.",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1000),
				},
			},
			"type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Canonical of the `cycloid_environment_type` to associate with this environment (e.g. `production`, `staging`). Defaults to `production` until the backend infers the type from the environment canonical.",
				MarkdownDescription: "Canonical of the [`cycloid_environment_type`](./environment_type.md) to associate with this environment (e.g. `production`, `staging`). Defaults to `production` until the backend infers the type from the environment canonical.",
			},
			"owner": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Username of the organization member that owns this environment. The owner has full permissions on the environment. Defaults to the API key owner at creation.",
				MarkdownDescription: "Username of the organization member that owns this environment. The owner has full permissions on the environment. Defaults to the API key owner at creation.",
			},
			"cloud_account_canonicals": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				Description:         "Canonicals of the `cycloid_cloud_account` entries to link to this environment. PATCH semantics: omitting the attribute leaves existing links untouched, an empty list `[]` unlinks all, a non-empty list replaces the set.",
				MarkdownDescription: "Canonicals of the [`cycloid_cloud_account`](./cloud_account.md) entries to link to this environment. PATCH semantics: omitting the attribute leaves existing links untouched, an empty list `[]` unlinks all, a non-empty list replaces the set.",
			},
			"variables": schema.ListNestedAttribute{
				Optional:            true,
				Description:         "Environment variables surfaced under `.environment.variables` during interpolation. PATCH semantics: omit to leave variables untouched, pass `[]` to wipe, pass a non-empty list to replace.",
				MarkdownDescription: "Environment variables surfaced under `.environment.variables` during interpolation. PATCH semantics: omit to leave variables untouched, pass `[]` to wipe, pass a non-empty list to replace.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Required:            true,
							Description:         "Variable identifier referenced as `($ .environment.variables.<key> $)`. Must contain at least one alphanumeric character and no dots.",
							MarkdownDescription: "Variable identifier referenced as `($ .environment.variables.<key> $)`. Must contain at least one alphanumeric character and no dots.",
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 255),
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^[^.]*[A-Za-z0-9][^.]*$`),
									"must contain at least one alphanumeric character and no dots",
								),
							},
						},
						"type": schema.StringAttribute{
							Required:            true,
							Description:         "Declared shape of the value. One of: string, boolean, integer, float, array, map.",
							MarkdownDescription: "Declared shape of the value. One of: `string`, `boolean`, `integer`, `float`, `array`, `map`.",
							Validators: []validator.String{
								stringvalidator.OneOf("string", "boolean", "integer", "float", "array", "map"),
							},
						},
						"value": schema.StringAttribute{
							Required:            true,
							Description:         "The variable value as a string. For non-string types, encode the value as its string representation (e.g. `\"true\"` for boolean, `\"42\"` for integer).",
							MarkdownDescription: "The variable value as a string. For non-string types, encode the value as its string representation (e.g. `\"true\"` for boolean, `\"42\"` for integer).",
						},
						"description": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							Description:         "Free-form description shown in the UI.",
							MarkdownDescription: "Free-form description shown in the UI.",
							Validators: []validator.String{
								stringvalidator.LengthAtMost(1000),
							},
						},
						"sensitive": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Description:         "When true, the UI masks the value. The API still returns the value in plaintext, so prefer `cycloid_credential` for true secrets.",
							MarkdownDescription: "When true, the UI masks the value. The API still returns the value in plaintext, so prefer [`cycloid_credential`](./credential.md) for true secrets.",
						},
					},
				},
			},
			"created_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Unix timestamp at which the environment was created.",
				MarkdownDescription: "Unix timestamp at which the environment was created.",
			},
			"updated_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Unix timestamp at which the environment was last updated.",
				MarkdownDescription: "Unix timestamp at which the environment was last updated.",
			},
		},
	}
}

type EnvironmentVariableModel struct {
	Key         types.String `tfsdk:"key"`
	Type        types.String `tfsdk:"type"`
	Value       types.String `tfsdk:"value"`
	Description types.String `tfsdk:"description"`
	Sensitive   types.Bool   `tfsdk:"sensitive"`
}

type EnvironmentModel struct {
	Project                types.String `tfsdk:"project"`
	Canonical              types.String `tfsdk:"canonical"`
	Color                  types.String `tfsdk:"color"`
	ID                     types.Int64  `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Organization           types.String `tfsdk:"organization"`
	Description            types.String `tfsdk:"description"`
	Type                   types.String `tfsdk:"type"`
	Owner                  types.String `tfsdk:"owner"`
	CloudAccountCanonicals types.List   `tfsdk:"cloud_account_canonicals"`
	Variables              types.List   `tfsdk:"variables"`
	CreatedAt              types.Int64  `tfsdk:"created_at"`
	UpdatedAt              types.Int64  `tfsdk:"updated_at"`
}
