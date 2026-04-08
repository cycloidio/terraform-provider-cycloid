package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/cycloidio/cycloid-cli/client/models"
	middleware "github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/cycloidio/terraform-provider-cycloid/resource_organization"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &organizationResource{}

// In case we need to implement state migration
// var _ resource.ResourceWithUpgradeState = &organizationResource{}

func NewOrganizationResource() resource.Resource {
	return &organizationResource{}
}

type organizationResource struct {
	provider *CycloidProvider
}

type organizationResourceModel resource_organization.OrganizationModel
type licenceResourceModel resource_organization.LicenceModel
type subscriptionResourceModel resource_organization.SubscriptionModel

func (r *organizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *organizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_organization.OrganizationResourceSchema(ctx)
}

func (r *organizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	pv, ok := req.ProviderData.(*CycloidProvider)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}

	r.provider = pv
}

func (r *organizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var orgState organizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &orgState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Middleware
	name, canonical, err := NameOrCanonical(orgState.Name.ValueString(), orgState.Canonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("canonical"),
			fmt.Sprintf("failed to infer canonical, from name %q and canonical %q", name, canonical),
			"Fill either `name` or `canonical` attribute. subsequent error: "+err.Error(),
		)
		return
	}

	// Check if the org exists, if so, put a valid error
	var org *models.Organization
	parentOrg := orgState.ParentOrganization.ValueString()
	orgs, _, err := m.ListOrganizationChildrens(Coalesce(parentOrg, canonical))
	if err != nil {
		resp.Diagnostics.AddError(
			"fail to read current organizations",
			err.Error(),
		)
		return
	}

	for _, o := range orgs {
		if ptr.Value(o.Canonical) == canonical {
			resp.Diagnostics.AddError(
				fmt.Sprintf("An organization named %q with canonical %q already exists", ptr.Value(o.Name), ptr.Value(o.Canonical)),
				"You should either choose a different name or import the existing one in the state.",
			)
			return
		}
	}

	if parentOrg != "" {
		org, _, err = m.CreateOrganizationChild(parentOrg, canonical, &name)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to create organization",
				err.Error(),
			)
			return
		}
	} else {
		org, _, err = m.CreateOrganization(name)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to create organization",
				err.Error(),
			)
			return
		}
	}

	// Manage licence
	var licenceState licenceResourceModel
	if !orgState.Licence.IsUnknown() && !orgState.Licence.IsNull() {
		if diags := orgState.Licence.As(ctx, &licenceState, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		key := licenceState.Key.ValueString()
		if key == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("licence.key"),
				"when `licence.apply_licence` is true, the key must be filled",
				"The licence key is empty, check your configuration.",
			)
			return
		}

		_, err := m.ActivateLicence(canonical, key)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to update licence for org %q", canonical),
				err.Error(),
			)
			return
		}
	}

	var licence *models.Licence
	_, err = m.GenericRequest(middleware.Request{
		Method:       "GET",
		Organization: &canonical,
		Route:        []string{"organizations", canonical, "licence"},
	}, licence)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch licence for org \""+canonical+"\"", err.Error())
		return
	}

	// Manage subscription
	var subscription *models.Subscription
	var subscriptionState subscriptionResourceModel
	if !orgState.Subscription.IsUnknown() && !orgState.Subscription.IsNull() {
		if diags := orgState.Subscription.As(ctx, &subscriptionState, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		timestampValue := subscriptionState.ExpiresAtRFC3339.ValueString()
		t, err := time.Parse(time.RFC3339, timestampValue)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("subscription.expires_at_rfc3339"),
				fmt.Sprintf("failed to parse timestamp %q", timestampValue),
				err.Error(),
			)
			return
		}

		// Middleware doesn't send back the sub
		_, _, err = m.CreateOrUpdateSubscription(
			canonical, subscriptionState.Plan.ValueString(), t,
			uint64(subscriptionState.MembersCount.ValueInt64()), true,
		)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to update subscription for org %q", canonical),
				err.Error(),
			)
			return
		}

		// We fix the be not returning the sub by fixing the values ourselves
		// TODO: after CLI fix, fix this
		subscription = &models.Subscription{
			CurrentMembers: ptr.Ptr(uint64(0)),
			MembersCount:   ptr.Ptr(uint64(subscriptionState.MembersCount.ValueInt64())),
			ExpiresAt:      ptr.Ptr(uint64(t.UnixMilli())),
			Plan: &models.SubscriptionPlan{
				Canonical: subscriptionState.Plan.ValueStringPointer(),
			},
		}
	}

	resp.Diagnostics.Append(
		organizationCYModelToData(
			ctx, &orgState, &licenceState, &subscriptionState,
			*org, orgState.ParentOrganization.ValueStringPointer(), licence, subscription,
		)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &orgState)...)
}

