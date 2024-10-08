// Code generated by terraform-plugin-framework-generator DO NOT EDIT.

package resource_catalog_repository

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func CatalogRepositoryResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"branch": schema.StringAttribute{
				Required:            true,
				Description:         "Branch needs to be valid git repository branch",
				MarkdownDescription: "Branch needs to be valid git repository branch",
			},
			"canonical": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The canonical of an entity",
				MarkdownDescription: "The canonical of an entity",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9]+[a-z0-9\\-_]+[a-z0-9]+$"), ""),
				},
			},
			"credential_canonical": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The canonical of an entity",
				MarkdownDescription: "The canonical of an entity",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9]+[a-z0-9\\-_]+[a-z0-9]+$"), ""),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "The name of an entity",
				MarkdownDescription: "The name of an entity",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"organization_canonical": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "A canonical of an organization.",
				MarkdownDescription: "A canonical of an organization.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9]+[a-z0-9\\-_]+[a-z0-9]+$"), ""),
				},
			},
			"owner": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "User canonical that owns this entity. If omitted then the person creating this\nentity will be assigned as owner. When a user is the owner of the entity he has\nall the permissions on it.\nIn case of API keys, the owner of API key is assigned as an owner. If \nAPI key has no owner, then no owner is set for entity as well.\n",
				MarkdownDescription: "User canonical that owns this entity. If omitted then the person creating this\nentity will be assigned as owner. When a user is the owner of the entity he has\nall the permissions on it.\nIn case of API keys, the owner of API key is assigned as an owner. If \nAPI key has no owner, then no owner is set for entity as well.\n",
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9]+[a-z0-9\\-_]+[a-z0-9]+$"), ""),
				},
			},
			"url": schema.StringAttribute{
				Required:            true,
				Description:         "GitURL represents all git URL formats we accept.\n",
				MarkdownDescription: "GitURL represents all git URL formats we accept.\n",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("^((/|~)[^/]*)+.(\\.git)|(([\\w\\]+@[\\w\\.]+))(:(//)?)([\\w\\.@\\:/\\-~]+)(/)?"), ""),
				},
			},
		},
	}
}

type CatalogRepositoryModel struct {
	Branch                types.String `tfsdk:"branch"`
	Canonical             types.String `tfsdk:"canonical"`
	CredentialCanonical   types.String `tfsdk:"credential_canonical"`
	Name                  types.String `tfsdk:"name"`
	OrganizationCanonical types.String `tfsdk:"organization_canonical"`
	Owner                 types.String `tfsdk:"owner"`
	Url                   types.String `tfsdk:"url"`
}
