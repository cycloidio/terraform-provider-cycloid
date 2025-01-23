package provider

import (
	"context"
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/resource_catalog_repository"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/sanity-io/litter"
	"github.com/stretchr/testify/assert"
)

// Return a pointer from a value
// To be able to build struct literrals with constants
func p[T any](val T) *T {
	return &val
}

func TestCyModelToData(t *testing.T) {
	testCases := []struct {
		Model *models.ServiceCatalogSource
		Data  *catalogRepositoryResourceModel
	}{
		{
			Model: &models.ServiceCatalogSource{
				Branch:              "branch",
				Canonical:           p("stack-canonical"),
				CredentialCanonical: "cred-canonical",
				ServiceCatalogs: []*models.ServiceCatalog{{
					Canonical: p("stack-canonical"),
				}},
				Owner: &models.User{
					Username: p("owner"),
				},
				URL:        p("osef"),
				Name:       p("stack-name"),
				StackCount: p(uint32(1)),
				ID:         p(uint32(1)),
			},

			Data: &catalogRepositoryResourceModel{
				Branch:                types.StringValue(""),
				Canonical:             types.StringValue(""),
				CredentialCanonical:   types.StringValue(""),
				Name:                  types.StringValue(""),
				OrganizationCanonical: types.StringValue(""),
				Owner:                 types.StringValue(""),
				Url:                   types.StringValue(""),
				Stacks:                types.ListNull(types.ObjectType{}),
			},
		},
	}

	for _, testCase := range testCases {
		diags := catalogRepositoryCYModelToData("fake-cycloid", testCase.Model, testCase.Data)
		if diags.HasError() {
			t.Fatal(diags)
		}

		assert.Equal(t, testCase.Model.Branch, testCase.Data.Branch.ValueString(), "branch must be equal")

		if testCase.Data.Stacks.IsNull() || testCase.Data.Stacks.IsUnknown() {
			t.Log("data is nill or unknown")
			litter.Dump(testCase.Data)
			t.FailNow()
		}

		var stackElements []resource_catalog_repository.Stack
		diags = testCase.Data.Stacks.ElementsAs(
			context.Background(),
			&stackElements,
			false,
		)
		if diags.HasError() {
			t.Fatal(diags)
		}

		assert.Equal(t, len(testCase.Model.ServiceCatalogs), len(stackElements), "the number of elements in %v must be equal to the number of input stacks", stackElements)

		assert.Equal(t, testCase.Model.Branch, testCase.Data.Branch.ValueString(), "branch must be equal")
		assert.Equal(t, *testCase.Model.Canonical, testCase.Data.Canonical.ValueString(), "canonical must be equal")
		assert.Equal(t, testCase.Model.CredentialCanonical, testCase.Data.CredentialCanonical.ValueString(), "credentialcanonical must be equal")
		assert.Equal(t, *testCase.Model.Name, testCase.Data.Name.ValueString(), "name must be equal")
		assert.Equal(t, *testCase.Model.Owner.Username, testCase.Data.Owner.ValueString(), "owner must be equal")
		assert.Equal(t, *testCase.Model.URL, testCase.Data.Url.ValueString(), "url must be equal")

		for index, stack := range testCase.Model.ServiceCatalogs {
			tfStack := stackElements[index]
			assert.Equal(t, *stack.Canonical, tfStack.Canonical.ValueString(), "branch must be equal")
		}
	}
}
