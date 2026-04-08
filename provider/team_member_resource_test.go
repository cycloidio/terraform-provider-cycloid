package provider

import (
	"context"
	"fmt"
	"testing"

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