func (r *organizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var orgState organizationResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &orgState)...)

	var licenceState licenceResourceModel
	if !orgState.Licence.IsNull() && !orgState.Licence.IsUnknown() {
		if diags := orgState.Licence.As(ctx, &licenceState, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	var subscriptionState subscriptionResourceModel
	if !orgState.Subscription.IsNull() && !orgState.Subscription.IsUnknown() {
		if diags := orgState.Subscription.As(ctx, &subscriptionState, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	m := r.provider.Middleware

	var err error
	_, canonical, err := NameOrCanonical(orgState.Name.ValueString(), orgState.Canonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("canonical"), "failed to get current org canonical", err.Error(),
		)
	}

	orgs, _, err := m.ListOrganizationChildrens(Coalesce(orgState.ParentOrganization.ValueString(), canonical))
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to read org %q from API", canonical), err.Error())
		return
	}

	var org *models.Organization
	for _, o := range orgs {
		if ptr.Value(o.Canonical) == canonical {
			org = o
			break
		}
	}

	if org == nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Organization %q not found", canonical), "")
		return
	}

	var licence *models.Licence
	_, err = m.GenericRequest(middleware.Request{
		Method:       "GET",
		Organization: &canonical,
		Route:        []string{"organizations", canonical, "licence"},
	}, licence)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch licence for org \""+canonical+"\"", err.Error())
		return
	}

	resp.Diagnostics.Append(
		organizationCYModelToData(
			ctx, &orgState, &licenceState, &subscriptionState,
			*org, orgState.ParentOrganization.ValueStringPointer(), licence, org.Subscription,
		)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &orgState)...)
}

