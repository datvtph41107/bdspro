#!/usr/bin/env bash
# Disposable HTTP execution example; never run against production data.
set -euo pipefail

if [[ "${BDS_TEST_MODE:-}" != "disposable" ]]; then
  echo "Refusing to seed non-disposable database" >&2
  exit 1
fi
DB_URL="${BDS_TEST_DB_URL:?BDS_TEST_DB_URL is required}"
if [[ "$DB_URL" != *"/bdspro_test?"* ]]; then
  echo "Refusing to seed a database other than bdspro_test" >&2
  exit 1
fi
BASE="${BDS_TEST_API_BASE:-http://127.0.0.1:8080/api/v1}"
: "${BDS_TEST_TOKEN_KEY:?BDS_TEST_TOKEN_KEY is required}"

session_owner="00000000-0000-4000-8000-000000000001"
session_member="00000000-0000-4000-8000-000000000002"

# Existing migrations are already installed in bdspro_test by CI.
ids="$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 <<'SQL'
INSERT INTO parties DEFAULT VALUES RETURNING id AS owner_person \gset
INSERT INTO persons(party_id) VALUES (:owner_person);
INSERT INTO accounts(person_id, email, is_email_verified)
VALUES (:owner_person, 'team-demo-owner@example.invalid', true)
RETURNING id AS owner_account \gset
INSERT INTO parties DEFAULT VALUES RETURNING id AS member_person \gset
INSERT INTO persons(party_id) VALUES (:member_person);
INSERT INTO accounts(person_id, email, is_email_verified)
VALUES (:member_person, 'team-demo-member@example.invalid', true)
RETURNING id AS member_account \gset
INSERT INTO parties DEFAULT VALUES RETURNING id AS organization_id \gset
INSERT INTO organizations(party_id, name) VALUES (:organization_id, 'Demo Organization');
INSERT INTO organization_members(organization_id, person_id, role, owner_person_id)
VALUES (:organization_id, :owner_person, 'owner', :owner_person);
INSERT INTO organization_members(organization_id, person_id, role)
VALUES (:organization_id, :member_person, 'member');
INSERT INTO sessions(id, account_id, refresh_token, user_agent, client_ip, expires_at)
VALUES ('00000000-0000-4000-8000-000000000001', :owner_account,
  'disposable-owner-refresh', 'smoke', '127.0.0.1', now() + interval '1 hour');
INSERT INTO sessions(id, account_id, refresh_token, user_agent, client_ip, expires_at)
VALUES ('00000000-0000-4000-8000-000000000002', :member_account,
  'disposable-member-refresh', 'smoke', '127.0.0.1', now() + interval '1 hour');
\echo :organization_id :owner_account :member_account
SQL
)"
read -r org_id owner_id member_id <<< "$ids"
if [[ ! "$org_id" =~ ^[0-9]+$ || ! "$owner_id" =~ ^[0-9]+$ || ! "$member_id" =~ ^[0-9]+$ ]]; then
  echo "Invalid disposable fixture IDs: $ids" >&2
  exit 1
fi

token_source="$(mktemp /tmp/bdspro-team-token-XXXXXXXX.go)"
response="$(mktemp /tmp/bdspro-team-response-XXXXXXXX.json)"
trap 'rm -f "$token_source" "$response"' EXIT

cat > "$token_source" <<'GO'
package main

import (
    "fmt"
    "os"
    "strconv"
    "time"

    "github.com/datvtph41107/bdspro/modules/auth"
    "github.com/google/uuid"
)

func main() {
    id, err := strconv.ParseInt(os.Getenv("BDS_TEAM_DEMO_ACCOUNT"), 10, 64)
    if err != nil { panic(err) }
    session, err := uuid.Parse(os.Getenv("BDS_TEAM_DEMO_SESSION"))
    if err != nil { panic(err) }
    maker, err := auth.NewJWTMaker(os.Getenv("BDS_TEST_TOKEN_KEY"))
    if err != nil { panic(err) }
    token, _, err := maker.CreateSessionAccessToken(id, 10*time.Minute, session)
    if err != nil { panic(err) }
    fmt.Print(token)
}
GO

make_token() {
  BDS_TEAM_DEMO_ACCOUNT="$1" BDS_TEAM_DEMO_SESSION="$2" go run "$token_source"
}
owner_token="$(make_token "$owner_id" "$session_owner")"
member_token="$(make_token "$member_id" "$session_member")"

call_api() {
  curl -sS -o "$response" -w '%{http_code}' "$@"
}
assert_status() {
  if [[ "$1" != "$2" ]]; then
    echo "Expected HTTP $2, got $1; body:" >&2
    cat "$response" >&2
    echo >&2
    exit 1
  fi
}

url="$BASE/orgs/$org_id/teams"
status="$(call_api -H "Authorization: Bearer $owner_token" "$url")"
assert_status "$status" 200
echo "List teams (owner): 200"

status="$(call_api -X POST -H "Authorization: Bearer $owner_token" \
  -H 'Content-Type: application/json' -d '{"name":"Engineering","description":"Backend"}' "$url")"
assert_status "$status" 201
team_id="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["id"])' "$response")"
echo "Create team (owner): 201; id=$team_id"

status="$(call_api -H "Authorization: Bearer $member_token" "$url/$team_id")"
assert_status "$status" 200
echo "Read team (member): 200"

status="$(call_api -X PATCH -H "Authorization: Bearer $owner_token" \
  -H 'Content-Type: application/json' -d '{"name":"Platform"}' "$url/$team_id")"
assert_status "$status" 200
python3 -c 'import json,sys; assert json.load(open(sys.argv[1]))["name"] == "Platform"' "$response"
echo "Rename team (owner): 200; name=Platform"

status="$(call_api -X PATCH -H "Authorization: Bearer $owner_token" \
  -H 'Content-Type: application/json' -d '{}' "$url/$team_id")"
assert_status "$status" 400
echo "Omitted PATCH name field rejected: 400"

status="$(call_api -X PATCH -H "Authorization: Bearer $owner_token" \
  -H 'Content-Type: application/json' -d '{"name":""}' "$url/$team_id")"
assert_status "$status" 400
echo "Present but empty PATCH name rejected: 400"

status="$(call_api -X PATCH -H "Authorization: Bearer $owner_token" \
  -H 'Content-Type: application/json' -d '{"name":null}' "$url/$team_id")"
assert_status "$status" 400
echo "Null PATCH name rejected: 400"

status="$(call_api -X PATCH -H "Authorization: Bearer $member_token" \
  -H 'Content-Type: application/json' -d '{"name":"Unauthorized"}' "$url/$team_id")"
assert_status "$status" 403
echo "Rename team (member): 403"

status="$(call_api -X POST -H "Authorization: Bearer $owner_token" \
  -H 'Content-Type: application/json' -d '{"name":"Platform"}' "$url")"
assert_status "$status" 409
echo "Duplicate team name: 409"

status="$(call_api -X POST -H "Authorization: Bearer $owner_token" \
  -H 'Content-Type: application/json' -d '{"name":"Blocked-by-IP-Admission"}' "$url")"
assert_status "$status" 429
echo "Third Team create request denied by IPActive: 429"

status="$(call_api "$url")"
assert_status "$status" 401
echo "Missing Bearer token: 401"

psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -c \
  "UPDATE sessions SET is_blocked = true WHERE id = '$session_owner';" >/dev/null
status="$(call_api -H "Authorization: Bearer $owner_token" "$url")"
assert_status "$status" 401
echo "Revoked session: 401"
