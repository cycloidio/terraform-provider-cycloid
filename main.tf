terraform {
  required_providers {
    cycloid = {
      source = "registry.terraform.io/cycloidio/cycloid"
    }
  }
}

resource "cycloid_credential" "tf_credential" {
  name = "tf test 2"
  path = "path"
  type = "aws"
  body = {
    access_key = "access"
    secret_key = "secret"
  }
}

provider "cycloid" {
  url                    = "https://api.staging.cycloid.io/"
  jwt                    = "eyJhbGciOiJIUzI1NiIsImtpZCI6IjJmMjEyMmRlLTYzZjItNGVlYy05YzZmLWM2YWJiM2UxZjAwNyIsInR5cCI6IkpXVCJ9.eyJjeWNsb2lkIjp7InVzZXIiOnsiaWQiOjAsInVzZXJuYW1lIjoidGYtdGVzdCIsImdpdmVuX25hbWUiOiIiLCJmYW1pbHlfbmFtZSI6IiIsInBpY3R1cmVfdXJsIjoiIiwibG9jYWxlIjoiIn0sImFwaV9rZXkiOiJ0Zi10ZXN0Iiwib3JnYW5pemF0aW9uIjp7ImlkIjoxMiwiY2Fub25pY2FsIjoic2VyYWYiLCJuYW1lIjoiQ3ljbG9pZCBzdGFnaW5nIiwiYmxvY2tlZCI6W10sImhhc19jaGlsZHJlbiI6ZmFsc2UsInN1YnNjcmlwdGlvbiI6eyJleHBpcmVzX2F0IjotNjIxMzU1OTY4MDAsInBsYW4iOnsibmFtZSI6IkludmFsaWQiLCJjYW5vbmljYWwiOiJpbnZhbGlkIn19LCJhcHBlYXJhbmNlIjp7Im5hbWUiOiIiLCJjYW5vbmljYWwiOiIiLCJkaXNwbGF5X25hbWUiOiIiLCJsb2dvIjoiIiwiZmF2aWNvbiI6IiIsImZvb3RlciI6IiIsImlzX2FjdGl2ZSI6ZmFsc2UsImNvbG9yIjp7ImIiOjAsImciOjAsInIiOjB9fX0sImhhc2giOiI4NDEyMTBmMGM2YzI5Mzg1NjViMzQwYjYzYzRlNDA3NWM2ZjhjZGU1In0sInNjb3BlIjoiYXBpLWtleSIsImF1ZCI6ImN1c3RvbWVyIiwianRpIjoiNTkwOGNiNTgtNDM4Yy00MmYwLWEwZmEtM2JkNzJkMmRmNmQzIiwiaWF0IjoxNzA2NjA5Nzk5LCJpc3MiOiJodHRwczovL2N5Y2xvaWQuaW8iLCJuYmYiOjE3MDY2MDk3OTksInN1YiI6Imh0dHBzOi8vY3ljbG9pZC5pbyJ9.tCM8CLk5-eAnLR1e4-6VIajahYfspXo3aCiAd_kB9Vw"
  organization_canonical = "seraf"
}

#resource "cycloid_organization" "child_test" {
#name = "terraform organization test"
#}

