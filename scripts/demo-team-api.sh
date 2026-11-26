#!/usr/bin/env bash
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
headers="$(mktemp /tmp/bdspro-team-headers-XXXXXXXX.txt)"
oversized="$(mktemp /tmp/bdspro-team-oversized-XXXXXXXX.json)"
trap 'rm -f "$token_source" "$response" "$headers" "$oversized"' EXIT

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

status="$(curl -sS -D "$headers" -o "$response" -w '%{http_code}' \
  -H "Authorization: Bearer $owner_token" "$url?page=2&limit=1")"
assert_status "$status" 200
python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["page"]==2 and d["limit"]==1 and d["total"]==1 and len(d["items"])==0' "$response"
grep -qi '^X-Total-Count: 1' "$headers"
grep -q 'rel="prev"' "$headers"
grep -qi '^Access-Control-Expose-Headers:.*X-Total-Count' "$headers"
echo "Team pagination SQL count and Link/X-Total-Count headers: 200"

status="$(call_api -H "Authorization: Bearer $owner_token" "$url?page=0")"
assert_status "$status" 400
status="$(call_api -H "Authorization: Bearer $owner_token" "$url?limit=999")"
assert_status "$status" 400
echo "Invalid pagination rejected: 400"

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

status="$(call_api -X PATCH -H "Authorization: Bearer $owner_token" \
  -H 'Content-Type: application/json' -d '{"name":"Alpha"}{"name":"Beta"}' "$url/$team_id")"
assert_status "$status" 400
echo "Multiple JSON documents rejected: 400"

python3 -c 'import sys; open(sys.argv[1],"wb").write(b"{\"name\":\"" + b"x"*1048576 + b"\"}")' "$oversized"
status="$(call_api -X PATCH -H "Authorization: Bearer $owner_token" \
  -H 'Content-Type: application/json' --data-binary "@$oversized" "$url/$team_id")"
assert_status "$status" 413
echo "Oversized JSON body rejected: 413"

status="$(call_api -X PATCH -H "Authorization: Bearer $member_token" \
  -H 'Content-Type: application/json' -d '{"name":"Unauthorized"}' "$url/$team_id")"
assert_status "$status" 403
echo "Rename team (member): 403"

status="$(call_api -X POST -H "Authorization: Bearer $owner_token" \
  -H 'Content-Type: application/json' -d '{"name":"Platform"}' "$url")"
assert_status "$status" 409
echo "Duplicate team name: 409"

status="$(curl -sS -D "$headers" -o "$response" -w '%{http_code}' -X POST \
  -H "Authorization: Bearer $owner_token" -H 'Content-Type: application/json' \
  -d '{"name":"Blocked-by-IP-Admission"}' "$url")"
assert_status "$status" 429
grep -qi "^Retry-After: [1-9]" "$headers"
echo "Third Team create request denied by IPActive: 429 with Retry-After"

status="$(call_api "$url")"
assert_status "$status" 401
echo "Missing Bearer token: 401"

hash_source="$(mktemp /tmp/bdspro-password-XXXXXXXX.go)"
cookie_jar="$(mktemp /tmp/bdspro-cookie-XXXXXXXX.txt)"
old_cookie="$(mktemp /tmp/bdspro-cookie-old-XXXXXXXX.txt)"
trap 'rm -f "$token_source" "$response" "$headers" "$oversized" "$hash_source" "$cookie_jar" "$old_cookie"' EXIT
cat > "$hash_source" <<'GO'
package main
import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)
func main() {
    h,err := bcrypt.GenerateFromPassword([]byte("disposable-test-password"), bcrypt.DefaultCost)
    if err != nil { panic(err) }
    fmt.Print(string(h))
}
GO
password_hash="$(go run "$hash_source")"
psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -v account_id="$owner_id" -v password_hash="$password_hash" <<'SQL'
INSERT INTO password_credentials(account_id,password_hash) VALUES (:account_id, :'password_hash');
SQL
login_url="$BASE/auth/login"
browser_origin="${BDS_TEST_BROWSER_ORIGIN:-http://127.0.0.1:8080}"
browser_headers=(-H "Origin: $browser_origin" -H 'X-BDS-Request: 1' -H 'Content-Type: application/json')
status="$(call_api -X POST -H 'Content-Type: application/json' -d '{"identifier":"team-demo-owner@example.invalid","password":"disposable-test-password"}' "$login_url")"
assert_status "$status" 403
status="$(call_api -X POST "${browser_headers[@]}" -d '{"identifier":"team-demo-owner@example.invalid","password":"wrong-password"}' "$login_url")"
assert_status "$status" 401
status="$(call_api -X POST "${browser_headers[@]}" -d '{"identifier":"does-not-exist@example.invalid","password":"wrong-password"}' "$login_url")"
assert_status "$status" 401
status="$(call_api -c "$cookie_jar" -X POST "${browser_headers[@]}" -d '{"identifier":"team-demo-owner@example.invalid","password":"disposable-test-password","stay_signed_in":true}' "$login_url")"
assert_status "$status" 200
login_access="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["status"]=="authenticated" and d["user"]["id"]==sys.argv[2] and d["user"]["credentials"]["password_set"]; assert "refresh_token" not in d and "Refresh" not in d; print(d["access_token"])' "$response" "$owner_id")"
status="$(call_api -H "Authorization: Bearer $login_access" "$BASE/me")"
assert_status "$status" 200
python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["id"]==sys.argv[2] and d["email"]=="team-demo-owner@example.invalid"; assert "password_hash" not in d' "$response" "$owner_id"
cp "$cookie_jar" "$old_cookie"
status="$(call_api -b "$cookie_jar" -c "$cookie_jar" -X POST "${browser_headers[@]}" -d '{}' "$BASE/auth/refresh")"
assert_status "$status" 200
rotated_access="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["access_token"])' "$response")"
status="$(call_api -b "$old_cookie" -X POST "${browser_headers[@]}" -d '{}' "$BASE/auth/refresh")"
assert_status "$status" 401
status="$(call_api -H "Authorization: Bearer $rotated_access" "$BASE/me")"
assert_status "$status" 401
status="$(call_api -c "$cookie_jar" -X POST "${browser_headers[@]}" -d '{"identifier":"team-demo-owner@example.invalid","password":"disposable-test-password"}' "$login_url")"
assert_status "$status" 200
logout_access="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["access_token"])' "$response")"
status="$(call_api -b "$cookie_jar" -c "$cookie_jar" -X POST "${browser_headers[@]}" -d '{}' "$BASE/auth/logout")"
assert_status "$status" 204
status="$(call_api -H "Authorization: Bearer $logout_access" "$BASE/me")"
assert_status "$status" 401
echo "Verified email password login, /me, rotation/reuse revocation and logout: passed"

