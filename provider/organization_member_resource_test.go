package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOrganizationMemberResource(t *testing.T) {
	t.Parallel()

	const roleCanonical = "organization-admin"
	memberEmail := fmt.Sprintf("test+%s@example.com", RandomCanonical(""))
	ctx := context.Background()
	orgCanonical := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create organization member with organization_canonical parameter
			{
				Config: testAccOrganizationMemberConfig_basic(orgCanonical, memberEmail, roleCanonical),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_organization_member.test", "organization_canonical", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_organization_member.test", "email", memberEmail),
					resource.TestCheckResourceAttr("cycloid_organization_member.test", "role_canonical", roleCanonical),
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

// TestAccOrganizationMemberResource_existingAdmin reproduces TFPRO-37: when the
// invitee is already a member of the org with a different role (e.g. auto-enrolled
// as organization-admin), InviteMember returns the existing role and the provider
// previously crashed with "Provider produced inconsistent result after apply".
func TestAccOrganizationMemberResource_existingAdmin(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	memberEmail := fmt.Sprintf("test+%s@example.com", RandomCanonical(""))
	orgCanonical := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	if depManager.provider.Middleware == nil {
		t.Skip("skipping acceptance test: CY_API_URL, CY_API_KEY and CY_ORG must be set")
	}

	// Pre-invite as organization-admin outside of Terraform to simulate a user who
	// is already a member with a higher role when TF first encounters them.
	preInvite := func() {
		if _, _, err := depManager.provider.Middleware.InviteMember(orgCanonical, memberEmail, "organization-admin"); err != nil {
			t.Fatalf("pre-invite setup failed: %v", err)
		}
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				PreConfig: preInvite,
				Config:    testAccOrganizationMemberConfig_basic(orgCanonical, memberEmail, "default-no-permissions"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_organization_member.test", "organization_canonical", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_organization_member.test", "email", memberEmail),
					resource.TestCheckResourceAttr("cycloid_organization_member.test", "role_canonical", "default-no-permissions"),
				),
			},
			{
				Config:  " ",
				Destroy: true,
			},
		},
	})
}

// Test configuration functions
func testAccOrganizationMemberConfig_basic(org, email, role string) string {
	return fmt.Sprintf(`
resource "cycloid_organization_member" "test" {
  organization_canonical = "%s"
  email                 = "%s"
  role_canonical        = "%s"
}
`, org, email, role)
}