func (r *organizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var orgPlan organizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &orgPlan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var orgState organizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &orgState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := r.provider.Middleware
	var name, canonical string
	var err error
	if orgState.Canonical.IsNull() || orgState.Canonical.IsUnknown() {
		name, canonical, err = NameOrCanonical(orgPlan.Name.ValueString(), orgPlan.Canonical.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("canonical"),
				fmt.Sprintf("failed to infer canonical, from name %q and canonical %q", name, canonical),
				"Fill either `name` or `canonical` attribute. subsequent error: "+err.Error(),
			)
			return
		}
	} else {
		name, canonical = Coalesce(orgPlan.Name.ValueString(), orgState.Name.ValueString()), orgState.Canonical.ValueString()
	}

	parentOrg := orgPlan.ParentOrganization.ValueString()
	orgs, _, err := m.ListOrganizationChildrens(Coalesce(parentOrg, canonical))
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update organization "+canonical,
			err.Error(),
		)
		return
	}

	var currentOrg *models.Organization
	for _, o := range orgs {
		if ptr.Value(o.Canonical) == canonical {
			currentOrg = o
		}
	}

	var org *models.Organization
	if currentOrg == nil && parentOrg != "" {
		org, _, err = m.CreateOrganizationChild(parentOrg, canonical, &name)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to create org %s", canonical),
				err.Error(),
			)
			return
		}
	} else if currentOrg == nil && parentOrg == "" {
		org, _, err = m.CreateOrganization(name)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to create org %s", canonical),
				err.Error(),
			)
			return
		}
	} else {
		org, _, err = m.UpdateOrganization(canonical, name)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to update org %s", canonical),
				err.Error(),
			)
			return
		}
	}

	// Manage licence
	var licence *models.Licence
	var licenceState licenceResourceModel
	if !orgPlan.Licence.IsNull() && !orgPlan.Licence.IsUnknown() {
		if diags := orgPlan.Licence.As(ctx, &licenceState, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		key := licenceState.Key.ValueString()
		if key == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("licence.key"),
				"when `licence.apply_licence` is true, the key must be filled",
				"The licence key is empty, check your configuration.",
			)
			return
		}

		_, err := m.ActivateLicence(canonical, key)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to update licence for org %q", canonical),
				err.Error(),
			)
			return
		}
	}

	_, err = m.GenericRequest(middleware.Request{
		Method:       "GET",
		Organization: &canonical,
		Route:        []string{"organizations", canonical, "licence"},
	}, licence)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch licence for org \""+canonical+"\"", err.Error())
		return
	}

	// Manage subscription
	var subscription *models.Subscription
	var subscriptionState subscriptionResourceModel
	if !orgPlan.Subscription.IsUnknown() && !orgPlan.Subscription.IsNull() {
		if diags := orgPlan.Subscription.As(ctx, &subscriptionState, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		timestampValue := subscriptionState.ExpiresAtRFC3339.ValueString()
		t, err := time.Parse(time.RFC3339, timestampValue)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("subscription.expires_at_rfc3339"),
				fmt.Sprintf("failed to parse timestamp %q", timestampValue),
				err.Error(),
			)
			return
		}

		var body = map[string]any{
			"expires_at":    t.UTC().Format(time.RFC3339),
			"members_count": subscriptionState.MembersCount.ValueInt64(),
			"overwrite":     true,
		}
		r, err := m.GenericRequest(middleware.Request{
			Method:       "PUT",
			Organization: org.Canonical,
			Route:        []string{"organizations", canonical, "subscriptions"},
			Body:         body,
		}, subscription)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to update subscription for org %q", canonical),
				err.Error(),
			)
			return
		}

		if r.StatusCode != 200 {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to update subscription for org %q", canonical),
				fmt.Sprintf("Error code: %q", r.Status),
			)
			return
		}

		// We fix the BE not returning the sub by fixing the values ourselves
		subscription = &models.Subscription{
			CurrentMembers: ptr.Ptr(uint64(0)),
			MembersCount:   ptr.Ptr(uint64(subscriptionState.MembersCount.ValueInt64())),
			ExpiresAt:      ptr.Ptr(uint64(t.UnixMilli())),
			Plan: &models.SubscriptionPlan{
				Canonical: subscriptionState.Plan.ValueStringPointer(),
			},
		}
	}

	resp.Diagnostics.Append(
		organizationCYModelToData(
			ctx, &orgPlan, &licenceState, &subscriptionState,
			*org, orgPlan.ParentOrganization.ValueStringPointer(), licence, subscription,
		)...,
	)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &orgPlan)...)
}

func (r *organizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var orgState organizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &orgState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if soft destroy is enabled
	if orgState.SoftDestroy.ValueBool() {
		// Soft destroy: just remove from state, don't delete from Cycloid
		resp.Diagnostics.AddWarning(
			"Soft destroy performed",
			fmt.Sprintf("Organization %q has been removed from Terraform state but still exists in Cycloid. You can now manage it manually through the UI or API.", orgState.Canonical.ValueString()),
		)
		return
	}

	// Check if destruction is allowed
	if !orgState.AllowDestroy.ValueBool() {
		resp.Diagnostics.AddError(
			"Organization destruction blocked",
			"allow_destroy is set to false. Set allow_destroy to true and apply again to destroy this organization. This prevents accidental deletion of organizations containing projects, environments, and components.",
		)
		return
	}

	// Ensure canonical exists in state
	if orgState.Canonical.IsNull() || orgState.Canonical.IsUnknown() {
		resp.Diagnostics.AddError(
			"Invalid organization state",
			"Organization canonical is not available in state. This indicates an inconsistent state that requires manual intervention.",
		)
		return
	}

	m := r.provider.Middleware
	_, err := m.DeleteOrganization(orgState.Canonical.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to delete org "+orgState.Canonical.ValueString(), err.Error())
		return
	}

	var licenceState licenceResourceModel
	var subscriptionState subscriptionResourceModel
	resp.Diagnostics.Append(
		organizationCYModelToData(
			ctx, &orgState, &licenceState, &subscriptionState, models.Organization{}, nil, nil, nil,
		)...,
	)
}

