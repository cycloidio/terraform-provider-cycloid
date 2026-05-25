data "cycloid_organization_members" "all" {}

output "admin_emails" {
  value = [
    for m in data.cycloid_organization_members.all.members :
    m.email if m.role == "organization-admin"
  ]
}
