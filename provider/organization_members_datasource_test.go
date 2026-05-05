package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit tests — no TF_ACC required

func TestOrgMembersToListValue_EmptyList(t *testing.T) {
	ctx := context.Background()

	listVal, diags := orgMembersToListValue(ctx, nil)

	require.False(t, diags.HasError())
	assert.Equal(t, 0, len(listVal.Elements()))
}

func TestOrgMembersToListValue_NilPointerFields(t *testing.T) {
	ctx := context.Background()

	members := []*models.MemberOrg{
		{
			// ID nil, Role nil, no Username/Email
		},
	}

	listVal, diags := orgMembersToListValue(ctx, members)

	require.False(t, diags.HasError())
	require.Equal(t, 1, len(listVal.Elements()))

	items := make([]orgMemberDatasourceItem, 1)
	diags = listVal.ElementsAs(ctx, &items, false)
	require.False(t, diags.HasError())
	assert.Equal(t, int64(0), items[0].MemberId.ValueInt64())
	assert.Equal(t, "", items[0].RoleCanonical.ValueString())
	assert.Equal(t, "", items[0].Email.ValueString())
}

func TestOrgMembersToListValue_NilRoleCanonical(t *testing.T) {
	ctx := context.Background()

	id := uint32(42)
	members := []*models.MemberOrg{
		{
			ID:   &id,
			Role: &models.Role{}, // Role exists but Canonical is nil
		},
	}

	listVal, diags := orgMembersToListValue(ctx, members)

	require.False(t, diags.HasError())
	require.Equal(t, 1, len(listVal.Elements()))
}

func TestOrgMembersToListValue_PendingPrefersInvitationEmail(t *testing.T) {
	ctx := context.Background()

	id := uint32(7)
	email := strfmt.Email("real@example.com")
	inviteEmail := strfmt.Email("invite@example.com")
	members := []*models.MemberOrg{
		{
			ID:              &id,
			Email:           email,
			InvitationEmail: inviteEmail,
			InvitationState: "pending",
		},
	}

	listVal, diags := orgMembersToListValue(ctx, members)

	require.False(t, diags.HasError())
	require.Equal(t, 1, len(listVal.Elements()))

	items := make([]orgMemberDatasourceItem, 1)
	diags = listVal.ElementsAs(ctx, &items, false)
	require.False(t, diags.HasError())
	assert.Equal(t, "invite@example.com", items[0].Email.ValueString())
	assert.Equal(t, "pending", items[0].InvitationState.ValueString())
}

func TestOrgMembersToListValue_AcceptedUsesEmail(t *testing.T) {
	ctx := context.Background()

	id := uint32(3)
	email := strfmt.Email("user@example.com")
	role := "organization-admin"
	members := []*models.MemberOrg{
		{
			ID:              &id,
			Email:           email,
			InvitationState: "accepted",
			Role:            &models.Role{Canonical: &role},
		},
	}

	listVal, diags := orgMembersToListValue(ctx, members)

	require.False(t, diags.HasError())
	items := make([]orgMemberDatasourceItem, 1)
	diags = listVal.ElementsAs(ctx, &items, false)
	require.False(t, diags.HasError())
	assert.Equal(t, "user@example.com", items[0].Email.ValueString())
	assert.Equal(t, "organization-admin", items[0].RoleCanonical.ValueString())
}

// Acceptance tests — require TF_ACC=1 and running stack

func TestAccOrganizationMembersDataSource(t *testing.T) {
	orgCanonical := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationMembersConfig_default(),
				Check: resource.ComposeTestCheckFunc(
					// At least one member exists (the bootstrap admin)
					resource.TestCheckResourceAttrSet("data.cycloid_organization_members.default", "members.0.member_id"),
					resource.TestCheckResourceAttrSet("data.cycloid_organization_members.default", "members.0.email"),
					// organization_canonical defaults to CY_ORG
					resource.TestCheckResourceAttr("data.cycloid_organization_members.default", "organization_canonical", os.Getenv("CY_ORG")),
				),
			},
			{
				Config: testAccOrganizationMembersConfig_explicit(orgCanonical),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cycloid_organization_members.explicit", "organization_canonical", orgCanonical),
					resource.TestCheckResourceAttrSet("data.cycloid_organization_members.explicit", "members.0.role_canonical"),
				),
			},
		},
	})
}

func TestAccOrganizationMembersDataSource_PendingInvite(t *testing.T) {
	orgCanonical := testAccGetOrganizationCanonical()
	depManager := NewTestDependencyManager(t)

	if depManager.provider == nil || depManager.provider.Middleware == nil {
		t.Skip("skipping: middleware not configured")
	}

	inviteEmail := fmt.Sprintf("test+%s@example.com", RandomCanonical(""))
	var invitedMemberID uint32

	preInvite := func() {
		m, _, err := depManager.provider.Middleware.InviteMember(orgCanonical, inviteEmail, "default-no-permissions")
		if err != nil {
			t.Fatalf("failed to invite test member: %v", err)
		}
		if m.ID != nil {
			invitedMemberID = *m.ID
		}
	}

	t.Cleanup(func() {
		if invitedMemberID != 0 {
			_, _ = depManager.provider.Middleware.DeleteMember(orgCanonical, invitedMemberID)
		}
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				PreConfig: preInvite,
				Config:    testAccOrganizationMembersConfig_explicit(orgCanonical),
				Check: resource.ComposeTestCheckFunc(
					checkOrgMembersContainsPendingEmail(orgCanonical, inviteEmail),
				),
			},
		},
	})
}

func TestAccOrganizationMembersDataSource_NotFound(t *testing.T) {
	depManager := NewTestDependencyManager(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config:      testAccOrganizationMembersConfig_explicit("does-not-exist-zzz"),
				ExpectError: regexp.MustCompile("failed to list organization members"),
			},
		},
	})
}

// checkOrgMembersContainsPendingEmail walks members.* attributes to find the expected pending entry.
func checkOrgMembersContainsPendingEmail(org, email string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		dsName := "data.cycloid_organization_members.explicit"
		rs, ok := s.RootModule().Resources[dsName]
		if !ok {
			return fmt.Errorf("%s not found in state", dsName)
		}

		attrs := rs.Primary.Attributes
		countStr, ok := attrs["members.#"]
		if !ok {
			return fmt.Errorf("members.# not set")
		}

		count := 0
		fmt.Sscanf(countStr, "%d", &count)

		for i := 0; i < count; i++ {
			prefix := fmt.Sprintf("members.%d.", i)
			if attrs[prefix+"email"] == email && attrs[prefix+"invitation_state"] == "pending" {
				return nil
			}
		}

		return fmt.Errorf("no pending member with email %q found in %s", email, dsName)
	}
}

func testAccOrganizationMembersConfig_default() string {
	return `data "cycloid_organization_members" "default" {}`
}

func testAccOrganizationMembersConfig_explicit(org string) string {
	return fmt.Sprintf(`
data "cycloid_organization_members" "explicit" {
  organization_canonical = %q
}
`, org)
}