status="$(call_api -c "$cookie_jar" -X POST "${browser_headers[@]}" -d '{"identifier":"team-demo-owner@example.invalid","password":"disposable-test-password"}' "$login_url")"
assert_status "$status" 200
before_change="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["access_token"])' "$response")"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $before_change" -d '{"current_password":"wrong-password","new_password":"disposable-changed-password"}' "$BASE/me/password/change")"
assert_status "$status" 401
status="$(call_api -b "$cookie_jar" -c "$cookie_jar" -X POST "${browser_headers[@]}" -d '{}' "$BASE/auth/logout")"
assert_status "$status" 204
sessions_url="$BASE/me/sessions"
other_owner_session="00000000-0000-4000-8000-000000000003"
psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -c \
  "INSERT INTO sessions(id,account_id,refresh_token,user_agent,client_ip,expires_at) VALUES ('$other_owner_session',$owner_id,'disposable-secondary-refresh','smoke','127.0.0.1',now()+interval '1 hour');" >/dev/null
secondary_token="$(make_token "$owner_id" "$other_owner_session")"

status="$(call_api "$sessions_url")"
assert_status "$status" 401
status="$(call_api -H "Authorization: Bearer $owner_token" "$sessions_url")"
assert_status "$status" 200
python3 -c 'import json,sys; rows=json.load(open(sys.argv[1]))["items"]; assert len(rows)==2; assert {r["id"] for r in rows}=={sys.argv[2],sys.argv[3]}; assert sum(r["current"] for r in rows)==1; assert all(set(r)=={"id","user_agent","client_ip","created_at","expires_at","current"} for r in rows)' "$response" "$session_owner" "$other_owner_session"

status="$(call_api -X DELETE -H "Authorization: Bearer $owner_token" "$sessions_url/$session_member")"
assert_status "$status" 404
status="$(call_api -H "Authorization: Bearer $member_token" "$sessions_url")"
assert_status "$status" 200
status="$(call_api -X DELETE -H "Authorization: Bearer $owner_token" "$sessions_url/not-a-uuid")"
assert_status "$status" 400

status="$(call_api -X DELETE -H "Authorization: Bearer $owner_token" "$sessions_url/$other_owner_session")"
assert_status "$status" 204
status="$(call_api -H "Authorization: Bearer $secondary_token" "$url")"
assert_status "$status" 401
status="$(call_api -H "Authorization: Bearer $owner_token" "$sessions_url")"
assert_status "$status" 200
python3 -c 'import json,sys; rows=json.load(open(sys.argv[1]))["items"]; assert len(rows)==1 and rows[0]["id"]==sys.argv[2] and rows[0]["current"]' "$response" "$session_owner"
status="$(call_api -X DELETE -H "Authorization: Bearer $owner_token" "$sessions_url/$other_owner_session")"
assert_status "$status" 404

status="$(call_api -X DELETE -H "Authorization: Bearer $owner_token" "$sessions_url/$session_owner")"
assert_status "$status" 204
status="$(call_api -H "Authorization: Bearer $owner_token" "$url")"
assert_status "$status" 401
status="$(call_api -H "Authorization: Bearer $owner_token" "$sessions_url")"
assert_status "$status" 401
audit_count="$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM audit_events WHERE action='auth.session.revoked' AND actor_account_id=$owner_id AND target_ref IN ('$session_owner','$other_owner_session')")"
test "$audit_count" = 2
echo "Session list, cross-account denial, selected/current revocation and audit: passed"

status="$(call_api -c "$cookie_jar" -X POST "${browser_headers[@]}" -d '{"identifier":"team-demo-owner@example.invalid","password":"disposable-test-password"}' "$login_url")"
assert_status "$status" 200
before_change="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["access_token"])' "$response")"
status="$(call_api -b "$cookie_jar" -c "$cookie_jar" -X POST "${browser_headers[@]}" -H "Authorization: Bearer $before_change" -d '{"current_password":"disposable-test-password","new_password":"disposable-changed-password"}' "$BASE/me/password/change")"
assert_status "$status" 200
after_change="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["access_token"])' "$response")"
status="$(call_api -H "Authorization: Bearer $before_change" "$BASE/me")"
assert_status "$status" 401
status="$(call_api -H "Authorization: Bearer $after_change" "$BASE/me")"
assert_status "$status" 200
status="$(call_api -X POST "${browser_headers[@]}" -d '{"identifier":"team-demo-owner@example.invalid","password":"disposable-test-password"}' "$login_url")"
assert_status "$status" 401
read -r reset_proof reset_digest <<< "$(python3 -c 'import secrets,hashlib; t=secrets.token_urlsafe(32); print(t,hashlib.sha256(t.encode()).hexdigest())')"
psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -v account_id="$owner_id" -v digest="$reset_digest" <<'SQL'
INSERT INTO password_reset_tokens(token_hash,account_id,expires_at) VALUES (decode(:'digest','hex'),:account_id,now()+interval '15 minutes');
SQL
status="$(call_api -X POST "${browser_headers[@]}" -d '{"email":"team-demo-owner@example.invalid"}' "$BASE/auth/password/reset/request")"
assert_status "$status" 503
reset_body="$(python3 -c 'import json,sys; print(json.dumps({"reset_token":sys.argv[1],"new_password":"disposable-recovered-password"}))' "$reset_proof")"
status="$(call_api -X POST "${browser_headers[@]}" -d "$reset_body" "$BASE/auth/password/reset/confirm")"
assert_status "$status" 204
status="$(call_api -X POST "${browser_headers[@]}" -d "$reset_body" "$BASE/auth/password/reset/confirm")"
assert_status "$status" 400
status="$(call_api -H "Authorization: Bearer $after_change" "$BASE/me")"
assert_status "$status" 401
status="$(call_api -X POST "${browser_headers[@]}" -d '{"identifier":"team-demo-owner@example.invalid","password":"disposable-recovered-password"}' "$login_url")"
assert_status "$status" 200
password_audit="$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM audit_events WHERE action IN ('auth.password.changed','auth.password.reset') AND actor_account_id=$owner_id")"
test "$password_audit" = 2
echo "Password change, recovery proof consumption/replay rejection, session rotation/revocation and audit: passed"

profile_access="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["access_token"])' "$response")"
status="$(call_api "$BASE/me/profile")"
assert_status "$status" 401
status="$(call_api -H "Authorization: Bearer $profile_access" "$BASE/me/settings")"
assert_status "$status" 200
python3 -c 'import json,sys; assert json.load(open(sys.argv[1]))=={"locale":"vi","timezone":"Asia/Ho_Chi_Minh","currency":"VND"}' "$response"
status="$(call_api -X PATCH -H "Authorization: Bearer $profile_access" -H 'Content-Type: application/json' -d '{"display_name":"Test Owner","first_name":"Owner"}' "$BASE/me/profile")"
assert_status "$status" 200
status="$(call_api -X PATCH -H "Authorization: Bearer $profile_access" -H 'Content-Type: application/json' -d '{"last_name":"Updated"}' "$BASE/me/profile")"
assert_status "$status" 200
python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d=={"display_name":"Test Owner","first_name":"Owner","last_name":"Updated"}' "$response"
status="$(call_api -X PATCH -H "Authorization: Bearer $profile_access" -H 'Content-Type: application/json' -d '{"display_name":null}' "$BASE/me/profile")"
assert_status "$status" 400
status="$(call_api -X PATCH -H "Authorization: Bearer $profile_access" -H 'Content-Type: application/json' -d '{"currency":"EUR"}' "$BASE/me/settings")"
assert_status "$status" 400
status="$(call_api -X PATCH -H "Authorization: Bearer $profile_access" -H 'Content-Type: application/json' -d '{"locale":"en","timezone":"UTC","currency":"USD"}' "$BASE/me/settings")"
assert_status "$status" 200
status="$(call_api -H "Authorization: Bearer $profile_access" "$BASE/me")"
assert_status "$status" 200
python3 -c 'import json,sys; assert json.load(open(sys.argv[1]))["display_name"]=="Test Owner"' "$response"
status="$(call_api -H "Authorization: Bearer $member_token" "$BASE/me/profile")"
assert_status "$status" 200
python3 -c 'import json,sys; assert json.load(open(sys.argv[1]))["display_name"]==""' "$response"
status="$(call_api -H "Authorization: Bearer $profile_access" "$BASE/me/credentials")"
assert_status "$status" 200
python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["email"]=="t***@example.invalid" and d["credentials"]["password_set"]; assert "password_hash" not in d' "$response"
status="$(call_api -H "Authorization: Bearer $profile_access" "$BASE/me/connections")"
assert_status "$status" 200
python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert 0<len(d["items"])<=50; assert all(set(r)=={"id","session_id","action","ip_address","user_agent","created_at"} for r in d["items"])' "$response"
echo "Own profile, settings, credential summary and connection history: passed"

