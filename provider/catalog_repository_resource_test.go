package provider

import (
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
			},
		},
	}

	for _, testCase := range testCases {
		diags := catalogRepositoryCYModelToData("fake-cycloid", testCase.Model, testCase.Data)
		if diags.HasError() {
			t.Fatal(diags)
		}

		assert.Equal(t, testCase.Model.Branch, testCase.Data.Branch.ValueString(), "branch must be equal")
		assert.Equal(t, testCase.Model.Branch, testCase.Data.Branch.ValueString(), "branch must be equal")
		assert.Equal(t, *testCase.Model.Canonical, testCase.Data.Canonical.ValueString(), "canonical must be equal")
		assert.Equal(t, testCase.Model.CredentialCanonical, testCase.Data.CredentialCanonical.ValueString(), "credentialcanonical must be equal")
		assert.Equal(t, *testCase.Model.Name, testCase.Data.Name.ValueString(), "name must be equal")
		assert.Equal(t, *testCase.Model.Owner.Username, testCase.Data.Owner.ValueString(), "owner must be equal")
		assert.Equal(t, *testCase.Model.URL, testCase.Data.Url.ValueString(), "url must be equal")
	}
}
