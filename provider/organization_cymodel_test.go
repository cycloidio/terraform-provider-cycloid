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
		Plan:           nil,                             // the crashing case
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