func organizationCYModelToData(ctx context.Context, orgState *organizationResourceModel, licenceState *licenceResourceModel, subscriptionState *subscriptionResourceModel, org models.Organization, parentOrg *string, licence *models.Licence, subscription *models.Subscription) diag.Diagnostics {
	var diags diag.Diagnostics
	// Store the protection-related fields before modifying orgState
	preserveAllowDestroy := orgState.AllowDestroy
	preserveSoftDestroy := orgState.SoftDestroy
	// var licenceState basetypes.ObjectValue
	var licenceValue basetypes.ObjectValue
	if licence == nil {
		licenceValue = types.ObjectNull(
			resource_organization.LicenceAttrTypes,
		)
		if diags.HasError() {
			return diags
		}
	} else {
		expiresAt := time.UnixMilli(int64(ptr.Value(licence.ExpiresAt)))
		licenceValue, diags = types.ObjectValue(
			resource_organization.LicenceAttrTypes,
			map[string]attr.Value{
				"current_members":           types.Int64Value(int64(ptr.Value(licence.CurrentMembers))),
				"expires_at_rfc3339":        types.StringValue(expiresAt.UTC().Format("2006-01-02T15:04:05Z")),
				"expires_at_unix_timestamp": types.Int64Value(expiresAt.UnixMilli()),
				"is_on_prem":                types.BoolPointerValue(licence.OnPrem),
				"key":                       types.StringPointerValue(licence.Key),
				"members_count":             types.Int64Value(int64(ptr.Value(licence.MembersCount))),
			},
		)
		if diags.HasError() {
			return diags
		}
	}

	var subscriptionValue basetypes.ObjectValue
	if subscription == nil {
		subscriptionValue = types.ObjectNull(resource_organization.SubscriptionAttrTypes)
	} else {
		expiresAt := time.UnixMilli(int64(ptr.Value(subscription.ExpiresAt)))
		subscriptionValue, diags = types.ObjectValue(
			resource_organization.SubscriptionAttrTypes,
			map[string]attr.Value{
				"current_members":           types.Int64Value(int64(ptr.Value(subscription.CurrentMembers))),
				"expires_at_rfc3339":        types.StringValue(expiresAt.UTC().Format("2006-01-02T15:04:05Z")),
				"expires_at_unix_timestamp": types.Int64Value(expiresAt.UnixMilli()),
				"plan":                      types.StringPointerValue(subscription.Plan.Canonical),
				"members_count":             types.Int64Value(int64(ptr.Value(subscription.MembersCount))),
			},
		)
		if diags.HasError() {
			return diags
		}
	}

	var concourseState basetypes.ObjectValue
	concourseValues := make(map[string]attr.Value)
	concourseValues["team_name"] = types.StringPointerValue(org.CiTeamName)
	concourseValues["url"] = types.StringPointerValue(org.CiURL)
	concourseValues["port"] = types.StringPointerValue(org.CiPort)
	concourseState, diags = types.ObjectValue(
		resource_organization.ConcourseAttrTypes,
		concourseValues,
	)
	if diags.HasError() {
		return diags
	}

	orgState.Canonical = types.StringPointerValue(org.Canonical)
	orgState.Concourse = concourseState
	orgState.HasChildren = types.BoolPointerValue(org.HasChildren)
	orgState.ID = types.Int64Value(int64(ptr.Value(org.ID)))
	orgState.IsRoot = types.BoolPointerValue(org.IsRoot)
	orgState.Licence = licenceValue
	orgState.Name = types.StringPointerValue(org.Name)
	orgState.ParentOrganization = types.StringPointerValue(parentOrg)
	orgState.Subscription = subscriptionValue
	// Restore the protection-related fields from the input state
	orgState.AllowDestroy = preserveAllowDestroy
	orgState.SoftDestroy = preserveSoftDestroy
	return nil
}

// Keeping this for now, this is the boilerplate for upgrading the state of existing
// resources, since the old one wasn't working very well, there may be no need to implement it.
// TODO: check at next release.
// func (r *organizationResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
// 	return map[int64]resource.StateUpgrader{
// 		0: {
// 			PriorSchema: &schema.Schema{},
// 			// Optionally, the PriorSchema field can be defined.
// 			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
// 			},
// 		},
// 	}
// }
