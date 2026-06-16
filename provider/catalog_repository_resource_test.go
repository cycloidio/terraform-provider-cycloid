package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/common"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
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
	repoURL := cfg.Repositories.Catalog.URL
	repoBranch := cfg.Repositories.Catalog.Branch
	credCanonical := cfg.Repositories.Catalog.Credential
	if repoURL == "" {
		t.Skip("repositories.catalog.url must be set for this test")
	}
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr("cycloid_catalog_repository.test", "organization_canonical", orgCanonical),
		resource.TestCheckResourceAttr("cycloid_catalog_repository.test", "name", repoName),
		resource.TestCheckResourceAttr("cycloid_catalog_repository.test", "url", repoURL),
		resource.TestCheckResourceAttr("cycloid_catalog_repository.test", "branch", repoBranch),
	}
	if credCanonical != "" {
		checks = append(checks, resource.TestCheckResourceAttr("cycloid_catalog_repository.test", "credential_canonical", credCanonical))
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create catalog repository — credential omitted for public repos.
			{
				Config: testAccCatalogRepositoryConfig_basic(orgCanonical, repoName, repoURL, repoBranch, credCanonical),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
			// Destroy testing
			{
				Config:  " ", // Empty config to trigger destroy
				Destroy: true,
			},
		},
	})
}

// TestAccCatalogRepositoryResource_RefreshOnCreate verifies that setting refresh_on_create = true
// causes the versions/refresh endpoint to be called after create, making branches immediately
// resolvable. The test creates a catalog repo with refresh_on_create = true and confirms apply
// succeeds without error (the refresh endpoint is called synchronously as part of Create).
func TestAccCatalogRepositoryResource_RefreshOnCreate(t *testing.T) {
	t.Parallel()

	repoName := RandomCanonical("test-catalog-refresh")
	ctx := context.Background()
	orgCanonical := testAccGetOrganizationCanonical()
	cfg := testAccGetTestConfig(t)
	repoURL := cfg.Repositories.Catalog.URL
	repoBranch := cfg.Repositories.Catalog.Branch
	credCanonical := cfg.Repositories.Catalog.Credential
	if repoURL == "" {
		t.Skip("repositories.catalog.url must be set for this test")
	}
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create with refresh_on_create = true: apply must succeed and the
			// resource must report refresh_on_create = true in state.
			{
				Config: testAccCatalogRepositoryConfig_withRefresh(orgCanonical, repoName, repoURL, repoBranch, credCanonical, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_catalog_repository.test_refresh", "organization_canonical", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_catalog_repository.test_refresh", "name", repoName),
					resource.TestCheckResourceAttr("cycloid_catalog_repository.test_refresh", "refresh_on_create", "true"),
				),
			},
			// Destroy
			{
				Config:  " ",
				Destroy: true,
			},
		},
	})
}

// testAccCatalogRepositoryConfig_withRefresh generates a cycloid_catalog_repository config
// with refresh_on_create set to the given value.
func testAccCatalogRepositoryConfig_withRefresh(org, name, url, branch, credCanonical string, refresh bool) string {
	cred := ""
	if credCanonical != "" {
		cred = fmt.Sprintf("  credential_canonical   = %q\n", credCanonical)
	}
	refreshVal := "false"
	if refresh {
		refreshVal = "true"
	}
	return fmt.Sprintf(`
resource "cycloid_catalog_repository" "test_refresh" {
  name                   = %q
  url                    = %q
  branch                 = %q
  organization_canonical = %q
  refresh_on_create      = %s
%s}
`, name, url, branch, org, refreshVal, cred)
}

// testAccCatalogRepositoryConfig_basic generates a cycloid_catalog_repository config.
// credential_canonical is omitted when credCanonical is empty (public repos).
func testAccCatalogRepositoryConfig_basic(org, name, url, branch, credCanonical string) string {
	cred := ""
	if credCanonical != "" {
		cred = fmt.Sprintf("  credential_canonical   = %q\n", credCanonical)
	}
	return fmt.Sprintf(`
resource "cycloid_catalog_repository" "test" {
  name                   = %q
  url                    = %q
  branch                 = %q
  organization_canonical = %q
%s}
`, name, url, branch, org, cred)
}

// TestCatalogRepositorySchema_RefreshOnCreateAttribute verifies that the refresh_on_create
// attribute is present in the schema with the expected type and default.
func TestCatalogRepositorySchema_RefreshOnCreateAttribute(t *testing.T) {
	ctx := context.Background()
	r := NewCatalogRepositoryResource().(*catalogRepositoryResource)

	var schemaResp tfresource.SchemaResponse
	r.Schema(ctx, tfresource.SchemaRequest{}, &schemaResp)

	attr, ok := schemaResp.Schema.Attributes["refresh_on_create"]
	require.True(t, ok, "refresh_on_create attribute must exist in schema")

	boolAttr, ok := attr.(schema.BoolAttribute)
	require.True(t, ok, "refresh_on_create must be a BoolAttribute")
	assert.True(t, boolAttr.IsOptional(), "refresh_on_create must be Optional")
	assert.True(t, boolAttr.IsComputed(), "refresh_on_create must be Computed (has default)")
	assert.NotEmpty(t, boolAttr.Description, "refresh_on_create must have a description")
}

// TestRefreshCatalogRepositoryVersions_HitsCorrectEndpoint verifies that
// refreshCatalogRepositoryVersions calls GET .../versions/refresh exactly once
// and propagates the error when the endpoint fails.
func TestRefreshCatalogRepositoryVersions_HitsCorrectEndpoint(t *testing.T) {
	var refreshCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/organizations/test-org/service_catalog_sources/test-catalog/versions/refresh" {
			atomic.AddInt32(&refreshCalls, 1)
			_, _ = fmt.Fprint(w, `{"data":[{"id":1,"commit_hash":"abc1234","name":"main","type":"branch"}]}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	apiClient := common.NewAPI(common.WithURL(srv.URL), common.WithToken("test-token"))
	r := &catalogRepositoryResource{
		provider: &CycloidProvider{
			APIClient:  apiClient,
			Middleware: middleware.NewMiddleware(apiClient),
		},
	}

	err := r.refreshCatalogRepositoryVersions("test-org", "test-catalog")
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&refreshCalls), "versions/refresh endpoint called exactly once")
}

// TestRefreshCatalogRepositoryVersions_PropagatesError verifies that an API error from
// the versions/refresh endpoint is surfaced to the caller.
func TestRefreshCatalogRepositoryVersions_PropagatesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "backend error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	apiClient := common.NewAPI(common.WithURL(srv.URL), common.WithToken("test-token"))
	r := &catalogRepositoryResource{
		provider: &CycloidProvider{
			APIClient:  apiClient,
			Middleware: middleware.NewMiddleware(apiClient),
		},
	}

	err := r.refreshCatalogRepositoryVersions("test-org", "test-catalog")
	require.Error(t, err, "error from versions/refresh must be propagated")
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
