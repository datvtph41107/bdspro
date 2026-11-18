#!/usr/bin/env bash
# Disposable-only end-to-end check of password login, refresh and logout.
# Never execute against a real account database.
set -euo pipefail
umask 077

if [[ "$BDS_TEST_MODE" != "disposable" ]]; then
  echo "Auth smoke requires BDS_TEST_MODE=disposable" >&2
  exit 1
fi
DB_URL="$BDS_TEST_DB_URL"
if [[ "$DB_URL" != *"/bdspro_test?"* ]]; then
  echo "Refusing to seed any database except bdspro_test" >&2
  exit 1
fi
BASE="http://127.0.0.1:8080/api/v1"

response="$(mktemp /tmp/bdspro-auth-response-XXXXXXXX)"
headers="$(mktemp /tmp/bdspro-auth-headers-XXXXXXXX)"
first="$(mktemp /tmp/bdspro-auth-first-XXXXXXXX)"
second="$(mktemp /tmp/bdspro-auth-second-XXXXXXXX)"
request="$(mktemp /tmp/bdspro-auth-request-XXXXXXXX)"
trap 'rm -f "$response" "$headers" "$first" "$second" "$request"' EXIT

# Precomputed bcrypt fixture for "FixturePassword-2026!".
# This account exists only in CI's freshly migrated disposable DB.
account_id="$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 <<'SQL'
INSERT INTO parties DEFAULT VALUES RETURNING id AS person_id \gset
INSERT INTO persons(party_id) VALUES (:person_id);
INSERT INTO accounts(person_id, email, is_email_verified)
VALUES (:person_id, 'auth-smoke-2026@example.invalid', true)
RETURNING id AS account_id \gset
INSERT INTO password_credentials(account_id, password_hash)
VALUES (:account_id, '$2y$10$tvYSn4pjA1pTbq30PKPCXeJ6YCSqdGqFv21/2WxTkAjYdYy1G7rNu');
\echo :account_id
SQL
)"
[[ "$account_id" =~ ^[0-9]+$ ]] || { echo "Invalid auth fixture account" >&2; exit 1; }

call_api() {
  curl -sS -D "$headers" -o "$response" -w '%{http_code}' "$@"
}
assert_status() {
  if [[ "$1" != "$2" ]]; then
    echo "Expected HTTP $2, got $1" >&2
    # Avoid displaying successful token payloads in CI logs.
    exit 1
  fi
}

status="$(call_api -H 'Content-Type: application/json' \
  -d '{"email":"auth-smoke-2026@example.invalid","password":"incorrect"}' \
  "$BASE/users/login")"
assert_status "$status" 401
status="$(call_api -H 'Content-Type: application/json' \
  -d '{"email":"missing-auth-smoke@example.invalid","password":"FixturePassword-2026!"}' \
  "$BASE/users/login")"
assert_status "$status" 401
echo "Wrong and unknown password credentials: HTTP 401"

status="$(call_api -H 'Content-Type: application/json' \
  -d '{"email":"auth-smoke-2026@example.invalid","password":"FixturePassword-2026!"}' \
  "$BASE/users/login")"
assert_status "$status" 200
grep -qi '^Cache-Control: no-store' "$headers"
cp "$response" "$first"

python3 - "$first" "$account_id" <<'PY'
import json,sys,uuid
d=json.load(open(sys.argv[1]))
assert d["account_id"]==int(sys.argv[2])
uuid.UUID(d["session_id"])
assert len(d["refresh_token"])==43
assert d["access_token"].startswith("eyJ")
PY

first_session="$(python3 - "$first" <<'PY'
import json,sys
print(json.load(open(sys.argv[1]))["session_id"])
PY
)"
first_access="$(python3 - "$first" <<'PY'
import json,sys
print(json.load(open(sys.argv[1]))["access_token"])
PY
)"
first_digest="$(python3 - "$first" <<'PY'
import hashlib,json,sys
print(hashlib.sha256(json.load(open(sys.argv[1]))["refresh_token"].encode()).hexdigest())
PY
)"

stored_digest="$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c \
  "SELECT refresh_token FROM sessions WHERE id='$first_session';")"
[[ "$first_digest" == "$stored_digest" ]] || { echo "Refresh token was not stored as digest" >&2; exit 1; }
login_audit="$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c \
  "SELECT count(*) FROM audit_event WHERE action='auth.session.issued' AND actor_id=$account_id AND target_ref='$first_session';")"
[[ "$login_audit" == "1" ]] || { echo "Required session issuance audit missing" >&2; exit 1; }
echo "Password login persisted hashed refresh and required audit"

status="$(call_api -H "Authorization: Bearer $first_access" "$BASE/users/me/sessions")"
assert_status "$status" 200
python3 - "$response" "$first_session" <<'PY'
import json,sys
d=json.load(open(sys.argv[1]))
assert any(s["id"]==sys.argv[2] for s in d["items"])
PY
echo "Verified Doer can list own active sessions"

python3 - "$first" "$request" <<'PY'
import json,sys
d=json.load(open(sys.argv[1]))
with open(sys.argv[2], "w") as out:
    json.dump({"session_id":d["session_id"],"refresh_token":d["refresh_token"]},out)
PY
status="$(call_api -H 'Content-Type: application/json' --data-binary "@$request" \
  "$BASE/users/refresh")"
assert_status "$status" 200
grep -qi '^Cache-Control: no-store' "$headers"
cp "$response" "$second"

second_session="$(python3 - "$second" "$first_session" <<'PY'
import json,sys
d=json.load(open(sys.argv[1]))
assert d["session_id"] != sys.argv[2]
print(d["session_id"])
PY
)"
second_access="$(python3 - "$second" <<'PY'
import json,sys
print(json.load(open(sys.argv[1]))["access_token"])
PY
)"

status="$(call_api -H 'Content-Type: application/json' --data-binary "@$request" \
  "$BASE/users/refresh")"
assert_status "$status" 401
status="$(call_api -H "Authorization: Bearer $first_access" "$BASE/users/me/sessions")"
assert_status "$status" 401
status="$(call_api -H "Authorization: Bearer $second_access" "$BASE/users/me/sessions")"
assert_status "$status" 200
echo "Single-use refresh rotation rejects replay and revokes previous session"

status="$(call_api -X POST -H "Authorization: Bearer $second_access" \
  "$BASE/users/me/logout")"
assert_status "$status" 204
status="$(call_api -H "Authorization: Bearer $second_access" "$BASE/users/me/sessions")"
assert_status "$status" 401
revoke_audit="$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c \
  "SELECT count(*) FROM audit_event WHERE action='auth.session.revoked' AND actor_id=$account_id AND target_ref='$second_session';")"
[[ "$revoke_audit" == "1" ]] || { echo "Required session revocation audit missing" >&2; exit 1; }
active="$(psql "$DB_URL" -X -qAt -v ON_ERROR_STOP=1 -c \
  "SELECT count(*) FROM sessions WHERE account_id=$account_id AND is_blocked=false;")"
[[ "$active" == "0" ]] || { echo "Login session remained active after logout" >&2; exit 1; }
echo "Self logout: 204, access revoked, audit persisted and no active sessions"
