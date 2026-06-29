package resource_organization_environment

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func OrganizationEnvironmentResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "This resource manages an organization-scoped Cycloid environment. Unlike `cycloid_environment`, it is NOT linked to any project: the environment is a first-class organization entity that can later be attached to one or more projects via `cycloid_environment_link`. Creating this resource creates the environment in the organization; deleting it deletes the environment itself. Docs: https://docs.cycloid.io/reference/core-concepts/",
		MarkdownDescription: "This resource manages an organization-scoped Cycloid environment. Unlike [`cycloid_environment`](./environment.md), it is **not** linked to any project: the environment is a first-class organization entity that can later be attached to one or more projects via [`cycloid_environment_link`](./environment_link.md). Creating this resource creates the environment in the organization; deleting it deletes the environment itself. [Docs](https://docs.cycloid.io/reference/core-concepts/).",
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
				Description:         "Stable identifier of the environment. Either `name` or `canonical` must be set. Changing it forces a new resource.",
				MarkdownDescription: "Stable identifier of the environment. Either `name` or `canonical` must be set. Changing it forces a new resource.",
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
				Description:         "Canonical of the `cycloid_environment_type` to associate with this environment (e.g. `production`, `staging`). Defaults to `production` when omitted.",
				MarkdownDescription: "Canonical of the [`cycloid_environment_type`](./environment_type.md) to associate with this environment (e.g. `production`, `staging`). Defaults to `production` when omitted.",
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

type OrganizationEnvironmentVariableModel struct {
	Key         types.String `tfsdk:"key"`
	Type        types.String `tfsdk:"type"`
	Value       types.String `tfsdk:"value"`
	Description types.String `tfsdk:"description"`
	Sensitive   types.Bool   `tfsdk:"sensitive"`
}

type OrganizationEnvironmentModel struct {
	Canonical              types.String `tfsdk:"canonical"`
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
