#! /usr/bin/env sh
#
# Backend initialization script.
#
# This script contains function for
# - Create the first org / admin / api-key
# - Init a child org, add a subscrption and an API key
#
# shellcheck disable=SC2317
# > _parse_args masks command usage in the script so we ignore this check

set -eu

test -n "${CI_DEBUG-}" && set -x

. ci/lib.sh

CY_API_URL=${CY_API_URL?}
CY_ADMIN_EMAIL=${CY_ADMIN_EMAIL?}
CY_ADMIN_PASSWORD=${CY_ADMIN_PASSWORD?}
CY_LICENCE=${API_LICENCE_KEY:?Input a licence key with an env var}

_require_repo_root

mkUrl() {
  echo "$CY_API_URL${1:?path as first arg}"
}

cy_curl() {
  method=${1?method as first arg}
  shift

  path=${1?path as second arg}
  shift

  curl -sS -X "$method" "$(mkUrl "$path")" \
    -H "Content-Type: application/vnd.cycloid.io.v1+json" \
    "$@"
}

auth_header() {
  token=${1-$(login)}
  echo "Authorization: Bearer $token"
}

refresh_token() {
  org=${1:?org as first arg}
  token=${2:?token as second arg}

  # refresh token for the new org
  route="/user/refresh_token?organization_canonical=${org}"
  response=$(
    cy_curl GET "$route" \
      -H "$(auth_header "$token")" \
      -w "%{http_code}"
  )
  msg=${response%???}
  code=${response#"$msg"}

  case "$code" in "200") _log_info "refresh token for org '$org' ok" ;; *) http_fail "$route" "$code" "$msg" ;; esac

  new_token=$(echo "$msg" | jq -r .data.token)
  export TOKEN="$new_token"
  echo "$TOKEN"
}

http_fail() {
  route=${1?api route as first arg}
  shift

  code=${1?code as second arg}
  shift

  msg="$*"

  _log_err "API call '$route' failed with code '$code':"
  _log_err "$msg"
  exit "$code"
}

create_admin() {
  response=$(
    cy_curl POST "/user" \
      -w "%{http_code}" \
      -s \
      --data "$(
        cat <<EOF
{
  "username": "admin",
  "email": "$CY_ADMIN_EMAIL",
  "password": "$CY_ADMIN_PASSWORD",
  "given_name": "admin",
  "family_name": "admin"
}
EOF
      )"
  )

  msg=${response%???}
  code=${response#"$msg"}

  case $code in
  "204" | "409") _log_info "admin created" ;;
  *)
    echo "api call to create user failed with code $code"
    echo "$msg"

    exit 1
    ;;
  esac
}

# Login then export the token to TOKEN env var
login() {
  response=$(
    cy_curl POST "/user/login" \
      -w "%{http_code}" \
      --data '{"email": "'"$CY_ADMIN_EMAIL"'", "password": "'"$CY_ADMIN_PASSWORD"'"}'
  )

  msg=${response%???}
  code=${response#"$msg"}

  if [ "$code" != "200" ]; then http_fail "/login" "$code" "$msg"; fi

  TOKEN="$(echo "$msg" | jq -r .data.token)"
  export TOKEN

  echo "$TOKEN"
}

# Licence an org given org + token
licence_org() {
  org=${1?org as first arg}
  shift

  token=${1?token a second arg, dont forget token refresh}

  route="/organizations/${org}/licence"
  response=$(
    cy_curl POST "$route" \
      -H "$(auth_header "$token")" \
      -w "%{http_code}" \
      --data '{"key": "'"$CY_LICENCE"'"}'
  )

  msg=${response%???}
  code=${response#"$msg"}

  case "$code" in "204") _log_info "org '$org' is licenced." ;; *) http_fail "$route" "$code" "$msg" ;; esac
}

create_org() {
  org_canonical="${1:?give the org canonical as first parameter}"
  token=$(login)

  # Ensure the org is created
  route="/organizations"
  response=$(
    cy_curl POST "$route" \
      -H "$(auth_header "$token")" \
      -w "%{http_code}" \
      --data '{"name": "'"$org_canonical"'", "canonical": "'"$org_canonical"'"}'
  )

  msg=${response%???}
  code=${response#"$msg"}

  case "$code" in "200" | "409") _log_info "org '$org_canonical' created." ;; *) http_fail "$route" "$code" "$msg" ;; esac

  licence_org "$org_canonical" "$(refresh_token "$org_canonical" "$token")"

}