status="$(call_api -X POST "${browser_headers[@]}" -d '{"email":"register-new@example.invalid"}' "$BASE/auth/register/start")"
assert_status "$status" 503

registration_browser="$(python3 -c 'import secrets; print(secrets.token_urlsafe(32))')"
registration_cookie="bdspro_register_dev=$registration_browser"
reg_code="12345678"
seed_registration() {
  local email="$1" challenge="$2"
  local digests browser_digest code_mac
  digests="$(BDS_REG_BROWSER="$registration_browser" BDS_REG_EMAIL="$email" BDS_REG_ID="$challenge" BDS_REG_CODE="$reg_code" python3 -c 'import os,hashlib,hmac; b=hashlib.sha256(os.environ["BDS_REG_BROWSER"].encode()).digest(); p=("bdspro.email.registration.v1\x00"+os.environ["BDS_REG_ID"]+"\x00"+os.environ["BDS_REG_EMAIL"]+"\x00").encode()+b+os.environ["BDS_REG_CODE"].encode(); print(b.hex(),hmac.new(os.environ["BDS_TEST_TOKEN_KEY"].encode(),p,hashlib.sha256).hexdigest())')"
  read -r browser_digest code_mac <<< "$digests"
  psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -v id="$challenge" -v email="$email" -v browser="$browser_digest" -v mac="$code_mac" <<'SQL'
INSERT INTO email_registration_challenges(id,email,browser_digest,code_mac,expires_at,resend_after)
VALUES (:'id',:'email',decode(:'browser','hex'),decode(:'mac','hex'),now()+interval '5 minutes',now()+interval '60 seconds');
SQL
}
new_challenge="00000000-0000-4000-8000-000000000101"
old_challenge="00000000-0000-4000-8000-000000000102"
locked_challenge="00000000-0000-4000-8000-000000000103"
seed_registration "register-new@example.invalid" "$new_challenge"
seed_registration "team-demo-owner@example.invalid" "$old_challenge"
seed_registration "register-locked@example.invalid" "$locked_challenge"
reg_verify="$BASE/auth/register/verify"
verify_body() { printf '{"challenge_id":"%s","code":"%s"}' "$1" "$2"; }

status="$(call_api -X POST "${browser_headers[@]}" -d "$(verify_body "$new_challenge" "$reg_code")" "$reg_verify")"
assert_status "$status" 400
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$(verify_body "$new_challenge" "00000000")" "$reg_verify")"
assert_status "$status" 400
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT failed_attempts FROM email_registration_challenges WHERE id='$new_challenge'")" = 1
for attempt in 1 2 3 4 5; do
  status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$(verify_body "$locked_challenge" "00000000")" "$reg_verify")"
  assert_status "$status" 400
done
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$(verify_body "$locked_challenge" "$reg_code")" "$reg_verify")"
assert_status "$status" 400
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT failed_attempts FROM email_registration_challenges WHERE id='$locked_challenge'")" = 5

status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$(verify_body "$old_challenge" "$reg_code")" "$reg_verify")"
assert_status "$status" 200
python3 -c 'import json,sys; assert json.load(open(sys.argv[1]))=={"status":"sign_in_required"}' "$response"
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$(verify_body "$new_challenge" "$reg_code")" "$reg_verify")"
assert_status "$status" 200
reg_proof="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["status"]=="registration_required"; print(d["registration_token"])' "$response")"
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM accounts WHERE email='register-new@example.invalid'")" = 0
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$(verify_body "$new_challenge" "$reg_code")" "$reg_verify")"
assert_status "$status" 400
complete_body="$(python3 -c 'import json,sys; print(json.dumps({"registration_token":sys.argv[1],"username":"New_User","password":"disposable-register-password","terms_version":"v1","marketing_opt_in":False,"stay_signed_in":False}))' "$reg_proof")"
status="$(call_api -X POST "${browser_headers[@]}" -d "$complete_body" "$BASE/auth/register/complete")"
assert_status "$status" 400
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$complete_body" "$BASE/auth/register/complete")"
assert_status "$status" 201
reg_access="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["status"]=="authenticated" and d["user"]["username"]=="new_user" and d["user"]["onboarding"]["completed"] and d["user"]["credentials"]["email_verified"]; print(d["access_token"])' "$response")"
reg_account="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["user"]["id"])' "$response")"
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$complete_body" "$BASE/auth/register/complete")"
assert_status "$status" 400
status="$(call_api -X POST "${browser_headers[@]}" -d '{"identifier":"NEW_USER","password":"disposable-register-password"}' "$login_url")"
assert_status "$status" 200
python3 -c 'import json,sys; assert json.load(open(sys.argv[1]))["user"]["id"]==sys.argv[2]' "$response" "$reg_account"
status="$(call_api -X PATCH -H "Authorization: Bearer $member_token" -H 'Content-Type: application/json' -d '{"username":"NEW_USER"}' "$BASE/me/username")"
assert_status "$status" 409
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM audit_events WHERE action='auth.account.registered' AND actor_account_id=$reg_account")" = 1
psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -c "UPDATE accounts SET status='suspended' WHERE id=$reg_account"
status="$(call_api -H "Authorization: Bearer $reg_access" "$BASE/me")"
assert_status "$status" 401
status="$(call_api -X POST "${browser_headers[@]}" -d '{"identifier":"NEW_USER","password":"disposable-register-password"}' "$login_url")"
assert_status "$status" 401
echo "Email registration proof, replay, attempt limit, existing email, username login/conflict, audit and suspension: passed"

