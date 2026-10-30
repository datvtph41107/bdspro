# CODE-04 — Developer workflow and one API coding contract

## What this PR changes

This PR makes the *existing* host-Go + Docker Compose development path
reproducible and prevents ordinary commands from accidentally rolling back
the database. It does not redesign HTTP routing, change SQL, implement new
business behavior or change public API URLs.

The reference is the current `dev` branch. The separately reviewed
CODE-01 (#85), CODE-02 (#86) and CODE-03 (#87) remain draft PRs, not
implicit dependencies.

## Local owner of each process

| Owner | Started by | Ended by | Source |
| --- | --- | --- | --- |
| PostgreSQL 18 | `make dev-up` | `make dev-down` | Compose `postgres` |
| Redis 7 | `make dev-up` | `make dev-down` | Compose `redis` |
| Mailpit | `make dev-up` | `make dev-down` | Compose `mailpit` |
| Go REST process + in-process worker | `make dev` | Ctrl+C | `main.go` |

`dev-up` runs `docker compose up -d --wait postgres redis mailpit`.
Postgres and Redis have healthchecks; the Mailpit service is only verified
to be running, not SMTP-ready. `make dev` binds Go HTTP to loopback
`127.0.0.1:8080` and explicitly selects local DB, Redis and SMTP addresses.
A separate `docker compose up --build` path remains available when the API
itself must run inside a container; do not run both at once on port 8080.

The existing volume names are retained. `make dev-down` only stops services,
so it does not wipe database or queue state. Intentionally clearing local
volumes is a separate manual and destructive Docker operation.

## The real request path and source editing rules

For a current authenticated Organization update:

1. **Route:** `internal/api/routes.go` is the only Gin declaration.
   Inspect the parent group to learn which authentication and admission
   middleware applies. Do not create a second route registry.
2. **Middleware:** `internal/api/middleware/` protects cross-cutting HTTP
   ingress, determines verified AccountID and denies unsafe requests. It is
   not a substitute for checking ownership of this specific Organization.
3. **Request:** `internal/api/handler/requests.go` defines the HTTP
   fields. Gin `ShouldBindJSON` parses and validates the HTTP shape.
   Missing/invalid body fields must be rejected before database calls.
4. **Handler:** `internal/api/handler/organization.go` converts HTTP
   path/body/actor to an operation and returns HTTP success or a safe
   response error. No handwritten SQL belongs here.
5. **Service:** `internal/service/organization.go` owns the operation's
   business invariants; it cannot import Gin. Transaction choreography
   belongs to the operation owner when justified, not every CRUD action.
6. **SQLC:** `db/query/organization.sql` contains the only editable SQL.
   `internal/database/*sql.go` is generated, and PostgreSQL constraints
   enforce actual concurrency invariants. The current authorization
   mutation predicate is part of the SQL, not just the route group.
7. **Resource:** `internal/api/handler/resources.go` exposes only public
   response fields. Avoid leaking credentials, driver internals or raw
   `pgconn.PgError`.
8. **Error & log:** `internal/api/response/` writes the public error
   envelope; unexpected causes are kept private. The safe access log
   uses route templates and request IDs rather than token-bearing URLs.

A new API does **not** have to touch all eight locations. Add Request,
Service or Resource only when that part actually owns a responsibility.
Do not add an interface or repository forwarding wrapper just to satisfy
a template. Preserve correct HTTP error/status behavior and test denial
paths, not only success cases.

## A developer's smallest verified iteration

```sh
make dev-up
make dev
# another terminal:
curl -i http://127.0.0.1:8080/health
curl -i http://127.0.0.1:8080/swagger/openapi.json

# after editing:
make format-check
make swagger
make test
make test-race
make vet
make build
# or: make check
git diff --check
```

When changing `db/query/*.sql`, use pinned SQLC v1.30.0 and `make sqlc`
intentionally, then review and commit generated Go output. Never edit
generated SQLC files directly. CI checks that regeneration creates no diff.

## Safeguards and known limits

- `make migratedown` is an all-migration rollback. `make migratedown1`
  rolls back one step. Both refuse by default unless the operator sets
  `CONFIRM_MIGRATION_DOWN=yes`, but this flag does **not** inspect the
  DB URL or prove that its database is safe to destroy. Never point
  rollback commands at production from the developer Makefile.
- The normal local database is `bdspro`. Integration tests require the
  distinct `bdspro_test` and `BDS_TEST_MODE=disposable`; developers
  must not run write tests against application data.
- Go's development-mode startup currently auto-applies migrations;
  this PR intentionally does not change that behavior.
- The existing `/health` endpoint is liveness-style, not a complete
  readiness signal for every dependency; PostgreSQL/Redis outage behavior
  remains a separate runtime/failure-contract review.
- A healthy Compose service at startup does not guarantee dependency
  availability later. The API and in-process Asynq worker still share
  startup/lifecycle dependencies.
- `make swagger` validates route coverage; it does **not** generate
  OpenAPI JSON. The specification is `doc/swagger/openapi.json` and
  must be edited with endpoint/contract changes. Full schema semantic
  completeness is **not** guaranteed by the current test.
- The `Organization` name and URL are *not* accepted as the final
  company/workspace vocabulary; this PR avoids a speculative rename.

## CI evidence required

GitHub Actions must check:
- Docker Compose config and dry-run developer Make targets.
- Rollback targets reject execution without confirmation.
- `make format-check` has the same non-mutating semantics locally and in CI.
- Existing SQLC drift, migrations/upgrade preservation, Go tests,
  race detector, vet, build and HTTP runtime smoke checks still pass.

Run the workflow for the **exact final PR HEAD** before declaring PASS.
A successful CI run verifies these commands, but it does not replace
a human cold-clone test on a workstation or certify production readiness.

Reference: Docker Compose documentation for `up --wait`, `healthcheck` and
`depends_on: condition: service_healthy`.
