package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTeamMemberResource(t *testing.T) {
	t.Parallel()
	t.Skip("Team member assignMemberToTeam returns 500 (entity not created); under investigation – see LINEAR_ISSUE_TEAM_MEMBER_500.md")

	const (
		username = "testuser"
		email    = "testuser@example.com"
	)
	ctx := context.Background()
	orgCanonical := testAccGetOrganizationCanonical()
	teamName := RandomCanonical("test-team")
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create team first
			{
				Config: testAccTeamMemberConfig_team(orgCanonical, teamName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_team.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_team.test", "name", teamName),
				),
			},
			// Create team member with organization parameter
			{
				Config: testAccTeamMemberConfig_basic(orgCanonical, teamName, username, email),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_team_member.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_team_member.test", "team", "cycloid_team.test.canonical"),
					resource.TestCheckResourceAttr("cycloid_team_member.test", "username", username),
					resource.TestCheckResourceAttr("cycloid_team_member.test", "email", email),
				),
			},
			// Update team member
			{
				Config: testAccTeamMemberConfig_updated(orgCanonical, teamName, username+"-updated", email+"-updated"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_team_member.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_team_member.test", "team", "cycloid_team.test.canonical"),
					resource.TestCheckResourceAttr("cycloid_team_member.test", "username", username+"-updated"),
					resource.TestCheckResourceAttr("cycloid_team_member.test", "email", email+"-updated"),
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

func TestFindTeamMemberIgnoresEmptyUsername(t *testing.T) {
	firstEmail := strfmt.Email("first@example.com")
	expectedEmail := strfmt.Email("expected@example.com")

	teamMembers := []*models.MemberTeam{
		{
			Username: "",
			Email:    &firstEmail,
		},
		{
			Username: "",
			Email:    &expectedEmail,
		},
	}

	teamMember := findTeamMember(teamMembers, "", "expected@example.com")

	if teamMember == nil {
		t.Fatal("expected team member to be found")
		return
	}
	if teamMember.Email == nil || teamMember.Email.String() != "expected@example.com" {
		t.Fatalf("expected email %q, got %v", "expected@example.com", teamMember.Email)
	}
}

func TestFindTeamMemberIgnoresEmptyEmail(t *testing.T) {
	firstEmail := strfmt.Email("")
	expectedEmail := strfmt.Email("")

	teamMembers := []*models.MemberTeam{
		{
			Username: "first",
			Email:    &firstEmail,
		},
		{
			Username: "expected",
			Email:    &expectedEmail,
		},
	}

	teamMember := findTeamMember(teamMembers, "expected", "")

	if teamMember == nil {
		t.Fatal("expected team member to be found")
		return
	}
	if teamMember.Username != "expected" {
		t.Fatalf("expected username %q, got %v", "expected", teamMember.Username)
	}
}

// TestAccTeamMemberResource_ByEmail verifies that a team member can be assigned and
// looked up using only an email address (no username). This is the acceptance-level
// regression test for the findTeamMember fix: the old loop would match any member
// whose username was "" when the lookup username was also "", causing wrong-member
// selection or false-positive matches. Using the bootstrap admin (always present in
// the test org) keeps the test self-contained without needing extra member setup.
func TestAccTeamMemberResource_ByEmail(t *testing.T) {
	t.Parallel()

	const adminEmail = "admin@cycloid.io"
	ctx := context.Background()
	orgCanonical := testAccGetOrganizationCanonical()
	teamName := RandomCanonical("test-team")
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccTeamMemberConfig_emailOnly(orgCanonical, teamName, adminEmail),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_team_member.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_team_member.test", "email", adminEmail),
				),
			},
			{
				Config:  " ",
				Destroy: true,
			},
		},
	})
}

// TestAccTeamMemberResource_ByUsername verifies that a team member can be assigned and
// looked up using only a username (no email). Mirrors TestAccTeamMemberResource_ByEmail
// for the other lookup path in findTeamMember.
func TestAccTeamMemberResource_ByUsername(t *testing.T) {
	t.Parallel()

	const adminUsername = "administrator"
	ctx := context.Background()
	orgCanonical := testAccGetOrganizationCanonical()
	teamName := RandomCanonical("test-team")
	depManager := NewTestDependencyManager(t)
	defer depManager.Cleanup(ctx, t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: depManager.GetProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccTeamMemberConfig_usernameOnly(orgCanonical, teamName, adminUsername),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cycloid_team_member.test", "organization", orgCanonical),
					resource.TestCheckResourceAttr("cycloid_team_member.test", "username", adminUsername),
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
func testAccTeamMemberConfig_team(org, team string) string {
	return fmt.Sprintf(`
resource "cycloid_team" "test" {
  organization = "%s"
  name         = "%s"
  roles        = ["organization-admin"]
}
`, org, team)
}

func testAccTeamMemberConfig_basic(org, team, username, email string) string {
	return fmt.Sprintf(`
resource "cycloid_team" "test" {
  organization = "%s"
  name         = "%s"
  roles        = ["organization-admin"]
}

resource "cycloid_team_member" "test" {
  organization = "%s"
  team         = cycloid_team.test.canonical
  username     = "%s"
  email        = "%s"
}
`, org, team, org, username, email)
}

func testAccTeamMemberConfig_updated(org, team, username, email string) string {
	return fmt.Sprintf(`
resource "cycloid_team" "test" {
  organization = "%s"
  name         = "%s"
  roles        = ["organization-admin"]
}

resource "cycloid_team_member" "test" {
  organization = "%s"
  team         = cycloid_team.test.canonical
  username     = "%s"
  email        = "%s"
}
`, org, team, org, username, email)
}

func testAccTeamMemberConfig_emailOnly(org, team, email string) string {
	return fmt.Sprintf(`
resource "cycloid_team" "test" {
  organization = "%s"
  name         = "%s"
  roles        = ["organization-admin"]
}

resource "cycloid_team_member" "test" {
  organization = "%s"
  team         = cycloid_team.test.canonical
  email        = "%s"
}
`, org, team, org, email)
}

func testAccTeamMemberConfig_usernameOnly(org, team, username string) string {
	return fmt.Sprintf(`
resource "cycloid_team" "test" {
  organization = "%s"
  name         = "%s"
  roles        = ["organization-admin"]
}

resource "cycloid_team_member" "test" {
  organization = "%s"
  team         = cycloid_team.test.canonical
  username     = "%s"
}
`, org, team, org, username)
}