status="$(call_api -X POST "${browser_headers[@]}" -d '{"phone":"bad","country_code":"VN"}' "$BASE/auth/phone/start")"
assert_status "$status" 400
status="$(call_api -X POST "${browser_headers[@]}" -d '{"phone":"+12025550123"}' "$BASE/auth/phone/start")"
assert_status "$status" 503
phone_code="123456"
seed_phone() {
  local phone="$1" challenge="$2"
  local digests browser_digest code_mac
  digests="$(BDS_REG_BROWSER="$registration_browser" BDS_PHONE="$phone" BDS_REG_ID="$challenge" BDS_PHONE_CODE="$phone_code" python3 -c 'import os,hashlib,hmac; b=hashlib.sha256(os.environ["BDS_REG_BROWSER"].encode()).digest(); p=("bdspro.phone.continuation.v1\x00"+os.environ["BDS_REG_ID"]+"\x00"+os.environ["BDS_PHONE"]+"\x00").encode()+b+os.environ["BDS_PHONE_CODE"].encode(); print(b.hex(),hmac.new(os.environ["BDS_TEST_TOKEN_KEY"].encode(),p,hashlib.sha256).hexdigest())')"
  read -r browser_digest code_mac <<< "$digests"
  psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -v id="$challenge" -v phone="$phone" -v browser="$browser_digest" -v mac="$code_mac" <<'SQL'
INSERT INTO phone_continuation_challenges(id,phone_e164,browser_digest,code_mac,expires_at,resend_after)
VALUES (:'id',:'phone',decode(:'browser','hex'),decode(:'mac','hex'),now()+interval '5 minutes',now()+interval '60 seconds');
SQL
}
phone_new="00000000-0000-4000-8000-000000000201"
phone_race="00000000-0000-4000-8000-000000000202"
phone_locked="00000000-0000-4000-8000-000000000203"
phone_existing="00000000-0000-4000-8000-000000000204"
phone_suspended="00000000-0000-4000-8000-000000000205"
phone_expired="00000000-0000-4000-8000-000000000206"
seed_phone "+12025550123" "$phone_new"
seed_phone "+12025550123" "$phone_race"
seed_phone "+12025550124" "$phone_locked"
seed_phone "+12025550125" "$phone_expired"
psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -c "UPDATE phone_continuation_challenges SET expires_at=now()-interval '1 second' WHERE id='$phone_expired'"
phone_verify="$BASE/auth/phone/verify"
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$(verify_body "$phone_expired" "$phone_code")" "$phone_verify")"
assert_status "$status" 400
status="$(call_api -X POST "${browser_headers[@]}" -d "$(verify_body "$phone_new" "$phone_code")" "$phone_verify")"
assert_status "$status" 400
for attempt in 1 2 3 4 5; do
  status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$(verify_body "$phone_locked" "000000")" "$phone_verify")"
  assert_status "$status" 400
done
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$(verify_body "$phone_locked" "$phone_code")" "$phone_verify")"
assert_status "$status" 400
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT failed_attempts FROM phone_continuation_challenges WHERE id='$phone_locked'")" = 5
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$(verify_body "$phone_new" "$phone_code")" "$phone_verify")"
assert_status "$status" 200
phone_proof="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["status"]=="registration_required" and "user" not in d; print(d["registration_token"])' "$response")"
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$(verify_body "$phone_race" "$phone_code")" "$phone_verify")"
assert_status "$status" 200
race_proof="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["status"]=="registration_required"; print(d["registration_token"])' "$response")"
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM phone_credentials WHERE phone_e164='+12025550123'")" = 0
phone_complete_body() { python3 -c 'import json,sys; print(json.dumps({"registration_token":sys.argv[1],"username":sys.argv[2],"terms_version":"v1","marketing_opt_in":False,"stay_signed_in":False}))' "$1" "$2"; }
phone_body="$(phone_complete_body "$phone_proof" "phone_user")"
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$phone_body" "$BASE/auth/phone/complete")"
assert_status "$status" 201
phone_account="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); u=d["user"]; assert d["status"]=="authenticated" and u["username"]=="phone_user" and u["email"] is None and u["phone"]=="+12025550123" and u["credentials"]["phone_verified"] and not u["credentials"]["password_set"]; print(u["id"])' "$response")"
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$phone_body" "$BASE/auth/phone/complete")"
assert_status "$status" 400
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$(phone_complete_body "$race_proof" "other_phone_user")" "$BASE/auth/phone/complete")"
assert_status "$status" 409
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM accounts WHERE username='other_phone_user'")" = 0
seed_phone "+12025550123" "$phone_existing"
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$(verify_body "$phone_existing" "$phone_code")" "$phone_verify")"
assert_status "$status" 200
phone_access="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["status"]=="authenticated" and d["user"]["id"]==sys.argv[2]; print(d["access_token"])' "$response" "$phone_account")"
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$(verify_body "$phone_existing" "$phone_code")" "$phone_verify")"
assert_status "$status" 400
status="$(call_api -H "Authorization: Bearer $phone_access" "$BASE/me")"
assert_status "$status" 200
psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -c "UPDATE accounts SET status='suspended' WHERE id=$phone_account"
seed_phone "+12025550123" "$phone_suspended"
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$(verify_body "$phone_suspended" "$phone_code")" "$phone_verify")"
assert_status "$status" 401
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM audit_events WHERE action='auth.account.registered' AND actor_account_id=$phone_account")" = 1
echo "Phone proof, expired OTP, lockout, new identity, ownership conflict, existing login, replay, suspension and audit: passed"

status="$(call_api "$BASE/auth/google/start?stay_signed_in=true")"
assert_status "$status" 503
status="$(call_api "$BASE/auth/google/callback?code=synthetic&state=synthetic")"
assert_status "$status" 503

seed_google_result() {
  local subject="$1" email="$2" authoritative="$3"
  google_browser="$(python3 -c 'import secrets; print(secrets.token_urlsafe(32))')"
  google_cookie="bdspro_google_dev=$google_browser"
  local browser_digest
  browser_digest="$(python3 -c 'import hashlib,sys; print(hashlib.sha256(sys.argv[1].encode()).hexdigest())' "$google_browser")"
  psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -v browser="$browser_digest" -v subject="$subject" -v email="$email" -v authoritative="$authoritative" <<'SQL'
INSERT INTO google_auth_flows(browser_digest,phase,subject,email,authoritative_email,remember,expires_at)
VALUES (decode(:'browser','hex'),'ready',:'subject',:'email',:'authoritative'::boolean,false,now()+interval '5 minutes');
SQL
}
google_result="$BASE/auth/google/result"
google_complete="$BASE/auth/google/complete"
seed_google_result "disposable-conflicting-subject" "team-demo-owner@example.invalid" true
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" "$google_result")"
assert_status "$status" 200
python3 -c 'import json,sys; assert json.load(open(sys.argv[1]))=={"status":"sign_in_required"}' "$response"
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM oauth2_credentials WHERE subject='disposable-conflicting-subject'")" = 0
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" "$google_result")"
assert_status "$status" 400

