package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

func TestCyModelToData(t *testing.T) {
	testCases := []struct {
		Model         *models.ServiceCatalogSource
		Data          *catalogRepositoryResourceModel
		ExpectedOwner string
	}{
		{
			Model: &models.ServiceCatalogSource{
				Branch:              "branch",
				Canonical:           ptr.Ptr("stack-canonical"),
				CredentialCanonical: "cred-canonical",
				ServiceCatalogs: []*models.ServiceCatalog{{
					Canonical: ptr.Ptr("stack-canonical"),
				}},
				Owner: &models.User{
					Username: ptr.Ptr("owner"),
				},
				URL:        ptr.Ptr("osef"),
				Name:       ptr.Ptr("stack-name"),
				StackCount: ptr.Ptr(uint32(1)),
				ID:         ptr.Ptr(uint32(1)),
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
			ExpectedOwner: "owner",
		},
		{
			Model: &models.ServiceCatalogSource{
				Branch:              "branch",
				Canonical:           ptr.Ptr("stack-canonical"),
				CredentialCanonical: "cred-canonical",
				ServiceCatalogs: []*models.ServiceCatalog{{
					Canonical: ptr.Ptr("stack-canonical"),
				}},
				Owner:      nil,
				URL:        ptr.Ptr("osef"),
				Name:       ptr.Ptr("stack-name"),
				StackCount: ptr.Ptr(uint32(1)),
				ID:         ptr.Ptr(uint32(1)),
			},
			Data: &catalogRepositoryResourceModel{
				Branch:                types.StringValue(""),
				Canonical:             types.StringValue(""),
				CredentialCanonical:   types.StringValue(""),
				Name:                  types.StringValue(""),
				OrganizationCanonical: types.StringValue(""),
				Owner:                 types.StringUnknown(),
				Url:                   types.StringValue(""),
			},
			ExpectedOwner: "",
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
		assert.Equal(t, testCase.ExpectedOwner, testCase.Data.Owner.ValueString(), "owner must be equal")
		assert.Equal(t, *testCase.Model.URL, testCase.Data.Url.ValueString(), "url must be equal")
	}
}

func TestAccCatalogRepositoryResource(t *testing.T) {
	t.Parallel()

	repoName := RandomCanonical("test-catalog-repo")
	ctx := context.Background()
	orgCanonical := testAccGetOrganizationCanonical()
	cfg := testAccGetTestConfig(t)
	if cfg.Repositories.Catalog.Credential == "" {
		t.Skip("repositories.catalog.credential must be set in test_config.yaml for this test")
	}
	repoURL := cfg.Repositories.Catalog.URL
	repoBranch := cfg.Repositories.Catalog.Branch
	credCanonical := cfg.Repositories.Catalog.Credential
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create catalog repository with organization_canonical parameter
			{
				Config: testAccCatalogRepositoryConfig_basic(orgCanonical, repoName, repoURL, repoBranch, credCanonical),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_catalog_repository.test", "organization_canonical", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_catalog_repository.test", "name", repoName),
					resource.TestCheckResourceAttr("cycloid_catalog_repository.test", "url", repoURL),
					resource.TestCheckResourceAttr("cycloid_catalog_repository.test", "branch", repoBranch),
					resource.TestCheckResourceAttr("cycloid_catalog_repository.test", "credential_canonical", credCanonical),
				),
			},
			// Destroy testing
			{
				Config:  " ", // Empty config to trigger destroy
				Destroy: true,
			},
		},
	})
}

// Test configuration functions (credCanonical from test config)
func testAccCatalogRepositoryConfig_basic(org, name, url, branch, credCanonical string) string {
	return fmt.Sprintf(`
resource "cycloid_catalog_repository" "test" {
  name                   = "%s"
  credential_canonical   = "%s"
  url                    = "%s"
  branch                 = "%s"
  organization_canonical = "%s"
}
`, name, credCanonical, url, branch, org)
}

func TestConfiguredCatalogRepositoryOwner(t *testing.T) {
	testCases := []struct {
		Name          string
		Owner         types.String
		ExpectedValue string
		ExpectedSet   bool
	}{
		{
			Name:          "known owner",
			Owner:         types.StringValue("alice"),
			ExpectedValue: "alice",
			ExpectedSet:   true,
		},
		{
			Name:          "empty owner",
			Owner:         types.StringValue(""),
			ExpectedValue: "",
			ExpectedSet:   false,
		},
		{
			Name:          "null owner",
			Owner:         types.StringNull(),
			ExpectedValue: "",
			ExpectedSet:   false,
		},
		{
			Name:          "unknown owner",
			Owner:         types.StringUnknown(),
			ExpectedValue: "",
			ExpectedSet:   false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			owner, set := configuredCatalogRepositoryOwner(testCase.Owner)
			assert.Equal(t, testCase.ExpectedValue, owner)
			assert.Equal(t, testCase.ExpectedSet, set)
		})
	}
}
