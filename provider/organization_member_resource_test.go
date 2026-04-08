package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOrganizationMemberResource(t *testing.T) {
	t.Parallel()

	const (
		memberEmail   = "test@example.com"
		roleCanonical = "organization-admin"
	)
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

func testAccOrganizationMemberConfig_updated(org, email, role string) string {
	return fmt.Sprintf(`
resource "cycloid_organization_member" "test" {
  organization_canonical = "%s"
  email                 = "%s"
  role_canonical        = "%s"
}
`, org, email, role)
}
