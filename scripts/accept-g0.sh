#!/bin/sh
# G0 local runtime acceptance: read-only checks for an already running Compose stack.
# Never starts/stops containers, changes Git refs, writes DB/Redis, or removes volumes.
# Do not probe expected 401/403/429 here: RequestMonitor increments Redis counters.
set -eu

cd "$(dirname "$0")/.."

failures=0
pass() { printf 'PASS  %s\n' "$1"; }
warn() { printf 'WARN  %s\n' "$1"; }
fail() { printf 'FAIL  %s\n' "$1"; failures=$((failures + 1)); }

for command_name in git docker curl; do
    if ! command -v "$command_name" >/dev/null 2>&1; then
        printf 'FAIL  Required command missing: %s\n' "$command_name" >&2
        exit 2
    fi
done

printf 'BDSPro G0 runtime acceptance (read-only)\n'
printf 'Git HEAD: %s\n' "$(git rev-parse --short HEAD)"
if [ -n "$(git status --porcelain)" ]; then
    warn "Git worktree has local changes; no files will be modified"
else
    pass "Git worktree clean"
fi
if remote_head=$(git rev-parse --verify refs/remotes/origin/dev 2>/dev/null); then
    if [ "$(git rev-parse HEAD)" != "$remote_head" ]; then
        warn "HEAD differs from locally cached origin/dev; verify remote before claiming parity"
    else
        pass "HEAD matches locally cached origin/dev (not a live remote check)"
    fi
else
    warn "origin/dev tracking ref unavailable; repository parity not verified"
fi

if docker compose config --quiet; then
    pass "Docker Compose configuration"
else
    fail "Docker Compose configuration"
    exit 1
fi

service_running() {
    service_id=$(docker compose ps -q "$1" 2>/dev/null || true)
    if [ -n "$service_id" ] && [ "$(docker inspect --format '{{.State.Running}}' "$service_id" 2>/dev/null || true)" = "true" ]; then
        pass "Compose service running: $1"
    else
        fail "Compose service not running: $1"
    fi
}
for service in postgres redis mailpit api; do
    service_running "$service"
done

postgres_id=$(docker compose ps -q postgres 2>/dev/null || true)
if [ -n "$postgres_id" ]; then
    pg_version=$(docker compose exec -T postgres psql -X -A -t -q -v ON_ERROR_STOP=1 -U root -d bdspro -c "SHOW server_version_num;" 2>/dev/null || true)
    case "$pg_version" in
        18????) pass "PostgreSQL major version 18" ;;
        *) fail "PostgreSQL version mismatch/unavailable (expected 18)" ;;
    esac

    migration=$(docker compose exec -T postgres psql -X -A -t -q -v ON_ERROR_STOP=1 -U root -d bdspro -c "SELECT version::text || ':' || CASE WHEN dirty THEN 'true' ELSE 'false' END FROM schema_migrations;" 2>/dev/null || true)
    if [ "$migration" = "5:false" ]; then
        pass "Migration baseline 5, dirty=false"
    else
        fail "Migration baseline mismatch/unavailable (expected 5:false)"
    fi

    pg_mount=$(docker inspect --format '{{range .Mounts}}{{if eq .Destination "/var/lib/postgresql"}}{{.Type}}{{end}}{{end}}' "$postgres_id" 2>/dev/null || true)
    if [ "$pg_mount" = "volume" ]; then
        pass "PostgreSQL 18 data directory uses a Docker volume"
    else
        fail "PostgreSQL data volume not mounted at /var/lib/postgresql"
    fi
fi

redis_id=$(docker compose ps -q redis 2>/dev/null || true)
if [ -n "$redis_id" ]; then
    redis_ping=$(docker compose exec -T redis redis-cli PING 2>/dev/null | tr -d '\r' || true)
    if [ "$redis_ping" = "PONG" ]; then
        pass "Redis PING"
    else
        fail "Redis PING"
    fi

    persistence=$(docker compose exec -T redis redis-cli --raw INFO persistence 2>/dev/null || true)
    if printf '%s\n' "$persistence" | tr -d '\r' | grep -qx 'aof_enabled:1'; then
        pass "Redis AOF enabled"
    else
        fail "Redis AOF not enabled/unavailable"
    fi

    redis_mount=$(docker inspect --format '{{range .Mounts}}{{if eq .Destination "/data"}}{{.Type}}{{end}}{{end}}' "$redis_id" 2>/dev/null || true)
    if [ "$redis_mount" = "volume" ]; then
        pass "Redis /data uses a Docker volume"
    else
        fail "Redis data volume not mounted at /data"
    fi
fi

base_url=${G0_HTTP_BASE_URL:-http://127.0.0.1:8080}
health_status=$(curl -sS --max-time 5 -o /dev/null -w '%{http_code}' "$base_url/health" 2>/dev/null || true)
if [ "$health_status" = "200" ]; then
    pass "GET /health -> 200 (HTTP liveness only)"
else
    fail "GET /health expected 200, got ${health_status:-unavailable}"
fi

headers=$(curl -sS --max-time 5 -D - -o /dev/null "$base_url/health" 2>/dev/null || true)
request_id=$(printf '%s\n' "$headers" | awk -F: 'tolower($1)=="x-request-id" {sub(/^[[:space:]]*/, "", $2); gsub(/\r/, "", $2); print $2}' | head -n 1)
if [ -n "$request_id" ]; then
    pass "HTTP server supplies X-Request-ID"
else
    fail "HTTP X-Request-ID missing"
fi

mailpit_status=$(curl -sS --max-time 5 -o /dev/null -w '%{http_code}' 'http://127.0.0.1:8025/api/v1/messages?limit=1' 2>/dev/null || true)
if [ "$mailpit_status" = "200" ]; then
    pass "Mailpit local API reachable"
else
    fail "Mailpit local API expected 200, got ${mailpit_status:-unavailable}"
fi

if docker compose logs --tail=250 api 2>/dev/null | grep -Fq 'startup.ready'; then
    pass "startup.ready observed in recent API logs"
else
    warn "startup.ready not in last 250 API log lines; inspect full lifecycle manually"
fi

printf '\n'
if [ "$failures" -eq 0 ]; then
    printf 'G0 READ-ONLY CHECKS PASSED; WSL lifecycle, backup/restore and provider E2E still require separate evidence.\n'
else
    printf 'G0 FAILED: %s check(s). No services or data were changed.\n' "$failures"
    exit 1
fi
