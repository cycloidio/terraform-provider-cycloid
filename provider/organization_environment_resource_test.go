package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

// TestOrgEnvironmentToValue is a pure unit test (runs under `go test -short`)
// covering the API-model -> Terraform-state mapping for the org-scoped
// environment resource, including the owner / environment-type edge cases.
func TestOrgEnvironmentToValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name          string
		Org           string
		Env           *models.Environment
		ExpectedCanon string
		ExpectedName  string
		ExpectedDesc  string
		ExpectedType  types.String
		ExpectedOwner string
		ExpectedID    types.Int64
	}{
		{
			Name: "full environment with type and owner",
			Org:  "org-root",
			Env: &models.Environment{
				Canonical:   ptr.Ptr("production"),
				Name:        "Production",
				Description: "prod env",
				ID:          ptr.Ptr(uint32(42)),
				CreatedAt:   ptr.Ptr(uint64(1000)),
				UpdatedAt:   ptr.Ptr(uint64(2000)),
				EnvironmentType: &models.EnvironmentType{
					Canonical: ptr.Ptr("production"),
				},
				Owner: &models.User{Username: ptr.Ptr("alice")},
			},
			ExpectedCanon: "production",
			ExpectedName:  "Production",
			ExpectedDesc:  "prod env",
			ExpectedType:  types.StringValue("production"),
			ExpectedOwner: "alice",
			ExpectedID:    types.Int64Value(42),
		},
		{
			Name: "environment without type or owner",
			Org:  "org-root",
			Env: &models.Environment{
				Canonical: ptr.Ptr("staging"),
				Name:      "Staging",
				ID:        ptr.Ptr(uint32(7)),
			},
			ExpectedCanon: "staging",
			ExpectedName:  "Staging",
			ExpectedDesc:  "",
			ExpectedType:  types.StringNull(),
			ExpectedOwner: "",
			ExpectedID:    types.Int64Value(7),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var data organizationEnvironmentResourceModel
			diags := orgEnvironmentToValue(context.Background(), tc.Org, tc.Env, &data)
			if diags.HasError() {
				t.Fatal(diags)
			}

			assert.Equal(t, tc.ExpectedCanon, data.Canonical.ValueString(), "canonical")
			assert.Equal(t, tc.ExpectedName, data.Name.ValueString(), "name")
			assert.Equal(t, tc.Org, data.Organization.ValueString(), "organization")
			assert.Equal(t, tc.ExpectedDesc, data.Description.ValueString(), "description")
			assert.Equal(t, tc.ExpectedType, data.Type, "type")
			assert.Equal(t, tc.ExpectedOwner, data.Owner.ValueString(), "owner")
			assert.Equal(t, tc.ExpectedID, data.ID, "id")
		})
	}
}

// TestOrgEnvTypeForUpdate covers the precedence rules for the environment type
// sent on update: explicit config > current value on the API > default.
func TestOrgEnvTypeForUpdate(t *testing.T) {
	t.Parallel()

	withType := &models.Environment{
		EnvironmentType: &models.EnvironmentType{Canonical: ptr.Ptr("staging")},
	}
	noType := &models.Environment{}

	testCases := []struct {
		Name       string
		Current    *models.Environment
		Configured string
		Expected   string
	}{
		{"configured wins", withType, "production", "production"},
		{"falls back to current", withType, "", "staging"},
		{"defaults when nothing set", noType, "", defaultOrgEnvType},
		{"defaults when current nil", nil, "", defaultOrgEnvType},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			assert.Equal(t, tc.Expected, orgEnvTypeForUpdate(tc.Current, tc.Configured))
		})
	}
}

// TestAccOrganizationEnvironmentResource exercises create -> read -> update ->
// delete of an org-scoped environment against a live API. It skips when
// CY_API_URL / CY_API_KEY / CY_ORG are not configured (the repo's acceptance
// pattern), so it is safe to run under `go test` without credentials.
func TestAccOrganizationEnvironmentResource(t *testing.T) {
	t.Parallel()

	envName := RandomCanonical("test-org-env")

	ctx := context.Background()
	orgCanonical := testAccGetOrganizationCanonical()

	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	// Reuse the dependency manager only to gate on credentials being present.
	if depManager.GetProvider().Middleware == nil {
		t.Skip("skipping acceptance test: CY_API_URL, CY_API_KEY and CY_ORG must be set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create org-level environment, no project link.
			{
				Config: testAccOrgEnvironmentConfig_basic(orgCanonical, envName, "production"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_organization_environment.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_organization_environment.test", "name", envName),
					resource.TestCheckResourceAttr("cycloid_organization_environment.test", "type", "production"),
				),
			},
			// Update description.
			{
				Config: testAccOrgEnvironmentConfig_updated(orgCanonical, envName, "production", "updated description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_organization_environment.test", "name", envName),
					resource.TestCheckResourceAttr("cycloid_organization_environment.test", "description", "updated description"),
				),
			},
			// Import.
			{
				ResourceName:      "cycloid_organization_environment.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Import is by canonical (ImportStatePassthroughID -> path "canonical"),
				// not the numeric id attribute. Supply the canonical explicitly.
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["cycloid_organization_environment.test"]
					if rs == nil {
						return "", fmt.Errorf("cycloid_organization_environment.test not in state")
					}
					return rs.Primary.Attributes["canonical"], nil
				},
				// owner/cloud_account/variables are not refreshed in Read by design.
				// organization is sourced from the provider default on import.
				ImportStateVerifyIgnore: []string{"owner", "cloud_account_canonicals", "variables", "organization"},
			},
			// Destroy: deletes the org environment itself.
			{
				Config:  " ",
				Destroy: true,
			},
		},
	})
}

func testAccOrgEnvironmentConfig_basic(org, env, envType string) string {
	return fmt.Sprintf(`
resource "cycloid_organization_environment" "test" {
  organization = %q
  name         = %q
  type         = %q
}
`, org, env, envType)
}

func testAccOrgEnvironmentConfig_updated(org, env, envType, desc string) string {
	return fmt.Sprintf(`
resource "cycloid_organization_environment" "test" {
  organization = %q
  name         = %q
  type         = %q
  description  = %q
}
`, org, env, envType, desc)
}
