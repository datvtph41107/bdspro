> **Historical R01 comparison (2026-11-03):** This is not a live package
> inventory. Current BDSPro has both `models/organization` and
> `services/organization`; verify the [current architecture](README.md),
> [capability map](../engineering/capability-map.md) and source code.
> Review the [cache/queue ownership](cache-queue.md) for subsequent changes.

# R01 — Package ownership based on actual Gitea code

This document explains the *work* owned by packages, rather than trying to
reproduce Gitea's list of folders. Compared with Gitea `main`, 2026-11-03.

## Choosing a location

| What the code owns | Location | Example in BDSPro |
| --- | --- | --- |
| HTTP admission and API contract | `routers/` | `routers/common/RequestID`, API handlers and responses |
| Business operation or invariant | `services/<area>/` | `services/auth/Register`, `models/organization` |
| Persistent state and atomic changes | `models/` | `models/db` (SQLC, pgx, transactions, schema runner) |
| Reusable technical mechanism *with callers* | `modules/<mechanism>/` | logging, settings, Asynq transport, bcrypt, cryptographic randomness |
| Static rendered assets | `templates/` | auth pages, Swagger UI and OpenAPI contract |
| Build, local setup and deployment | Makefile, `.air.toml`, Docker, CI | build artifact and separate web/worker runtime |
| Cross-package acceptance tests | `tests/` | import ownership checks in `tests/architecture` |

This is not a rigid onion architecture. Gitea's modules contain some
domain-specific mechanisms, not only cross-domain utilities. A new folder
requires a *specific current consumer*, a responsibility, lifecycle and a
failure/validation test. Do not create directories merely for future features.

## Gitea module examples: what is actually implemented

| Gitea source | Existing responsibility | BDSPro decision |
| --- | --- | --- |
| `modules/setting/path.go` and `config_provider.go` | resolve work/config paths; read INI | same role in `modules/setting`, with explicit config selection |
| `modules/log/manager.go` | own event writers and close behavior | synchronous writers in `modules/log`; do not copy Gitea's async manager |
| `modules/queue/queue.go` | queue, batches, workers, retries, backend | Asynq mechanism in `modules/queue`; jobs/outbox policy remain services |
| `modules/graceful/manager.go` | shutdown phases for HTTP, SSH, installer | `cmd` signal context + runtime shutdown already exist; add only for a failed shutdown contract |
| `modules/cache/cache.go` | shared cache semantics and backends | R28 implements opt-in memory/Redis cache with explicit error policy; no security-state use |
| `modules/httpcache/httpcache.go` | ETag and HTTP Cache-Control semantics | not equivalent to Redis data cache or route middleware |
| `modules/storage/storage.go` | storage interface, local and cloud backends | wait for a real document/upload persistence requirement |
| `modules/session/session.go` | web session middleware and stores | NOT the same as BDSPro's PostgreSQL authentication sessions |
| `modules/auth/common.go` | small auth-specific helper | BDSPro `modules/auth` owns password crypto, `services/auth` owns login rules |
| `modules/secret/secret.go` | cryptographic encryption/decryption | BDSPro `modules/secret` currently owns secure random tokens only |
| `modules/structs/` | API schemas and DTOs | request and response types remain near API boundary until genuinely shared |
| `modules/validation/binding.go` | reusable web binder validation rules | Gin request validation stays at HTTP boundary; services enforce invariants |
| `modules/paginator/` | shared pagination mechanics | add after stable cursor/sort semantics appear across multiple endpoints |
| `modules/httplib/` | HTTP request and URL utilities | prefer native `net/http` until there is repeated missing behavior |
| `modules/web/router.go` | Chi router, forms and middleware composition | BDSPro uses Gin directly; do not wrap Gin with another router |
| `modules/private/internal.go` | Gitea internal API client for SSH/Git processes | no internal HTTP hop needed between BDSPro web/worker |
| `modules/repository/init.go` | Git repository labels/license/template options | NOT an ORM/SQLC Repository pattern |
| `modules/migration/options.go` | Git repository import/migration fields | NOT schema-version SQL migration |
| `modules/templates/`, `modules/public/` | layered templates and public assets | `templates/` embed already supplies assets; no frontend pipeline |
| `modules/test/`, `testlogger/` | cross-package test and logging utilities | tests close to code; create reusable fixtures only after reuse |

Gitea code:
https://github.com/go-gitea/gitea/tree/main/modules

## Three BDSPro execution paths prove the boundaries

