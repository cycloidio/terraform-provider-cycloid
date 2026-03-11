package resource_component

import (
	"context"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/dynamicplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ComponentResourceSchema(ctx context.Context) schema.Schema {
	componentDescription := strings.Join([]string{
		"Manage components in Cycloid projects.",
		"",
		"Components are instances of stacks that run in specific environments.",
		"",
		"More information about components: [Components Documentation](https://docs.cycloid.io/reference/projects/concepts/components)",
	}, "\n")
	return schema.Schema{
		Description:         componentDescription,
		MarkdownDescription: componentDescription,
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:         "The organization canonical where to create the component, default to the provider's `default_organization`",
				MarkdownDescription: "The organization canonical where to create the component, default to the provider's `default_organization`",
				Optional:            true,
				Computed:            true,
			},
			"project": schema.StringAttribute{
				Description:         "The project canonical where to create the component.",
				MarkdownDescription: "The project canonical where to create the component.",
				Required:            true,
			},
			"environment": schema.StringAttribute{
				Description:         "The environment canonical where to create the component.",
				MarkdownDescription: "The environment canonical where to create the component.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The name of the component, displayed in the UI. Either this or `canonical` must be set.",
				MarkdownDescription: "The name of the component, displayed in the UI. Either this or `canonical` must be set.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(
						path.MatchRoot("name"),
						path.MatchRoot("canonical"),
					),
				},
			},
			"canonical": schema.StringAttribute{
				Description:         "The canonical of the component, either this or `name` must be set. The canonical will be inferred from the name if not set.",
				MarkdownDescription: "The canonical of the component, either this or `name` must be set. The canonical will be inferred from the name if not set.",
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
			"description": schema.StringAttribute{
				Description:         "The description of the component, displayed in the UI. Supports markdown formatting.",
				MarkdownDescription: "The description of the component, displayed in the UI. Supports markdown formatting.",
				Optional:            true,
			},
			"stack_ref": schema.StringAttribute{
				Description:         "The stack reference to use, the format is <org>:<stack_canonical>. You can list them using the CLI with `cy stack list`.",
				MarkdownDescription: "The stack reference to use, the format is <org>:<stack_canonical>. You can list them using the CLI with `cy stack list`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[^:]+:[^:]+$`),
						"must be in the format <organization>:<stack_canonical>",
					),
				},
			},
			"use_case": schema.StringAttribute{
				Description:         "The stack use case to use. You can list them using the CLI with `cy stack list`.",
				MarkdownDescription: "The stack use case to use. You can list them using the CLI with `cy stack list`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"stack_version": schema.StringAttribute{
				Description:         "The stack version to use, you can specify a branch name, a tag or a commit. Default to the catalog repository's default branch.",
				MarkdownDescription: "The stack version to use, you can specify a branch name, a tag or a commit. Default to the catalog repository's default branch.",
				Optional:            true,
				Computed:            true,
			},
			"allow_version_update": schema.BoolAttribute{
				Description:         "Whether Terraform will manage stack versions on each update. When disabled, versions are only applied on component creation. This setting is useful to allow users to manage versions through the UI.",
				MarkdownDescription: "Whether Terraform will manage stack versions on each update. When disabled, versions are only applied on component creation. This setting is useful to allow users to manage versions through the UI.",
				Optional:            true,
			},
			"allow_variable_update": schema.BoolAttribute{
				Description:         "Whether Terraform will manage variables on each update. When disabled, variables are only applied on component creation. This setting is useful to allow users to manage configuration through the UI.",
				MarkdownDescription: "Whether Terraform will manage variables on each update. When disabled, variables are only applied on component creation. This setting is useful to allow users to manage configuration through the UI.",
				Optional:            true,
			},
			"allow_destroy": schema.BoolAttribute{
				Description:         "Whether Terraform will allow destroying this component. When set to false, prevents accidental data loss. Many components deploy Terraform manifests, and deleting a component without running the destroy step first could lead to dangling infrastructure.",
				MarkdownDescription: "Whether Terraform will allow destroying this component. When set to false, prevents accidental data loss. Many components deploy Terraform manifests, and deleting a component without running the destroy step first could lead to dangling infrastructure.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"input_variables": schema.DynamicAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Dynamic{
					dynamicplanmodifier.UseStateForUnknown(),
				},
				Description: strings.Join([]string{
					"Stackforms variables for this component that will be applied on creation and updates.",
					"Stackforms define the configuration interface for stacks, allowing users to customize infrastructure deployment.",
					"",
					"More information: [Stackforms Documentation](https://docs.cycloid.io/reference/stackforms/)",
					"",
					"Expected format:",
					`input_variables = {
						"section_name" = {
							"group_name" = {
								"key" = value # Value type must match stackforms definition
							}
						}
					}`,
					"",
					"Section and group names must match the `name` attribute in the stack's stackforms configuration.",
				}, "\n"),
				MarkdownDescription: strings.Join([]string{
					"Stackforms variables for this component that will be applied on creation and updates.",
					"Stackforms define the configuration interface for stacks, allowing users to customize infrastructure deployment.",
					"",
					"More information: [Stackforms Documentation](https://docs.cycloid.io/reference/stackforms/)",
					"",
					"Expected format:",
					"```",
					`input_variables = {
						"section_name" = {
							"group_name" = {
								"key" = value # Value type must match stackforms definition
							}
						}
					}`,
					"```",
					"",
					"Section and group names must match the `name` attribute in the stack's stackforms configuration.",
				}, "\n"),
			},
			"current_config": schema.DynamicAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The current configuration of the component as returned by the API. This is a read-only attribute that shows the full component configuration including all variables.",
				Description:         "The current configuration of the component as returned by the API. This is a read-only attribute that shows the full component configuration including all variables.",
			},
		},
	}
}

type ComponentModel struct {
	Organization        types.String  `tfsdk:"organization"`
	Project             types.String  `tfsdk:"project"`
	Environment         types.String  `tfsdk:"environment"`
	Name                types.String  `tfsdk:"name"`
	Canonical           types.String  `tfsdk:"canonical"`
	Description         types.String  `tfsdk:"description"`
	StackRef            types.String  `tfsdk:"stack_ref"`
	StackVersion        types.String  `tfsdk:"stack_version"`
	UseCase             types.String  `tfsdk:"use_case"`
	AllowVersionUpdate  types.Bool    `tfsdk:"allow_version_update"`
	AllowVariableUpdate types.Bool    `tfsdk:"allow_variable_update"`
	AllowDestroy        types.Bool    `tfsdk:"allow_destroy"`
	InputVariables      types.Dynamic `tfsdk:"input_variables"`
	CurrentConfig       types.Dynamic `tfsdk:"current_config"`
}
