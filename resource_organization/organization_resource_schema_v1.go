package resource_organization

import (
	"context"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var (
	orgDescription = strings.Join([]string{
		"A Cycloid organization is the top-most entity level in Cycloid.",
		"Almost all resources in Cycloid are scoped by organizations.",
		"",
		"Organizations can be nested, meaning a org can have child and parent organizations.",
		"Lookup the [Cycloid documentation](https://docs.cycloid.io/reference/organizations) for more information on organizations.",
		"",
		"Warning: an API key created in an organization can only manage its children.",
	}, "\n")
)

func OrganizationResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Version:             1,
		Description:         orgDescription,
		MarkdownDescription: orgDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				Description:         "The id of the organization.",
				MarkdownDescription: "The id of the organization.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The name of an organization, fill either this or canonical at creation.",
				MarkdownDescription: "The name of an organization, fill either this or canonical at creation.",
			},
			"canonical": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The canonical of an organization, fill either this or name at creation.",
				MarkdownDescription: "The canonical of an organization, fill either this or name at creation.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9]+[a-z0-9\\-_]+[a-z0-9]+$"), ""),
				},
			},
			"parent_organization": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The canonical of the parent organization if you want this org to be a child organization.",
				MarkdownDescription: "The canonical of the parent organization if you want this org to be a child organization.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9]+[a-z0-9\\-_]+[a-z0-9]+$"), ""),
				},
			},
			"concourse": schema.SingleNestedAttribute{
				Description:         "Data related to concourse",
				MarkdownDescription: "Data related to concourse",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"team_name": schema.StringAttribute{
						Description:         "The name of the concourse team linked to this organization.",
						MarkdownDescription: "The name of the concourse team linked to this organization.",
						Computed:            true,
					},
					"url": schema.StringAttribute{
						Description:         "The URL to the concourse instance linked to this org.",
						MarkdownDescription: "The URL to the concourse instance linked to this org.",
						Computed:            true,
					},
					"port": schema.StringAttribute{
						Description:         "The port number of the concourse instance linked to this org.",
						MarkdownDescription: "The port number of the concourse instance linked to this org.",
						Computed:            true,
					},
				},
			},
			"is_root": schema.BoolAttribute{
				Computed:            true,
				Description:         "`true` if the organization is the root organization of the Cycloid console.",
				MarkdownDescription: "`true` if the organization is the root organization of the Cycloid console.",
			},
			"has_children": schema.BoolAttribute{
				Computed:            true,
				Description:         "`true` if the organization has child organizations.",
				MarkdownDescription: "`true` if the organization has child organizations.",
			},
			"licence": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Attributes related to the org licence, docs here: https://docs.cycloid.io/reference/organizations/concepts/licencing.",
				MarkdownDescription: "Attributes related to the org licence, [docs here](https://docs.cycloid.io/reference/organizations/concepts/licencing).",
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(
						path.MatchRoot("subscription"),
					),
				},
				Attributes: map[string]schema.Attribute{
					"expires_at_unix_timestamp": schema.Int64Attribute{
						Description:         "Unix timestamp (precise at the milliseconds) where this licence expires.",
						MarkdownDescription: "Unix timestamp (precise at the milliseconds) where this licence expires.",
						Computed:            true,
					},
					"expires_at_rfc3339": schema.StringAttribute{
						Description:         "Unix timestamp (precise at the milliseconds) in rfc3339 format where this licence expires.",
						MarkdownDescription: "Unix timestamp (precise at the milliseconds) in rfc3339 format where where this licence expires.",
						Computed:            true,
					},
					"key": schema.StringAttribute{
						Description:         "The licence key in JWT format. Required if `apply_licence` attribute is set.",
						MarkdownDescription: "The licence key in JWT format. Required if `apply_licence` attribute is set.",
						Required:            true,
						Sensitive:           true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(10),
						},
					},
					"members_count": schema.Int64Attribute{
						Description:         "number of allowed members for this licence, default to 5.",
						MarkdownDescription: "number of allowed members for this licence, default to 5.",
						Computed:            true,
					},
					"current_members": schema.Int64Attribute{
						Description:         "number of current members for this licence.",
						MarkdownDescription: "number of current members for this licence.",
						Computed:            true,
					},
					"is_on_prem": schema.BoolAttribute{
						Description:         "`true` if this licence is made for on-premise deployment",
						MarkdownDescription: "`true` if this licence is made for on-premise deployment",
						Computed:            true,
					},
				},
			},
			"subscription": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Attributes related to the org subscription, docs here: https://docs.cycloid.io/reference/organizations/concepts/licencing.",
				MarkdownDescription: "Attributes related to the org subscription, [docs here](https://docs.cycloid.io/reference/organizations/concepts/licencing).",
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(
						path.MatchRoot("licence"),
					),
				},
				Attributes: map[string]schema.Attribute{
					"expires_at_unix_timestamp": schema.Int64Attribute{
						Description:         "Unix timestamp (precise at the milliseconds) where this subscription expires.",
						MarkdownDescription: "Unix timestamp (precise at the milliseconds) where this subscription expires.",
						Computed:            true,
					},
					"expires_at_rfc3339": schema.StringAttribute{
						Description:         "Unix timestamp (precise at the milliseconds) in rfc3339 format where this subscription expires.",
						MarkdownDescription: "Unix timestamp (precise at the milliseconds) in rfc3339 format where where this subscription expires.",
						Computed:            true,
						Optional:            true,
					},
					"plan": schema.StringAttribute{
						Description:         "The type of plan of this subscription, can be either `free_tier` or `platform_teams`, default to `platform_teams`",
						MarkdownDescription: "The type of plan of this subscription, can be either `free_tier` or `platform_teams`, default to `platform_teams`",
						Validators: []validator.String{
							stringvalidator.OneOf("free_tier", "platform_teams"),
						},
						Optional: true,
						Computed: true,
					},
					"members_count": schema.Int64Attribute{
						Description:         "number of allowed members for this plan, default to 5.",
						MarkdownDescription: "number of allowed members for this plan, default to 5.",
						Optional:            true,
						Computed:            true,
					},
					"current_members": schema.Int64Attribute{
						Description:         "number of current members for this plan.",
						MarkdownDescription: "number of current members for this plan.",
						Computed:            true,
					},
				},
			},
		},
	}
}