**HTTP register**:
`routers/api/v1/user/auth.go` parses the request and invokes
`services/auth/AuthService.Register`. The service enforces password
requirements and calls `models/account.Register`. Bcrypt is in
`modules/auth`. Gin does not enter services or models.

**Async verification**:
`services/auth` creates an outbox intent.
`services/worker/OutboxDispatcher` reads and claims pending database rows.
`modules/queue` handles the Asynq enqueue/processor mechanics.
`services/worker` adapts Asynq tasks to `services/jobs` business work,
using `modules/secret` and `services/mail`.
Database commit and Redis enqueue are not atomic: at-least-once must be
validated with crash/replay tests; changing folders does not fix it.

**Schema migration**:
`cmd/migrate.go` loads only `setting.DatabaseSettings` and calls the
SQL migration runner in `models/db/migration.go`.
`db/migration/*.sql` owns schema versions. SQLC consumes schema/queries.
Gitea's `modules/migration` imports Git repositories, so its name is
not a reason to move SQL schema handling there or enable GORM AutoMigrate.

## `routers`, `models`, `services`: concrete scope

- `routers/init.go`: wire HTTP-facing dependencies, not domain rules.
- `routers/common`: shared protocol behavior (request ID, CORS).
- `routers/api`: REST route declaration, security admission, binding,
  response mapping, HTTP error contract.
- `routers/web`: Gitea renders browser pages; BDSPro currently only
  serves REST and a few embedded auth/Swagger assets, not a full web UI.
- `routers/private`: Gitea serves internal Git/SSH subprocess APIs.
  BDSPro worker uses Go services directly, not internal HTTP.
- `services/auth`: registration, sessions, OIDC/phone account business.
- `models/organization`: membership and organization rules.
- `services/jobs`: business operations invoked by task consumers.
- `services/worker`: outbox delivery semantics and task-to-business adapters.
- `services/mail`: outbound mail delivery implementation; keep its
  interface close to actual consumers.
- `models/db`: SQLC, pgxpool, transaction methods and DB readiness.
  Do not create a generic repository layer duplicating SQLC.
- `models/<domain>`: only when persistent domain ownership/transaction
  operations warrant a split. A domain name alone does not prove it.

## YAML, tooling and artifacts are different contracts

| Artifact | Ownership | Evidence required |
| --- | --- | --- |
| `sqlc.yaml` | SQL schema, queries and generated Go destination | regenerate and diff `models/db` |
| `.github/workflows/test.yml` | CI services and acceptance gates | exact-head green run; migration and build evidence |
| `docker-compose.yaml` | API/worker/DB/Redis process topology, volume and health policy | compose config, built image, health and shutdown |
| `custom/conf/app.example.ini` | config example, not runtime secrets | first clone and no-overwrite setup test |
| `.air.toml` | watcher build/restart logic | `make watch` builds `bin/bdspro`, runs web only |
| `Makefile` | developer operations, not database credential policy | same selected INI for web/worker/migrate |
| `options/label/Advanced.yaml` **in Gitea** | product label catalog | no counterpart until BDSPro has catalog consumers |
| `.golangci.yml` **in Gitea** | code quality and dependency constraints | do not blindly install Gitea's entire lint list |
| `go.mod`, `go.sum` | dependency graph and checksums | tidy + reproducible compile |
| `tests/architecture` | real repo-wide dependency boundaries | automatic Go AST import test in CI |

Notably, Gitea's `.golangci.yml` uses `depguard` to forbid direct
`gopkg.in/ini.v1` outside its configuration system. This expresses a
single-owner contract, not a folder naming preference.
Source: https://github.com/go-gitea/gitea/blob/main/.golangci.yml

Gitea Air: https://github.com/go-gitea/gitea/blob/main/.air.toml
Gitea labels: https://github.com/go-gitea/gitea/blob/main/options/label/Advanced.yaml

## New-package decision record (required during review)

For each proposed package write: consumer and call site, operation
it owns, dependencies it may use, owned state/cleanup, alternative using
native Go/existing package, unique added guarantee, failing test without
the proposed change, and removal/rollback cost.

Examples of **not yet justified**: another Gin router facade, a generic
`Repository` interface over SQLC, an unused storage provider, an empty
`routers/web`, a global DTO catalog, or a parallel queue implementation.

## R01 acceptance limit

Tests enforce a small number of specific import invariants; they do not
prove semantic separation, API parity, or distributed-system correctness.
Before merging inspect API contracts, auth guards, secret exposure, shutdown,
outbox concurrency/duplicate delivery, database configuration drift, Docker
behavior and log correlation separately.