seed_google_result "disposable-new-google-subject" "third-party@example.invalid" false
status="$(call_api -X POST "${browser_headers[@]}" "$google_result")"
assert_status "$status" 400
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" "$google_result")"
assert_status "$status" 200
google_proof="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["status"]=="registration_required"; print(d["registration_token"])' "$response")"
google_body="$(python3 -c 'import json,sys; print(json.dumps({"registration_token":sys.argv[1],"username":"google_user","terms_version":"v1","marketing_opt_in":False}))' "$google_proof")"
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM oauth2_credentials WHERE subject='disposable-new-google-subject'")" = 0
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" "$google_result")"
assert_status "$status" 400
status="$(call_api -X POST "${browser_headers[@]}" -b "$registration_cookie" -d "$google_body" "$google_complete")"
assert_status "$status" 400
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" -d "$google_body" "$google_complete")"
assert_status "$status" 201
google_account="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); u=d["user"]; assert d["status"]=="authenticated" and u["username"]=="google_user" and u["email"] is None and u["credentials"]["google_linked"] and not u["credentials"]["email_verified"] and not u["credentials"]["password_set"]; print(u["id"])' "$response")"
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" -d "$google_body" "$google_complete")"
assert_status "$status" 400
seed_google_result "disposable-new-google-subject" "changed-email@example.invalid" true
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" "$google_result")"
assert_status "$status" 200
google_access="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["status"]=="authenticated" and d["user"]["id"]==sys.argv[2] and d["user"]["email"] is None; print(d["access_token"])' "$response" "$google_account")"
status="$(call_api -H "Authorization: Bearer $google_access" "$BASE/me")"
assert_status "$status" 200
psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -c "UPDATE accounts SET status='suspended' WHERE id=$google_account"
seed_google_result "disposable-new-google-subject" "changed-email@example.invalid" true
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" "$google_result")"
assert_status "$status" 401
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM audit_events WHERE action='auth.account.registered' AND actor_account_id=$google_account")" = 1
echo "Google result binding, new identity, untrusted email, email conflict, replay, stable subject and suspended account: passed"

status="$(call_api -X POST "${browser_headers[@]}" -d '{"identifier":"team-demo-owner@example.invalid","password":"disposable-recovered-password"}' "$login_url")"
assert_status "$status" 200
credential_access="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["access_token"])' "$response")"
reauth_body() { python3 -c 'import json,sys; print(json.dumps({"password":"disposable-recovered-password","action":sys.argv[1]}))' "$1"; }
proof_body() { python3 -c 'import json,sys; print(json.dumps({"reauthentication_proof":sys.argv[1]}))' "$1"; }
reauth_proof() { python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["expires_in"]==600; print(d["reauthentication_proof"])' "$response"; }
reauth_url="$BASE/me/reauth/password"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $credential_access" -d '{"password":"wrong-password","action":"password.unlink"}' "$reauth_url")"
assert_status "$status" 401
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $credential_access" -d "$(reauth_body password.unlink)" "$reauth_url")"
assert_status "$status" 200
password_unlink_proof="$(reauth_proof)"
status="$(call_api -X DELETE "${browser_headers[@]}" -H "Authorization: Bearer $credential_access" -d "$(proof_body "$password_unlink_proof")" "$BASE/me/email")"
assert_status "$status" 403
status="$(call_api -X DELETE "${browser_headers[@]}" -H "Authorization: Bearer $member_token" -d "$(proof_body "$password_unlink_proof")" "$BASE/me/password")"
assert_status "$status" 403
status="$(call_api -X DELETE "${browser_headers[@]}" -H "Authorization: Bearer $credential_access" -d "$(proof_body "$password_unlink_proof")" "$BASE/me/password")"
assert_status "$status" 403
python3 -c 'import json,sys; assert json.load(open(sys.argv[1]))["code"]=="last_credential"' "$response"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $credential_access" -d "$(reauth_body email.unlink)" "$reauth_url")"
assert_status "$status" 200
email_unlink_proof="$(reauth_proof)"
status="$(call_api -X DELETE "${browser_headers[@]}" -H "Authorization: Bearer $credential_access" -d "$(proof_body "$email_unlink_proof")" "$BASE/me/email")"
assert_status "$status" 403
python3 -c 'import json,sys; assert json.load(open(sys.argv[1]))["code"]=="credential_dependency"' "$response"
psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -c "INSERT INTO oauth2_credentials(issuer,subject,account_id) VALUES ('https://accounts.google.com','disposable-owner-link',$owner_id)"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $credential_access" -d "$(reauth_body google.unlink)" "$reauth_url")"
assert_status "$status" 200
google_unlink_proof="$(reauth_proof)"
status="$(call_api -X DELETE "${browser_headers[@]}" -H "Authorization: Bearer $credential_access" -d "$(proof_body "$google_unlink_proof")" "$BASE/me/google")"
assert_status "$status" 200
credential_next="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert not d["user"]["credentials"]["google_linked"] and d["user"]["credentials"]["password_set"]; print(d["access_token"])' "$response")"
status="$(call_api -H "Authorization: Bearer $credential_access" "$BASE/me")"
assert_status "$status" 401
status="$(call_api -X DELETE "${browser_headers[@]}" -H "Authorization: Bearer $credential_next" -d "$(proof_body "$google_unlink_proof")" "$BASE/me/google")"
assert_status "$status" 403
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $credential_next" -d "$(reauth_body sessions.revoke_all)" "$reauth_url")"
assert_status "$status" 200
revoke_all_proof="$(reauth_proof)"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $credential_next" -d "$(proof_body "$revoke_all_proof")" "$BASE/me/sessions/revoke-all")"
assert_status "$status" 204
status="$(call_api -H "Authorization: Bearer $credential_next" "$BASE/me")"
assert_status "$status" 401
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM audit_events WHERE actor_account_id=$owner_id AND action IN ('auth.credential.removed','auth.sessions.revoked_all')")" = 2
seed_google_result "disposable-setup-subject" "setup-google@example.invalid" true
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" "$google_result")"
assert_status "$status" 200
setup_proof="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["registration_token"])' "$response")"
setup_body="$(python3 -c 'import json,sys; print(json.dumps({"registration_token":sys.argv[1],"username":"setup_google","terms_version":"v1"}))' "$setup_proof")"
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" -d "$setup_body" "$google_complete")"
assert_status "$status" 201
setup_account="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["user"]["id"])' "$response")"
setup_access="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["access_token"])' "$response")"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $setup_access" "$BASE/me/reauth/email/start")"
assert_status "$status" 503
setup_session="$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT id FROM sessions WHERE account_id=$setup_account AND NOT is_blocked")"
read -r setup_token setup_digest <<< "$(python3 -c 'import secrets,hashlib; t=secrets.token_urlsafe(32); print(t,hashlib.sha256(t.encode()).hexdigest())')"
psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -v digest="$setup_digest" -v account="$setup_account" -v session="$setup_session" <<'SQL'
INSERT INTO recent_auth_proofs(digest,account_id,session_id,action,method,email_snapshot,expires_at)
VALUES (decode(:'digest','hex'),:account,:'session','password.set','verified_email','setup-google@example.invalid',now()+interval '10 minutes');
SQL
setup_password_body="$(python3 -c 'import json,sys; print(json.dumps({"new_password":"disposable-setup-password","reauthentication_proof":sys.argv[1]}))' "$setup_token")"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $setup_access" -d "$setup_password_body" "$BASE/me/password/set")"
assert_status "$status" 200
setup_next="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["user"]["credentials"]["password_set"]; print(d["access_token"])' "$response")"
status="$(call_api -H "Authorization: Bearer $setup_access" "$BASE/me")"
assert_status "$status" 401
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $setup_next" -d "$setup_password_body" "$BASE/me/password/set")"
assert_status "$status" 409
status="$(call_api -X POST "${browser_headers[@]}" -d '{"identifier":"setup_google","password":"disposable-setup-password"}' "$login_url")"
assert_status "$status" 200
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM audit_events WHERE actor_account_id=$setup_account AND action='auth.password.set'")" = 1
echo "Recent auth purpose/account/session binding, last credential, dependency, credential removal, revoke-all and password setup: passed"

