package provider

import (
	"context"
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDataStacksToListValue_NilPointerFields verifies that dataStacksToListValue
// does not panic when pointer fields in the ServiceCatalog model are nil.
// Regression test for the direct * dereferences that existed before ptr.Value.
func TestDataStacksToListValue_NilPointerFields(t *testing.T) {
	ctx := context.Background()

	stacks := []*models.ServiceCatalog{
		{
			// All pointer fields left nil — only required value types set
			Blueprint:   false,
			Description: "",
			Keywords:    nil,
		},
	}

	_, diags := dataStacksToListValue(ctx, stacks)
	assert.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
}

// TestDataStacksToListValue_NilTeam verifies that a stack with a nil Team field
// does not panic and produces an empty team canonical.
func TestDataStacksToListValue_NilTeam(t *testing.T) {
	ctx := context.Background()

	name := "my-stack"
	canonical := "my-stack"
	author := "alice"
	dir := "stacks/my-stack"
	formEnabled := true
	quotaEnabled := false
	ref := "org:my-stack"
	trusted := true
	visibility := "public"
	orgCanonical := "my-org"

	stacks := []*models.ServiceCatalog{
		{
			Name:                  &name,
			Canonical:             &canonical,
			Author:                &author,
			Directory:             &dir,
			FormEnabled:           &formEnabled,
			QuotaEnabled:          &quotaEnabled,
			Ref:                   &ref,
			Trusted:               &trusted,
			Visibility:            &visibility,
			OrganizationCanonical: &orgCanonical,
			Team:                  nil, // explicitly nil
		},
	}

	result, diags := dataStacksToListValue(ctx, stacks)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	assert.False(t, result.IsNull())
	assert.Equal(t, 1, len(result.Elements()))
}

// TestDataStacksToListValue_NilTeamCanonical verifies that a stack whose Team
// exists but has a nil Canonical does not panic.
func TestDataStacksToListValue_NilTeamCanonical(t *testing.T) {
	ctx := context.Background()

	stacks := []*models.ServiceCatalog{
		{
			Team: &models.SimpleTeam{Canonical: nil},
		},
	}

	_, diags := dataStacksToListValue(ctx, stacks)
	assert.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
}

// TestDataStacksToListValue_NilCloudProviderFields verifies that nil Canonical
// and Name fields on a CloudProvider do not cause a panic.
func TestDataStacksToListValue_NilCloudProviderFields(t *testing.T) {
	ctx := context.Background()

	stacks := []*models.ServiceCatalog{
		{
			CloudProviders: []*models.CloudProvider{
				{
					Abbreviation: "AWS",
					Canonical:    nil,
					Name:         nil,
					Regions:      nil,
				},
			},
		},
	}

	_, diags := dataStacksToListValue(ctx, stacks)
	assert.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
}

// TestDataStacksToListValue_EmptyList verifies that an empty input produces an
// empty (non-null) list rather than panicking or erroring.
func TestDataStacksToListValue_EmptyList(t *testing.T) {
	ctx := context.Background()

	result, diags := dataStacksToListValue(ctx, nil)
	assert.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	assert.Equal(t, 0, len(result.Elements()))
}

// TestDataStacksToListValue_CloudProviderWithPtr exercises the ptr.Value path
// for canonical and name fields to ensure the happy path still works.
func TestDataStacksToListValue_CloudProviderWithPtr(t *testing.T) {
	ctx := context.Background()

	canonical := "aws"
	name := "Amazon Web Services"

	stacks := []*models.ServiceCatalog{
		{
			CloudProviders: []*models.CloudProvider{
				{
					Abbreviation: "AWS",
					Canonical:    &canonical,
					Name:         &name,
				},
			},
		},
	}

	_, diags := dataStacksToListValue(ctx, stacks)
	assert.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
}