create_child_org() {
  org_canonical="${1?give the org canonical as first parameter}"
  token="${2?token}"

  # Ensure the org is created
  route="/organizations/${CY_ROOT_ORG}/children"
  response=$(
    cy_curl POST "$route" \
      -H "$(auth_header "$token")" \
      -w "%{http_code}" \
      --data '{"name": "'"$org_canonical"'", "canonical": "'"$org_canonical"'"}'
  )

  msg=${response%???}
  code=${response#"$msg"}

  case "$code" in "200" | "409") _log_info "org '$org_canonical' created." ;; *) http_fail "$route" "$code" "$msg" ;; esac

  licence_org "$org_canonical" "$(refresh_token "$org_canonical" "$token")"

}

delete_api_key() {
  org=${1?org as first arg}
  token=${2?refreshed token as second arg}
  api_canonical=${3?api key canonical as third arg}

  route="/organizations/${org}/api_keys/${api_canonical}"
  response=$(
    cy_curl DELETE "$route" \
      -H "$(auth_header "$token")" \
      -w "%{http_code}" \
      --data "$payload"
  )
}

_register_cmd put_api_key "org: create and api key and write it to '<org>-api-key' file"
put_api_key() {
  org=${1?org as first arg}
  token=${2-$(refresh_token "$org" "$(login)")}

  route="/organizations/${org}/api_keys"
  payload='{"name":"admin-token","description":"first admin token generated for this cycloid org.","canonical":"admin-token","rules":[{"action":"organization:**","effect":"allow","resources":[]}]}'
  response=$(
    cy_curl POST "$route" \
      -H "$(auth_header "$token")" \
      -w "%{http_code}" \
      --data "$payload"
  )

  msg=${response%???}
  code=${response#"$msg"}

  case "$code" in "200")
    _log_info "api_key for '$org' created"
    ;;
  "409")
    _log_warn "api_key already exists for '$org', recreating..."
    delete_api_key "$org" "$token" "admin-token"
    put_api_key "$org" "$token"
    exit 0
    ;;
  *) http_fail "$route" "$code" "$msg" ;; esac

  CY_API_KEY=$(echo "$msg" | jq -r .data.token)
  export CY_API_KEY

  echo "$CY_API_KEY" | tee "$(_get_org_token_path "$org")"
}

add_member() {
  org="${1?org}"
  token=${2?token}
  email="${3?email}"
  role="${4-organization-admin}"

  route="/organizations/${org}/members"
  response=$(cy_curl POST "$route" \
    -H "$(auth_header "$token")" \
    -w "%{http_code}" \
    --data '{"email":"'"$email"'", "role_canonical": "'"$role"'"}')

  msg=${response%???}
  code=${response#"$msg"}

  case "$code" in
  200) _log_info "member $email added as $role to $org." ;;
  *) http_fail "$route" "$code" "$msg" ;;
  esac
}

_register_cmd init_root_org "create root org with admin user according to the env variables params. -> return <org_api_key>"
init_root_org() {
  create_admin
  create_org "$CY_ROOT_ORG"
  put_api_key "$CY_ROOT_ORG" "$(refresh_token "$CY_ROOT_ORG" "$(login)")" | tee ./.ci/"${CY_ROOT_ORG}-api-key"
}

_register_cmd init_child_org "org: token: create and licence an <org> using <token> -> return <org_api_key>"
init_child_org() {
  org=${1?org as first arg}
  token="$(refresh_token "$CY_ROOT_ORG" "$(login)")"

  create_child_org "$org" "$token"

  ./ci/dc.sh cmd exec -it "youdeploy-api" /go/youdeploy-http-api --config-file "/ci/config.yml" \
    create-subscription \
    --organization-canonical "$org" \
    --members-count 10 \
    --plan-canonical platform_teams \
    --overwrite \
    --expiration-date "$(date --date "+1 year" +"%Y-%m-%dT%TZ")" >"./.ci/${org}_backend_init.log" 2>&1

  ./ci/dc.sh cmd exec -it "youdeploy-api" /go/youdeploy-http-api --config-file "/ci/config.yml" \
    create-subscription \
    --organization-canonical "$org" \
    --members-count 10 \
    --plan-canonical end_users \
    --overwrite \
    --expiration-date "$(date --date "+1 year" +"%Y-%m-%dT%TZ")" >>"./.ci/${org}_backend_init.log" 2>&1

  # add_member "$org" "$token" "$CY_ADMIN_EMAIL"
  put_api_key "$org" >"$(_get_org_token_path "$org")"
}

_parse_subcommand "$@"