link_access="$setup_next"
session_from_token() {
  python3 -c 'import base64,json,sys; p=sys.argv[1].split(".")[1]; print(json.loads(base64.urlsafe_b64decode(p+"="*(-len(p)%4)))["session_id"])' "$1"
}
link_session="$(session_from_token "$link_access")"
link_reauth() {
  python3 -c 'import json,sys; print(json.dumps({"password":"disposable-setup-password","action":sys.argv[1]}))' "$1"
}
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $link_access" -d "$(link_reauth email.link)" "$reauth_url")"
assert_status "$status" 200
link_start_proof="$(reauth_proof)"
link_start_body="$(python3 -c 'import json,sys; print(json.dumps({"email":"linked-new@example.invalid","reauthentication_proof":sys.argv[1]}))' "$link_start_proof")"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $link_access" -d "$link_start_body" "$BASE/me/email/start")"
assert_status "$status" 503
seed_contact_link() {
  local kind="$1" destination="$2" challenge="$3" code="$4"
  local mac
  mac="$(BDS_LINK_KIND="$kind" BDS_LINK_DEST="$destination" BDS_LINK_ID="$challenge" BDS_LINK_CODE="$code" BDS_LINK_ACCOUNT="$setup_account" BDS_LINK_SESSION="$link_session" python3 -c 'import os,hmac,hashlib; parts=["bdspro.credential.link.v1",os.environ["BDS_LINK_ID"],os.environ["BDS_LINK_ACCOUNT"],os.environ["BDS_LINK_SESSION"],os.environ["BDS_LINK_KIND"],os.environ["BDS_LINK_DEST"],os.environ["BDS_LINK_CODE"]]; print(hmac.new(os.environ["BDS_TEST_TOKEN_KEY"].encode(),"\x00".join(parts).encode(),hashlib.sha256).hexdigest())')"
  psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -v id="$challenge" -v account="$setup_account" -v session="$link_session" -v kind="$kind" -v dest="$destination" -v mac="$mac" <<'SQL'
DELETE FROM credential_link_challenges WHERE account_id=:account AND kind=:'kind';
INSERT INTO credential_link_challenges(id,account_id,session_id,kind,destination,code_mac,expires_at,resend_after)
VALUES (:'id',:account,:'session',:'kind',:'dest',decode(:'mac','hex'),now()+interval '5 minutes',now()+interval '60 seconds');
SQL
}
link_conflict="00000000-0000-4000-8000-000000000301"
link_email="00000000-0000-4000-8000-000000000302"
link_phone="00000000-0000-4000-8000-000000000303"
seed_contact_link email "team-demo-owner@example.invalid" "$link_conflict" "12345678"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $link_access" -d "$(verify_body "$link_conflict" 12345678)" "$BASE/me/email/verify")"
assert_status "$status" 409
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT email FROM accounts WHERE id=$setup_account")" = "setup-google@example.invalid"
seed_contact_link email "linked-new@example.invalid" "$link_email" "12345678"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $member_token" -d "$(verify_body "$link_email" 12345678)" "$BASE/me/email/verify")"
assert_status "$status" 400
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $link_access" -d "$(verify_body "$link_email" 00000000)" "$BASE/me/email/verify")"
assert_status "$status" 400
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT failed_attempts FROM credential_link_challenges WHERE id='$link_email'")" = 1
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $link_access" -d "$(verify_body "$link_email" 12345678)" "$BASE/me/email/verify")"
assert_status "$status" 200
link_next="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["user"]["email"]=="linked-new@example.invalid" and d["user"]["credentials"]["email_verified"] and d["user"]["id"]==sys.argv[2]; print(d["access_token"])' "$response" "$setup_account")"
status="$(call_api -H "Authorization: Bearer $link_access" "$BASE/me")"
assert_status "$status" 401
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $link_next" -d "$(verify_body "$link_email" 12345678)" "$BASE/me/email/verify")"
assert_status "$status" 400
link_access="$link_next"
link_session="$(session_from_token "$link_access")"
seed_contact_link phone "+12025550127" "$link_phone" "123456"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $link_access" -d "$(verify_body "$link_phone" 123456)" "$BASE/me/phone/verify")"
assert_status "$status" 200
python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["user"]["id"]==sys.argv[2] and d["user"]["phone"]=="+12025550127" and d["user"]["credentials"]["phone_verified"] and d["user"]["credentials"]["password_set"] and d["user"]["credentials"]["google_linked"]' "$response" "$setup_account"
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM credential_link_challenges WHERE account_id=$setup_account")" = 0
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM audit_events WHERE actor_account_id=$setup_account AND action='auth.credential.linked'")" = 2
status="$(call_api -X POST "${browser_headers[@]}" -d '{"identifier":"linked-new@example.invalid","password":"disposable-setup-password"}' "$login_url")"
assert_status "$status" 200
python3 -c 'import json,sys; assert json.load(open(sys.argv[1]))["user"]["id"]==sys.argv[2]' "$response" "$setup_account"
echo "Contact link ownership conflict, cross-account denial, persistent failed attempt, email replacement, replay, phone attachment and audit: passed"

