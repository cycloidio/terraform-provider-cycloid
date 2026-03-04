// Create an administrator team
resource "cycloid_team" "a_team" {
  name         = "The A-Team"
  owner        = "hannibal"
  organization = "mercenaries"
  roles = [
    "organization-admin"
  ]
}

resource "cycloid_team_member" "lads" {
  for_each     = toset(["hannibal", "baracus", "murdock", "peck"])
  team         = cycloid_team.a_team.canonical
  organization = cycloid_team.a_team.organization // make sure you use the organization linked to the team
  username     = each.value
}

resource "cycloid_team_member" "email_invite" {
  team         = cycloid_team.a_team.canonical
  organization = cycloid_team.a_team.organization
  username     = "amy.allen@a-team.com"
}
