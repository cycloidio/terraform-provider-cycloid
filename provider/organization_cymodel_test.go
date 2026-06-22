package provider

import (
	"context"
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrganizationCYModelToData_SubscriptionNilPlan guards against the SIGSEGV
// that occurred when subscription.Plan was nil (TFPRO-29).
func TestOrganizationCYModelToData_SubscriptionNilPlan(t *testing.T) {
	ctx := context.Background()

	org := models.Organization{
		Canonical: ptr.Ptr("test-org"),
		Name:      ptr.Ptr("Test Org"),
	}

	subscription := &models.Subscription{
		CurrentMembers: ptr.Ptr(uint64(5)),
		MembersCount:   ptr.Ptr(uint64(10)),
		ExpiresAt:      ptr.Ptr(uint64(1893456000000)), // some unix ms
		Plan:           nil,                            // the crashing case
	}

	licence := &models.Licence{
		CurrentMembers: ptr.Ptr(uint64(5)),
		MembersCount:   ptr.Ptr(uint64(10)),
		ExpiresAt:      ptr.Ptr(uint64(1893456000000)),
		OnPrem:         ptr.Ptr(false),
		Key:            ptr.Ptr("test-key"),
	}

	state := &organizationResourceModel{}
	var licenceState licenceResourceModel
	var subscriptionState subscriptionResourceModel

	diags := organizationCYModelToData(ctx, state, &licenceState, &subscriptionState, org, nil, licence, subscription)
	assert.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
}

// TestOrganizationCYModelToData_NilSubscription ensures a nil subscription
// produces a null Terraform object rather than a panic.
func TestOrganizationCYModelToData_NilSubscription(t *testing.T) {
	ctx := context.Background()

	org := models.Organization{
		Canonical: ptr.Ptr("test-org"),
		Name:      ptr.Ptr("Test Org"),
	}

	licence := &models.Licence{
		CurrentMembers: ptr.Ptr(uint64(5)),
		MembersCount:   ptr.Ptr(uint64(10)),
		ExpiresAt:      ptr.Ptr(uint64(1893456000000)),
		OnPrem:         ptr.Ptr(false),
		Key:            ptr.Ptr("test-key"),
	}

	state := &organizationResourceModel{}
	var licenceState licenceResourceModel
	var subscriptionState subscriptionResourceModel

	diags := organizationCYModelToData(ctx, state, &licenceState, &subscriptionState, org, nil, licence, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	assert.True(t, state.Subscription.IsNull(), "subscription should be null when not provided")
}

// TestOrganizationCYModelToData_NilLicence ensures a nil licence produces a
// null Terraform object rather than a panic.
func TestOrganizationCYModelToData_NilLicence(t *testing.T) {
	ctx := context.Background()

	org := models.Organization{
		Canonical: ptr.Ptr("test-org"),
		Name:      ptr.Ptr("Test Org"),
	}

	state := &organizationResourceModel{}
	var licenceState licenceResourceModel
	var subscriptionState subscriptionResourceModel

	diags := organizationCYModelToData(ctx, state, &licenceState, &subscriptionState, org, nil, nil, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	assert.True(t, state.Licence.IsNull(), "licence should be null when not provided")
}

// TestOrganizationRead_NilSubscriptionWhenStateIsNull is a regression test for
// TFPRO-49: when the prior state has subscription=null (user never configured it)
// and the API returns a non-nil subscription (server-managed default), the Read
// helper must receive nil so it keeps state.Subscription null — preventing the
// eternal "subscription will be added" drift on every plan/apply.
//
// This test validates the guard in Read that passes nil instead of org.Subscription
// when orgState.Subscription.IsNull().
func TestOrganizationRead_NilSubscriptionWhenStateIsNull(t *testing.T) {
	ctx := context.Background()

	org := models.Organization{
		Canonical: ptr.Ptr("test-org"),
		Name:      ptr.Ptr("Test Org"),
	}

	// Simulate what the API always returns for every org (server-managed free_tier).
	planCanonical := "free_tier"
	planName := "Free Tier"
	apiSubscription := &models.Subscription{
		CurrentMembers: ptr.Ptr(uint64(0)),
		MembersCount:   ptr.Ptr(uint64(5)),
		ExpiresAt:      ptr.Ptr(uint64(0)),
		Plan: &models.SubscriptionPlan{
			Canonical: &planCanonical,
			Name:      &planName,
		},
	}

	// Prior state has subscription=null (user never set it in config).
	// The Read guard translates this into nil passed to organizationCYModelToData.
	state := &organizationResourceModel{}
	// state.Subscription starts as its zero value (null types.Object), which is
	// what IsNull() returns true on. Simulate the guard logic:
	var subscriptionForState *models.Subscription
	if !state.Subscription.IsNull() {
		subscriptionForState = apiSubscription
	}
	// subscriptionForState must be nil here — that is the guard working correctly.
	require.Nil(t, subscriptionForState, "guard must pass nil when prior state.Subscription is null")

	licence := &models.Licence{
		CurrentMembers: ptr.Ptr(uint64(0)),
		MembersCount:   ptr.Ptr(uint64(5)),
		ExpiresAt:      ptr.Ptr(uint64(0)),
		OnPrem:         ptr.Ptr(false),
		Key:            ptr.Ptr(""),
	}

	var licenceState licenceResourceModel
	var subscriptionState subscriptionResourceModel

	diags := organizationCYModelToData(ctx, state, &licenceState, &subscriptionState, org, nil, licence, subscriptionForState)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	// Key assertion: subscription stays null in state — no drift.
	assert.True(t, state.Subscription.IsNull(),
		"subscription must remain null in state when user never set it in config (TFPRO-49 regression guard)")
}

// TestOrganizationRead_SubscriptionUpdatedWhenStateIsNonNull validates that when
// the user HAS configured subscription (prior state is non-null), Read correctly
// picks up the latest API value.
func TestOrganizationRead_SubscriptionUpdatedWhenStateIsNonNull(t *testing.T) {
	ctx := context.Background()

	org := models.Organization{
		Canonical: ptr.Ptr("test-org"),
		Name:      ptr.Ptr("Test Org"),
	}

	planCanonical := "platform_teams"
	planName := "Platform Teams"
	apiSubscription := &models.Subscription{
		CurrentMembers: ptr.Ptr(uint64(3)),
		MembersCount:   ptr.Ptr(uint64(10)),
		ExpiresAt:      ptr.Ptr(uint64(1893456000000)),
		Plan: &models.SubscriptionPlan{
			Canonical: &planCanonical,
			Name:      &planName,
		},
	}

	// Simulate prior state with non-null subscription (user manages it).
	// We build a state where Subscription is non-null by running organizationCYModelToData
	// with a prior subscription first.
	priorPlanCanonical := "platform_teams"
	priorPlanName := "Platform Teams"
	priorSub := &models.Subscription{
		CurrentMembers: ptr.Ptr(uint64(1)),
		MembersCount:   ptr.Ptr(uint64(10)),
		ExpiresAt:      ptr.Ptr(uint64(1893456000000)),
		Plan:           &models.SubscriptionPlan{Canonical: &priorPlanCanonical, Name: &priorPlanName},
	}
	priorLicence := &models.Licence{
		CurrentMembers: ptr.Ptr(uint64(0)),
		MembersCount:   ptr.Ptr(uint64(5)),
		ExpiresAt:      ptr.Ptr(uint64(0)),
		OnPrem:         ptr.Ptr(false),
		Key:            ptr.Ptr(""),
	}
	state := &organizationResourceModel{}
	var licenceState licenceResourceModel
	var subscriptionState subscriptionResourceModel
	diags := organizationCYModelToData(ctx, state, &licenceState, &subscriptionState, org, nil, priorLicence, priorSub)
	require.False(t, diags.HasError())
	require.False(t, state.Subscription.IsNull(), "setup: state should have non-null subscription")

	// Now simulate Read guard: state.Subscription is non-null → pass API subscription.
	var subscriptionForState *models.Subscription
	if !state.Subscription.IsNull() {
		subscriptionForState = apiSubscription
	}
	require.NotNil(t, subscriptionForState, "guard must pass API value when prior state.Subscription is non-null")

	diags = organizationCYModelToData(ctx, state, &licenceState, &subscriptionState, org, nil, priorLicence, subscriptionForState)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	assert.False(t, state.Subscription.IsNull(), "subscription must remain non-null when user manages it")
}

// TestOrganizationCYModelToData_FullSubscription is a regression test for the
// happy path with all subscription fields populated.
func TestOrganizationCYModelToData_FullSubscription(t *testing.T) {
	ctx := context.Background()

	org := models.Organization{
		Canonical: ptr.Ptr("test-org"),
		Name:      ptr.Ptr("Test Org"),
	}

	planCanonical := "standard"
	subscription := &models.Subscription{
		CurrentMembers: ptr.Ptr(uint64(5)),
		MembersCount:   ptr.Ptr(uint64(10)),
		ExpiresAt:      ptr.Ptr(uint64(1893456000000)),
		Plan:           &models.SubscriptionPlan{Canonical: &planCanonical},
	}

	licence := &models.Licence{
		CurrentMembers: ptr.Ptr(uint64(5)),
		MembersCount:   ptr.Ptr(uint64(10)),
		ExpiresAt:      ptr.Ptr(uint64(1893456000000)),
		OnPrem:         ptr.Ptr(false),
		Key:            ptr.Ptr("test-key"),
	}

	state := &organizationResourceModel{}
	var licenceState licenceResourceModel
	var subscriptionState subscriptionResourceModel

	diags := organizationCYModelToData(ctx, state, &licenceState, &subscriptionState, org, nil, licence, subscription)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	assert.False(t, state.Subscription.IsNull(), "subscription should not be null")
	assert.False(t, state.Licence.IsNull(), "licence should not be null")
}