step_access="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["access_token"])' "$response")"
step_session="$(session_from_token "$step_access")"
seed_google_intent() {
  local subject="$1" account="$2" session="$3" purpose="$4" action="$5" age="$6"
  seed_google_result "$subject" "unused-provider@example.invalid" false
  local digest
  digest="$(python3 -c 'import hashlib,sys; print(hashlib.sha256(sys.argv[1].encode()).hexdigest())' "$google_browser")"
  psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -v browser="$digest" -v account="$account" -v session="$session" -v purpose="$purpose" -v action="$action" -v age="$age" <<'SQL'
UPDATE google_auth_flows SET auth_time=extract(epoch FROM clock_timestamp())::bigint-:age
WHERE browser_digest=decode(:'browser','hex');
INSERT INTO google_authenticated_intents(browser_digest,account_id,session_id,purpose,action)
VALUES (decode(:'browser','hex'),:account,:'session',:'purpose',:'action');
SQL
}
step_url="$BASE/me/reauth/google/result"
link_url="$BASE/me/google/result"
seed_google_intent "disposable-setup-subject" "$setup_account" "$step_session" reauth sessions.revoke_all 601
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" -H "Authorization: Bearer $step_access" "$step_url")"
assert_status "$status" 403
seed_google_intent "unlinked-subject" "$setup_account" "$step_session" reauth sessions.revoke_all 0
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" -H "Authorization: Bearer $step_access" "$step_url")"
assert_status "$status" 403
seed_google_intent "disposable-setup-subject" "$setup_account" "$step_session" reauth sessions.revoke_all 0
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" "$google_result")"
assert_status "$status" 400
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" -H "Authorization: Bearer $member_token" "$step_url")"
assert_status "$status" 403
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" -H "Authorization: Bearer $step_access" "$link_url")"
assert_status "$status" 400
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" -H "Authorization: Bearer $step_access" "$step_url")"
assert_status "$status" 200
step_proof="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["action"]=="sessions.revoke_all" and 0<d["expires_in"]<=600; print(d["reauthentication_proof"])' "$response")"
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" -H "Authorization: Bearer $step_access" "$step_url")"
assert_status "$status" 403
status="$(call_api -X DELETE "${browser_headers[@]}" -H "Authorization: Bearer $step_access" -d "$(proof_body "$step_proof")" "$BASE/me/phone")"
assert_status "$status" 403
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $step_access" -d "$(proof_body "$step_proof")" "$BASE/me/sessions/revoke-all")"
assert_status "$status" 204
status="$(call_api -H "Authorization: Bearer $step_access" "$BASE/me")"
assert_status "$status" 401
status="$(call_api -X POST "${browser_headers[@]}" -d '{"identifier":"team-demo-owner@example.invalid","password":"disposable-recovered-password"}' "$login_url")"
assert_status "$status" 200
owner_link_access="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["access_token"])' "$response")"
owner_link_session="$(session_from_token "$owner_link_access")"
seed_google_intent "disposable-setup-subject" "$owner_id" "$owner_link_session" link google.link 0
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" -H "Authorization: Bearer $owner_link_access" "$link_url")"
assert_status "$status" 409
seed_google_intent "disposable-new-owner-subject" "$owner_id" "$owner_link_session" link google.link 0
status="$(call_api -X POST "${browser_headers[@]}" -b "$google_cookie" -H "Authorization: Bearer $owner_link_access" "$link_url")"
assert_status "$status" 200
new_owner_access="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["user"]["id"]==sys.argv[2] and d["user"]["email"]=="team-demo-owner@example.invalid" and d["user"]["credentials"]["google_linked"]; print(d["access_token"])' "$response" "$owner_id")"
status="$(call_api -H "Authorization: Bearer $owner_link_access" "$BASE/me")"
assert_status "$status" 401
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM audit_events WHERE actor_account_id=$owner_id AND action='auth.credential.linked' AND target_type='google'")" = 1
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM google_authenticated_intents WHERE account_id=$owner_id")" = 0
echo "Google intent isolation, stale and wrong subject, one-use step-up, scoped revoke-all, ownership conflict and explicit link: passed"

org_owner_access="$new_owner_access"
org_owner_person="$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT person_id FROM accounts WHERE id=$owner_id")"
org_member_person="$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT person_id FROM accounts WHERE id=$member_id")"
status="$(call_api -X POST "${browser_headers[@]}" -d '{"name":"Unauthorized"}' "$BASE/orgs")"
assert_status "$status" 401
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $org_owner_access" -d '{"name":"  "}' "$BASE/orgs")"
assert_status "$status" 400
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $org_owner_access" -d '{"name":"Lifecycle Organization"}' "$BASE/orgs")"
assert_status "$status" 201
lifecycle_org="$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["role"]=="owner" and d["name"]=="Lifecycle Organization" and isinstance(d["id"],str); print(d["id"])' "$response")"
lifecycle_url="$BASE/orgs/$lifecycle_org"
status="$(call_api -H "Authorization: Bearer $org_owner_access" "$BASE/orgs")"
assert_status "$status" 200
python3 -c 'import json,sys; assert any(x["id"]==sys.argv[2] and x["role"]=="owner" for x in json.load(open(sys.argv[1]))["items"])' "$response" "$lifecycle_org"
status="$(call_api -H "Authorization: Bearer $member_token" "$lifecycle_url")"
assert_status "$status" 404
status="$(call_api -X PATCH "${browser_headers[@]}" -H "Authorization: Bearer $org_owner_access" -d '{"name":"Updated Organization"}' "$lifecycle_url")"
assert_status "$status" 200
python3 -c 'import json,sys; assert json.load(open(sys.argv[1]))["name"]=="Updated Organization"' "$response"
org_member_body="$(python3 -c 'import json,sys; print(json.dumps({"person_id":sys.argv[1]}))' "$org_member_person")"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $org_owner_access" -d "$org_member_body" "$lifecycle_url/members")"
assert_status "$status" 201
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $org_owner_access" -d "$org_member_body" "$lifecycle_url/members")"
assert_status "$status" 409
status="$(call_api -H "Authorization: Bearer $member_token" "$lifecycle_url")"
assert_status "$status" 200
status="$(call_api -H "Authorization: Bearer $member_token" "$lifecycle_url/members")"
assert_status "$status" 403
status="$(call_api -H "Authorization: Bearer $org_owner_access" "$lifecycle_url/members")"
assert_status "$status" 200
python3 -c 'import json,sys; assert len(json.load(open(sys.argv[1]))["items"])==2' "$response"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $org_owner_access" "$lifecycle_url/leave")"
assert_status "$status" 403
status="$(call_api -X DELETE "${browser_headers[@]}" -H "Authorization: Bearer $org_owner_access" "$lifecycle_url/owners/$org_owner_person")"
assert_status "$status" 403
status="$(call_api -X PUT "${browser_headers[@]}" -H "Authorization: Bearer $org_owner_access" "$lifecycle_url/owners/$org_member_person")"
assert_status "$status" 200
status="$(call_api -X DELETE "${browser_headers[@]}" -H "Authorization: Bearer $org_owner_access" "$lifecycle_url/owners/$org_member_person")"
assert_status "$status" 204
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $org_owner_access" -d "$(reauth_body organization.owner.transfer)" "$reauth_url")"
assert_status "$status" 200
transfer_proof="$(reauth_proof)"
transfer_body="$(python3 -c 'import json,sys; print(json.dumps({"person_id":sys.argv[1],"reauthentication_proof":sys.argv[2]}))' "$org_member_person" "$transfer_proof")"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $org_owner_access" -d "$transfer_body" "$lifecycle_url/owners/transfer")"
assert_status "$status" 204
status="$(call_api -H "Authorization: Bearer $org_owner_access" "$lifecycle_url")"
assert_status "$status" 200
python3 -c 'import json,sys; assert json.load(open(sys.argv[1]))["role"]=="member"' "$response"
status="$(call_api -X PATCH "${browser_headers[@]}" -H "Authorization: Bearer $org_owner_access" -d '{"name":"Stale owner"}' "$lifecycle_url")"
assert_status "$status" 403
status="$(call_api -H "Authorization: Bearer $member_token" "$lifecycle_url")"
assert_status "$status" 200
python3 -c 'import json,sys; assert json.load(open(sys.argv[1]))["role"]=="owner"' "$response"
lifecycle_team="$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "INSERT INTO teams(organization_id,name,description) VALUES ($lifecycle_org,'Lifecycle Team','') RETURNING id")"
psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -c "INSERT INTO team_members(team_id,person_id) VALUES ($lifecycle_team,$org_owner_person)"
status="$(call_api -H "Authorization: Bearer $org_owner_access" "$lifecycle_url/teams/$lifecycle_team")"
assert_status "$status" 200
status="$(call_api -X DELETE "${browser_headers[@]}" -H "Authorization: Bearer $member_token" "$lifecycle_url/members/$org_owner_person")"
assert_status "$status" 204
status="$(call_api -H "Authorization: Bearer $org_owner_access" "$lifecycle_url/teams/$lifecycle_team")"
assert_status "$status" 404
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM team_members WHERE team_id=$lifecycle_team AND person_id=$org_owner_person")" = 0
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM audit_events WHERE scope_type='organization' AND scope_ref='$lifecycle_org' AND action IN ('organization.owner.promoted','organization.owner.demoted','organization.owner.transferred','organization.member.removed')")" = 4
echo "Organization create, membership, last owner, transfer, stale authority, dependent Team removal and audit: passed"

