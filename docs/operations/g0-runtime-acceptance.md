# G0 — Runtime Baseline Acceptance (WSL / Docker Compose)

**Scope:** verify the **already running** BDSPro development stack, without changing application data.
**Code baseline:** `dev@902f9ea` (2026-10-24); PostgreSQL 18, migration version **5**, Redis 7 with AOF, Gin HTTP :8080, Mailpit :8025.
This is an *operational acceptance gate*, **not** proof of production readiness.

## 1. Distinguish three kinds of evidence

| Evidence | Proves | Does not prove |
| --- | --- | --- |
| GitHub Actions (PR #41/#42, success) | Tests, migrations, Go build, Linux runner HTTP smoke, vet and selected race suites | Developer's WSL containers, persisted DB contents, live Google or SMS delivery |
| `sh scripts/accept-g0.sh` on WSL | Read-only checks of the current Compose services, major DB version, migration, Redis AOF/volumes, HTTP responses, local Mailpit | New deployment works, DB backup is restorable, SMTP delivers externally, application image matches Git HEAD |
| Human review | Correct Git/volume ownership, startup/shutdown, real SMTP/browser flow where relevant, backup/restore on an **isolated** environment | An automatic security guarantee |

**Gate policy:** mark each item only after actual evidence. A passing script is necessary but not sufficient for final G0 sign-off.

## 2. Preflight — inspect before any recreate/rebuild

On WSL Ubuntu, from the existing repository:

```sh
cd ~/projects/bdspro
git status --short --branch
git fetch origin
git rev-parse HEAD
git rev-parse origin/dev
git rev-list --left-right --count HEAD...origin/dev
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Ports}}'
docker volume ls
docker compose ps
docker compose config --quiet
```

Do **not** `git reset --hard`, delete/reclone an existing worktree, or `docker compose down -v`.
If the worktree is dirty or `HEAD` differs from `origin/dev`, inspect the diff before changing source.
If pre-existing Redis or PostgreSQL containers/volumes are present, identify their owners before changing Compose.
PostgreSQL 14 files must **not** be attached directly to the PostgreSQL 18 container.
Adding a Redis volume does not migrate pending Asynq queues from an older ephemeral Redis container.

The acceptance script never runs `git fetch`, `docker compose up/down`, migrations, or writes to SQL/Redis.
It intentionally does **not** try to fix failed checks.

## 3. Run the non-destructive gate

Only after the intended Compose stack is already running:

```sh
sh scripts/accept-g0.sh
```

The command checks:

1. Local Git status and cached `origin/dev` ref (informational; no live fetch).
2. Compose config; running `postgres`, `redis`, `mailpit` and `api` services.
3. PostgreSQL major version **18**, `schema_migrations = 5, dirty=false`, and a Docker **volume** at `/var/lib/postgresql`.
4. Redis `PING`, `aof_enabled:1`, and a Docker **volume** mounted at `/data`.
5. `GET /health = 200`, unauthenticated `GET /users/me = 401`, `X-Request-ID`, and Mailpit API `200`.
6. A recent `startup.ready` log record, reported as a **warning** if the log tail has rotated past startup.

Optional for a deliberately different localhost HTTP port:

```sh
G0_HTTP_BASE_URL=http://127.0.0.1:8081 sh scripts/accept-g0.sh
```

The Mailpit API probe remains on loopback port 8025, matching this repository's Compose.
The script suppresses response bodies and does **not** print SQL credentials, bearer tokens, OTP codes, verification links or email message contents.
A non-zero exit means one or more machine-verifiable checks failed. A zero exit must **not** be presented as full G0 acceptance.

## 4. Manual acceptance after machine checks

For an intended new disposable installation only, after volume ownership is verified, the operator may choose `docker compose up -d --build`.
This is **not** part of the read-only gate: it can recreate containers or apply migrations.

Capture these observations without sharing secrets:

```sh
docker compose ps
docker compose logs --tail=200 api
docker compose exec -T postgres psql -U root -d bdspro \
  -c "SELECT version, dirty FROM schema_migrations;"
docker compose exec -T redis redis-cli INFO persistence
docker compose exec -T redis redis-cli LLEN 'asynq:{critical}:pending'
docker compose exec -T redis redis-cli ZCARD 'asynq:{critical}:scheduled'
docker compose exec -T redis redis-cli ZCARD 'asynq:{critical}:retry'
```

Expected startup sequence:
`startup.begin` → `postgres.connected` → `migration.checked` → `redis.connected` → `task.processor.started` → `http.listening` → `startup.ready`.

For graceful shutdown and `make server` colored `[INF]` output, use a **separate disposable development instance**, never interrupt a running stack with persistent queues merely to prove shutdown.

For real registration/email: follow [register-email-e2e.md](register-email-e2e.md), using a disposable test email; Mailpit only proves local SMTP capture. For database restore: restore a separately created backup into an **isolated test PostgreSQL instance** and inspect the migrated schema. Neither is exercised by `accept-g0.sh`.

## 5. G0 decision record

| Gate | Evidence expected | State until observed |
| --- | --- | --- |
| Source parity | `HEAD`, `origin/dev`, `git status`, intended deployment source | Pending WSL verification |
| Container and volume ownership | Compose services, mount paths, old volume inventory | Pending WSL verification |
| Database schema | PostgreSQL major, version 5, dirty=false | Pending WSL verification |
| Redis / worker | PING, AOF persistence, queue state reviewed | Pending WSL verification |
| HTTP ingress | Health 200, unauthenticated Me 401, server-owned request ID | Pending WSL verification |
| Process lifecycle | Startup evidence, shutdown on an isolated test instance | Pending WSL verification |
| Data recovery | Successful isolated backup/restore exercise | Pending WSL verification |
| External providers | Google browser OIDC and real SMS explicitly out of G0 scope | Separate gate G3 |

**Do not mark G0 completed solely from GitHub CI or this document.** Record the tested commit, date, environment and actual outputs, and keep local secrets out of tickets and chat.
