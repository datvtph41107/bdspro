# BDSPro — Engineering Source Contract

The four supplied Mindset documents require observation, root-cause analysis,
mechanism, impact, decision, implementation, verification and operating evidence.
The repository should not add abstractions just to imitate Laravel or SimpleBank.

## Ownership

- `main.go`: process startup, signal handling, dependency lifecycle.
- `internal/api`: Gin routes, HTTP middleware, input/output representations.
- `internal/service`: business operations, validation, authorization decisions.
- `internal/database`: sqlc-generated queries and explicitly scoped transactions.
- `db/query` and `db/migration`: editable SQL and schema source of truth.
- `internal/worker`: Asynq tasks and durable job-outbox dispatch.
- `internal/notification`: external delivery adapters.
- `doc/swagger`: versioned OpenAPI JSON and embedded Swagger UI assets
  served in development; `make swagger` tests OpenAPI route coverage and
  does not generate files. There is no `doc/statik` output.
- `docs`: engineer-facing operating and architecture documentation.

## Coding rules

No HTTP dependency in services or database. Avoid repository wrappers over SQLC.
Input structs exist only when they capture a meaningful HTTP contract.
Responses control public data rather than serializing database models.
Do not create unused interfaces or speculative framework registries.
A transaction must encompass all writes needed for one invariant, but not an
unbounded external provider call.

## Security invariants

- Phone/account uniqueness is enforced by PostgreSQL, including races.
- A bound account is rejected before issuing a new phone challenge.
- OTP failures commit their failed-attempt counter; replays cannot bind again.
- Redis prefixes, OAuth state names and task type values are persisted semantics:
  rename only through an explicit compatibility migration.
- JWT refresh-session revocation does not invalidate already issued access
  tokens immediately. Decide and test any stronger revocation requirement.
- Known example signing keys are refused outside development.

## Public HTTP error contract

The `error` field remains a backwards-compatible string; `code` is a stable
public identifier. Raw provider, database and internal error details are not
serialized. Service outages must not be reported as bad credentials.

## API documentation and operations

`doc/swagger/openapi.json` is the versioned HTTP specification.
`make swagger` **checks** OpenAPI-to-Gin route coverage; it does not
synchronize or generate assets. Embedded Swagger UI is development-only
and served by the Go `embed` package. The CSS/JS live in the embedded
assets, not an external CDN. CI verifies route coverage and runtime serving.

Production migrations must run separately before app startup. The outbox
currently enqueues to Redis while holding PostgreSQL row locks, with a
bounded timeout. At-least-once delivery and idempotent effects remain necessary.

## Acceptance

```sh
make dev-up
make dev
# In another terminal, verify /health and /swagger/openapi.json.
make check
git diff --check
make dev-down
```

`make check` is non-mutating and does not generate SQLC output. Use
`make sqlc` deliberately after editing SQL; CI checks generated-output drift.
Database rollback Make targets require an explicit confirmation flag.

CI additionally uses disposable Postgres 18 and a historical migration-upgrade
test. No production-ready claim until multi-dispatcher retry behavior,
backups/restore, provider failures and deployment rollback have been tested.