team_owner_access="$org_owner_access"
team_root="$BASE/orgs/$org_id/teams/$team_id"
team_member_body="$(python3 -c 'import json,sys; print(json.dumps({"person_id":sys.argv[1]}))' "$org_member_person")"
team_outsider_person="$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT person_id FROM accounts WHERE id=$setup_account")"
team_outsider_body="$(python3 -c 'import json,sys; print(json.dumps({"person_id":sys.argv[1]}))' "$team_outsider_person")"
status="$(call_api -H "Authorization: Bearer $member_token" "$team_root/members")"
assert_status "$status" 403
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $member_token" -d "$team_member_body" "$team_root/members")"
assert_status "$status" 403
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $team_owner_access" -d "$team_outsider_body" "$team_root/members")"
assert_status "$status" 404
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $team_owner_access" -d "$team_member_body" "$team_root/members")"
assert_status "$status" 201
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $team_owner_access" -d "$team_member_body" "$team_root/members")"
assert_status "$status" 409
status="$(call_api -H "Authorization: Bearer $team_owner_access" "$team_root/members?page=1&limit=1")"
assert_status "$status" 200
python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["total"]==1 and d["items"][0]["person_id"]==sys.argv[2]' "$response" "$org_member_person"
status="$(call_api -H "Authorization: Bearer $member_token" "$team_root/members/me")"
assert_status "$status" 200
status="$(call_api -H "Authorization: Bearer $team_owner_access" "$BASE/orgs/$org_id/teams/$lifecycle_team/members")"
assert_status "$status" 404
status="$(call_api -X DELETE "${browser_headers[@]}" -H "Authorization: Bearer $team_owner_access" "$team_root/members/$org_member_person")"
assert_status "$status" 204
status="$(call_api -H "Authorization: Bearer $member_token" "$team_root/members/me")"
assert_status "$status" 404
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $team_owner_access" -d '{"recipient":"team-demo-member@example.invalid","expires_in":300}' "$team_root/invites")"
assert_status "$status" 503

seed_team_invite() {
  local mode="$1" expiry="$2"
  local digest membership
  read -r invite_id invite_token digest <<< "$(python3 -c 'import uuid,secrets,hashlib; t=secrets.token_urlsafe(32); print(uuid.uuid4(),t,hashlib.sha256(t.encode()).hexdigest())')"
  membership="$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT id FROM organization_members WHERE organization_id=$org_id AND person_id=$org_member_person AND ended_at IS NULL")"
  psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -v id="$invite_id" -v team="$team_id" -v owner="$owner_id" -v recipient="$member_id" -v membership="$membership" -v digest="$digest" -v expiry="$expiry" -v mode="$mode" <<'SQL'
UPDATE team_invites SET revoked_at=clock_timestamp()
WHERE team_id=:team AND accepted_at IS NULL AND revoked_at IS NULL;
INSERT INTO team_invites(id,team_id,invited_email,invited_by_account_id,token_hash,expires_at,recipient_account_id,recipient_membership_id)
VALUES (:'id',:team,'team-demo-member@example.invalid',:owner,decode(:'digest','hex'),clock_timestamp()+(:'expiry')::interval,
CASE WHEN :'mode'='legacy' THEN NULL ELSE :recipient END,
CASE WHEN :'mode'='legacy' THEN NULL ELSE :membership END);
SQL
}
invite_body() { python3 -c 'import json,sys; print(json.dumps({"invite_token":sys.argv[1]}))' "$1"; }
accept_url="$BASE/team-invites/accept"
seed_team_invite bound "-1 second"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $member_token" -d "$(invite_body "$invite_token")" "$accept_url")"
assert_status "$status" 400
seed_team_invite legacy "10 minutes"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $member_token" -d "$(invite_body "$invite_token")" "$accept_url")"
assert_status "$status" 400
seed_team_invite bound "10 minutes"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $team_owner_access" -d "$(invite_body "$invite_token")" "$accept_url")"
assert_status "$status" 400
status="$(call_api -X DELETE "${browser_headers[@]}" -H "Authorization: Bearer $team_owner_access" "$team_root/invites/$invite_id")"
assert_status "$status" 204
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $member_token" -d "$(invite_body "$invite_token")" "$accept_url")"
assert_status "$status" 400

seed_team_invite bound "10 minutes"
status="$(call_api -X DELETE "${browser_headers[@]}" -H "Authorization: Bearer $team_owner_access" "$BASE/orgs/$org_id/members/$org_member_person")"
assert_status "$status" 204
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $team_owner_access" -d "$team_member_body" "$BASE/orgs/$org_id/members")"
assert_status "$status" 201
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $member_token" -d "$(invite_body "$invite_token")" "$accept_url")"
assert_status "$status" 400

seed_team_invite bound "10 minutes"
race_dir="$(mktemp -d)"
race_body="$(invite_body "$invite_token")"
curl -sS -o "$race_dir/a.json" -w '%{http_code}' -X POST "${browser_headers[@]}" -H "Authorization: Bearer $member_token" -d "$race_body" "$accept_url" > "$race_dir/a.status" &
race_a=$!
curl -sS -o "$race_dir/b.json" -w '%{http_code}' -X POST "${browser_headers[@]}" -H "Authorization: Bearer $member_token" -d "$race_body" "$accept_url" > "$race_dir/b.status" &
race_b=$!
wait "$race_a"
wait "$race_b"
race_status="$(printf '%s\n' "$(cat "$race_dir/a.status")" "$(cat "$race_dir/b.status")" | sort | paste -sd, -)"
if [[ "$race_status" != "200,400" ]]; then
  echo "Unexpected concurrent acceptance statuses: $race_status" >&2
  cat "$race_dir/a.json" "$race_dir/b.json" >&2
  rm -rf "$race_dir"
  exit 1
fi
rm -rf "$race_dir"
status="$(call_api -X POST "${browser_headers[@]}" -H "Authorization: Bearer $member_token" -d "$race_body" "$accept_url")"
assert_status "$status" 400
status="$(call_api -H "Authorization: Bearer $member_token" "$team_root/members/me")"
assert_status "$status" 200
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM team_members WHERE team_id=$team_id AND person_id=$org_member_person")" = 1
test "$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM audit_events WHERE action='team.invite.accepted' AND target_ref='$invite_id'")" = 1
echo "Team membership, roster privacy, cross-org denial, invite expiry/revoke/rejoin/replay and concurrent acceptance: passed"
