#!/usr/bin/env bash
# Run BDSPro's existing HTTP acceptance tests against disposable dependencies.
# Only containers named with this invocation's unique ID are removed.
set -Eeuo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
cd "$root"

for required in docker go git; do
  if ! command -v "$required" >/dev/null 2>&1; then
    echo "VERIFY_BDSPRO BLOCKED: missing $required" >&2
    exit 2
  fi
done
if ! docker info >/dev/null 2>&1; then
  echo "VERIFY_BDSPRO BLOCKED: Docker daemon unavailable" >&2
  exit 2
fi
if [[ "$(go env GOVERSION)" != go1.27.3 ]]; then
  echo "VERIFY_BDSPRO BLOCKED: expected Go 1.27.3, got $(go env GOVERSION)" >&2
  exit 2
fi

id="verify-bdspro-$(date -u +%Y%m%d%H%M%S)-$$"
pg="$id-postgres"
redis="$id-redis"
artifact="$root/data/verification/$id"
mkdir -p "$artifact"
chmod 700 "$artifact"
created_pg=0
created_redis=0
cleanup() {
  code=$?
  trap - EXIT
  if [[ "$created_redis" == 1 ]]; then docker rm -f "$redis" >/dev/null 2>&1 || true; fi
  if [[ "$created_pg" == 1 ]]; then docker rm -f "$pg" >/dev/null 2>&1 || true; fi
  echo "VERIFY_BDSPRO: result=$([[ "$code" == 0 ]] && echo PASS || echo FAIL) artifacts=$artifact"
  exit "$code"
}
trap cleanup EXIT

sha="$(git rev-parse HEAD)"
{
  printf 'sha=%s\n' "$sha"
  printf 'go=%s\n' "$(go env GOVERSION)"
  printf 'started_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf 'surface=Gin httptest + real PostgreSQL 18 and Redis 7\n'
  printf 'limit=no external HTTP listener, SMTP provider, production load or multi-instance check\n'
} > "$artifact/manifest.txt"
chmod 600 "$artifact/manifest.txt"

# -P is deliberately not used: publish only the required ports on loopback,
# with Docker selecting ephemeral host ports. Never bind over a developer stack.
docker run -d --name "$pg" --label "bdspro.verify=$id" \
  -e POSTGRES_USER=root -e POSTGRES_PASSWORD=verify_only \
  -e POSTGRES_DB=bdspro_test -p 127.0.0.1::5432 postgres:18-alpine \
  > "$artifact/postgres-container-id.txt"
created_pg=1
docker run -d --name "$redis" --label "bdspro.verify=$id" \
  -p 127.0.0.1::6379 redis:7-alpine > "$artifact/redis-container-id.txt"
created_redis=1
chmod 600 "$artifact/"*container-id.txt

pg_address="$(docker port "$pg" 5432/tcp)"
redis_address="$(docker port "$redis" 6379/tcp)"
pg_port="${pg_address##*:}"
redis_port="${redis_address##*:}"
if [[ ! "$pg_port" =~ ^[0-9]+$ || ! "$redis_port" =~ ^[0-9]+$ ]]; then
  echo "VERIFY_BDSPRO BLOCKED: unable to resolve ephemeral Docker ports" >&2
  exit 2
fi

ready=0
for attempt in {1..60}; do
  if docker exec "$pg" pg_isready -U root -d bdspro_test >/dev/null 2>&1 \
    && docker exec "$redis" redis-cli ping 2>/dev/null | grep -qx PONG; then
    ready=1
    break
  fi
  sleep 1
done
if [[ "$ready" != 1 ]]; then
  echo "VERIFY_BDSPRO BLOCKED: disposable dependencies not healthy" >&2
  exit 2
fi

# Doctor: validate DB identity, not just TCP readiness. Read-only query.
database="$(docker exec -e PGPASSWORD=verify_only "$pg" \
  psql -U root -d bdspro_test -Atqc 'select current_database()')"
if [[ "$database" != "bdspro_test" ]]; then
  echo "VERIFY_BDSPRO BLOCKED: wrong database identity" >&2
  exit 2
fi
echo "VERIFY_BDSPRO doctor=PASS db=bdspro_test redis=healthy"

export BDS_TEST_MODE=disposable
export BDS_TEST_DB_URL="postgres://root:verify_only@127.0.0.1:${pg_port}/bdspro_test?sslmode=disable"
export BDS_TEST_REDIS_ADDR="127.0.0.1:${redis_port}"
export BDS__DATABASE__DB_SOURCE="$BDS_TEST_DB_URL"

# Schema source is unchanged: use the application's own migration command.
if ! go run . --config custom/conf/app.example.ini migrate > "$artifact/migrate.log" 2>&1; then
  echo "VERIFY_BDSPRO FAIL: migration did not complete; see private migrate.log" >&2
  exit 1
fi

tests=(
  TestG12LiveSessionRevokeDeniesAccessAndRefreshHTTP
  TestOrganizationAccountMembershipRoleAndAuthorizationHTTP
  TestTeamInvitationAuthorizationAndReplayHTTP
)
pattern="^($(IFS='|'; echo "${tests[*]}"))$"
set +e
go test ./routers -run "$pattern" -count=1 -timeout=5m -v > "$artifact/drive.log" 2>&1
test_exit=$?
set -e
if [[ "$test_exit" != 0 ]]; then
  echo "VERIFY_BDSPRO FAIL: HTTP acceptance tests failed; see private drive.log" >&2
  tail -n 50 "$artifact/drive.log" >&2
  exit 1
fi
for test_name in "${tests[@]}"; do
  if ! grep -Fq -- "--- PASS: ${test_name} (" "$artifact/drive.log"; then
    echo "VERIFY_BDSPRO FAIL: $test_name did not execute and pass (skip is not pass)" >&2
    exit 1
  fi
  printf 'PASS %s\n' "$test_name" >> "$artifact/verdict.txt"
done
chmod 600 "$artifact/"*.log "$artifact/verdict.txt"
echo "VERIFY_BDSPRO PASS: 3 acceptance paths with disposable SQL/Redis and DB assertions"
