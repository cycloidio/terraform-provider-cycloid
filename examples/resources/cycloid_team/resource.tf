// Create an administrator team
resource "cycloid_team" "a_team" {
  name  = "The A-Team"
  owner = "hannibal"
  roles = [
    "organization-admin"
  ]
}
