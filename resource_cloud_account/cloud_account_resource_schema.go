package resource_cloud_account

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

func CloudAccountResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:         "Manage organization-scoped Cloud Accounts in Cycloid. A Cloud Account wraps a `cycloid_credential` and binds it to a cloud provider so it can be linked to environments. Built-in providers are `aws`, `google` and `azurerm`; any other canonical is treated as a custom provider for the organization.",
		MarkdownDescription: "Manage organization-scoped Cloud Accounts in Cycloid. A Cloud Account wraps a [`cycloid_credential`](./credential.md) and binds it to a cloud provider so it can be linked to environments. Built-in providers are `aws`, `google` and `azurerm`; any other canonical is treated as a custom provider for the organization.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical where the cloud account lives. Defaults to the provider's `default_organization`.",
				MarkdownDescription: "The organization canonical where the cloud account lives. Defaults to the provider's `default_organization`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description:         "Display name for the cloud account, shown in the UI. Either `name` or `canonical` must be set.",
				MarkdownDescription: "Display name for the cloud account, shown in the UI. Either `name` or `canonical` must be set.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
					stringvalidator.AtLeastOneOf(
						path.MatchRoot("name"),
						path.MatchRoot("canonical"),
					),
				},
			},
			"canonical": schema.StringAttribute{
				Description:         "Stable identifier for the cloud account. Lower-case alphanumerics with `-_` separators, 3-100 chars. Inferred from `name` when omitted. Changing the canonical forces a replacement.",
				MarkdownDescription: "Stable identifier for the cloud account. Lower-case alphanumerics with `-_` separators, 3-100 chars. Inferred from `name` when omitted. Changing the canonical forces a replacement.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9]+[a-z0-9\-_]+[a-z0-9]+$`),
						"must match ^[a-z0-9]+[a-z0-9\\-_]+[a-z0-9]+$",
					),
				},
			},
			"cloud_provider": schema.StringAttribute{
				Description:         "Canonical of the cloud provider this account targets. Built-ins: `aws`, `google`, `azurerm` (aliases like `gcp` and `microsoft_azure` are normalized server-side). Any other canonical is treated as a custom org-scoped provider. Cannot be changed after creation; updates trigger replacement.",
				MarkdownDescription: "Canonical of the cloud provider this account targets. Built-ins: `aws`, `google`, `azurerm` (aliases like `gcp` and `microsoft_azure` are normalized server-side). Any other canonical is treated as a custom org-scoped provider. Cannot be changed after creation; updates trigger replacement.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9]+[a-z0-9\-_]+[a-z0-9]+$`),
						"must match ^[a-z0-9]+[a-z0-9\\-_]+[a-z0-9]+$",
					),
				},
			},
			"credential_canonical": schema.StringAttribute{
				Description:         "Canonical of the `cycloid_credential` to wrap. The credential type must match the cloud provider for built-in providers; any credential type is accepted for custom providers (typically `custom`).",
				MarkdownDescription: "Canonical of the [`cycloid_credential`](./credential.md) to wrap. The credential type must match the cloud provider for built-in providers; any credential type is accepted for custom providers (typically `custom`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
				},
			},
			"description": schema.StringAttribute{
				Description:         "Free-form description of the cloud account.",
				MarkdownDescription: "Free-form description of the cloud account.",
				Optional:            true,
				Computed:            true,
			},
			"owner": schema.StringAttribute{
				Description:         "Username of the organization member that owns this cloud account. The owner has full permissions on the cloud account. Defaults to the API key owner at creation; preserved on update unless explicitly changed.",
				MarkdownDescription: "Username of the organization member that owns this cloud account. The owner has full permissions on the cloud account. Defaults to the API key owner at creation; preserved on update unless explicitly changed.",
				Optional:            true,
				Computed:            true,
			},
			"id": schema.Int64Attribute{
				Description:         "Internal numeric ID assigned by the Cycloid API.",
				MarkdownDescription: "Internal numeric ID assigned by the Cycloid API.",
				Computed:            true,
			},
		},
	}
}

type CloudAccountModel struct {
	Organization        types.String `tfsdk:"organization"`
	Name                types.String `tfsdk:"name"`
	Canonical           types.String `tfsdk:"canonical"`
	CloudProvider       types.String `tfsdk:"cloud_provider"`
	CredentialCanonical types.String `tfsdk:"credential_canonical"`
	Description         types.String `tfsdk:"description"`
	Owner               types.String `tfsdk:"owner"`
	ID                  types.Int64  `tfsdk:"id"`
}
