package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCloudAccountResource_basic(t *testing.T) {
	t.Parallel()

	credName := RandomCanonical("test-ca-cred")
	accountName := RandomCanonical("test-cloud-account")

	ctx := context.Background()
	orgCanonical := testAccGetOrganizationCanonical()

	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	if depManager.provider.Middleware == nil {
		t.Skip("skipping acceptance test: CY_API_URL, CY_API_KEY and CY_ORG must be set")
	}

	// Pre-create the credential outside TF to avoid the credential resource's post-apply drift.
	cred, _, err := depManager.provider.Middleware.CreateCredential(
		orgCanonical,
		credName,
		"aws",
		&models.CredentialRaw{AccessKey: "AKIAIOSFODNN7EXAMPLE", SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"},
		credName, // path == canonical
		credName,
		"",
	)
	if err != nil {
		t.Fatalf("failed to pre-create test credential: %v", err)
	}
	credCanonical := *cred.Canonical
	depManager.cleanupItems = append(depManager.cleanupItems, cleanupItem{
		resourceType: "credential",
		canonical:    credCanonical,
		cleanupFunc: func() error {
			_, err := depManager.provider.Middleware.DeleteCredential(orgCanonical, credCanonical)
			return err
		},
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccCloudAccountConfig_basic(orgCanonical, credCanonical, accountName, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_cloud_account.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_cloud_account.test", "name", accountName),
					resource.TestCheckResourceAttr("cycloid_cloud_account.test", "cloud_provider", "aws"),
					resource.TestCheckResourceAttrSet("cycloid_cloud_account.test", "canonical"),
					resource.TestCheckResourceAttrSet("cycloid_cloud_account.test", "id"),
				),
			},
			// Import
			{
				ResourceName:      "cycloid_cloud_account.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["cycloid_cloud_account.test"]
					if rs == nil {
						return "", fmt.Errorf("cycloid_cloud_account.test not in state")
					}
					return rs.Primary.Attributes["canonical"], nil
				},
				ImportStateVerifyIgnore: []string{"organization"},
			},
			// Update (description only; cloud_provider + credential_canonical trigger replacement)
			{
				Config: testAccCloudAccountConfig_basic(orgCanonical, credCanonical, accountName, "updated description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_cloud_account.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_cloud_account.test", "name", accountName),
					resource.TestCheckResourceAttr("cycloid_cloud_account.test", "description", "updated description"),
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

func testAccCloudAccountConfig_basic(org, credCanonical, accountName, description string) string {
	descAttr := ""
	if description != "" {
		descAttr = fmt.Sprintf(`  description = "%s"`, description)
	}
	return fmt.Sprintf(`
resource "cycloid_cloud_account" "test" {
  organization         = "%s"
  name                 = "%s"
  cloud_provider       = "aws"
  credential_canonical = "%s"
%s
}
`, org, accountName, credCanonical, descAttr)
}
