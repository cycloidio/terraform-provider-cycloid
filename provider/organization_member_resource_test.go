package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit tests — no TF_ACC required

func TestOrgMemberCYModelToData_AcceptedMember(t *testing.T) {
	id := uint32(42)
	role := "organization-admin"
	email := strfmt.Email("user@example.com")
	m := &models.MemberOrg{
		ID:              &id,
		Email:           email,
		InvitationState: "accepted",
		Role:            &models.Role{Canonical: &role},
		Username:        "john-doe",
	}
	var data organizationMemberResourceModel

	diags := orgMemberCYModelToData("my-org", m, &data)

	require.False(t, diags.HasError())
	assert.Equal(t, "john-doe", data.MemberCanonical.ValueString())
	assert.Equal(t, "user@example.com", data.Email.ValueString())
	assert.Equal(t, int64(42), data.MemberId.ValueInt64())
	assert.Equal(t, "organization-admin", data.RoleCanonical.ValueString())
}

// Pending invitations have no Username yet — member_canonical is empty until the invite is accepted.
func TestOrgMemberCYModelToData_PendingInvite(t *testing.T) {
	id := uint32(7)
	role := "default-no-permissions"
	inviteEmail := strfmt.Email("invite@example.com")
	m := &models.MemberOrg{
		ID:              &id,
		InvitationEmail: inviteEmail,
		InvitationState: "pending",
		Role:            &models.Role{Canonical: &role},
		Username:        "",
	}
	var data organizationMemberResourceModel

	diags := orgMemberCYModelToData("my-org", m, &data)

	require.False(t, diags.HasError())
	assert.Equal(t, "", data.MemberCanonical.ValueString())
	assert.Equal(t, types.StringValue(""), data.MemberCanonical)
	assert.Equal(t, "invite@example.com", data.Email.ValueString())
}

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
					// Pending invitations have no username yet; member_canonical is empty until accepted.
					resource.TestCheckResourceAttr("cycloid_organization_member.test", "member_canonical", ""),
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