type OrganizationModel struct {
	Canonical          types.String `tfsdk:"canonical"`
	Concourse          types.Object `tfsdk:"concourse"`
	HasChildren        types.Bool   `tfsdk:"has_children"`
	ID                 types.Int64  `tfsdk:"id"`
	IsRoot             types.Bool   `tfsdk:"is_root"`
	Licence            types.Object `tfsdk:"licence"`
	Name               types.String `tfsdk:"name"`
	ParentOrganization types.String `tfsdk:"parent_organization"`
	Subscription       types.Object `tfsdk:"subscription"`
}

type LicenceModel struct {
	CurrentMembers         types.Int64  `tfsdk:"current_members"`
	ExpiresAtRFC3339       types.String `tfsdk:"expires_at_rfc3339"`
	ExpiresAtUnixTimestamp types.Int64  `tfsdk:"expires_at_unix_timestamp"`
	IsOnPrem               types.Bool   `tfsdk:"is_on_prem"`
	Key                    types.String `tfsdk:"key"`
	MembersCount           types.Int64  `tfsdk:"members_count"`
}

var LicenceAttrTypes = map[string]attr.Type{
	"current_members":           types.Int64Type,
	"expires_at_rfc3339":        types.StringType,
	"expires_at_unix_timestamp": types.Int64Type,
	"is_on_prem":                types.BoolType,
	"key":                       types.StringType,
	"members_count":             types.Int64Type,
}

type ConcourseModel struct {
	TeamName types.String `tfsdk:"team_name"`
	Url      types.String `tfsdk:"url"`
	Port     types.String `tfsdk:"port"`
}

var ConcourseAttrTypes = map[string]attr.Type{
	"team_name": types.StringType,
	"url":       types.StringType,
	"port":      types.StringType,
}

type SubscriptionModel struct {
	CurrentMembers         types.Int64  `tfsdk:"current_members"`
	ExpiresAtRFC3339       types.String `tfsdk:"expires_at_rfc3339"`
	ExpiresAtUnixTimestamp types.Int64  `tfsdk:"expires_at_unix_timestamp"`
	MembersCount           types.Int64  `tfsdk:"members_count"`
	Plan                   types.String `tfsdk:"plan"`
}

var SubscriptionAttrTypes = map[string]attr.Type{
	"current_members":           types.Int64Type,
	"expires_at_rfc3339":        types.StringType,
	"expires_at_unix_timestamp": types.Int64Type,
	"plan":                      types.StringType,
	"members_count":             types.Int64Type,
}
