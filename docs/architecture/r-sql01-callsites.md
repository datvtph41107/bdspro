# R-SQL01 — Database executor call-site inventory

**Evidence state:** Source-verified at `refactor/r-sql01-core-contract` after `55f44e2`; stable line ranges below refer to production source unchanged from `dev` commit `abe04e46269632ef671a92bab00319b52696e8f5`. This is a **bounded source audit**, not production telemetry. Search scope: Go production callers of `db.GetQueries`, `db.GetEngine`, `db.WithTx`, `db.WithTx2`, `db.New`. Generated calls and comments are treated separately.

## Explicit production call graph

| Caller path | Call lines | Mechanism / ownership |
| --- | --- | --- |
| [services/org/team.go](../../services/org/team.go#L31-L120) | 36, 61: `GetQueries(ctx)`; 91, 117: `WithTx2(ctx,...)`; 92, 118: `GetQueries(txCtx)` | Non-transaction reads and two transactional writes; callbacks use derived transaction context correctly in inspected lines |
| [services/org/organization.go](../../services/org/organization.go#L15) | 15: `GetQueries(ctx)` | Uses caller context; no local explicit Tx at this call |
| [services/auth/access.go](../../services/auth/access.go#L25) | 25: `GetQueries(ctx)` | Auth DB read; transaction ownership follows supplied caller context |
| [services/jobs/team_invite.go](../../services/jobs/team_invite.go#L43) | 43: `GetQueries(ctx)` | Job DB read; preserve job-service behavior |
| [services/jobs/verify_email.go](../../services/jobs/verify_email.go#L42) | 42: `GetQueries(ctx)` | Job DB read |
| [services/jobs/password_reset.go](../../services/jobs/password_reset.go#L67-L72) | 67: `WithTx`; 70: `GetQueries(txCtx)` | Transactional credential recovery workflow |
| [services/worker/outbox.go](../../services/worker/outbox.go#L52-L54) | 52: `WithTx`; 53: `GetQueries(txCtx)` | Separate F40 ownership; do not refactor the queue as part of R-SQL01 |

**Totals at inspected production call sites:** `db.GetQueries`: 10 invocations across 7 files. `db.WithTx`: 2 invocations in 2 files. `db.WithTx2`: 2 invocations in 1 file. `db.GetEngine`: no direct production caller found in this set. `db.New`: no direct production caller found in this set. Their implementations, internal initialization and generated usage still matter; absence of a production caller does **not** automatically authorize removing a public API.

## Core-only / generated mechanisms

- [models/db/context.go](../../models/db/context.go): `GetQueries` and `GetEngine` both choose transaction scope or the process engine depending on `ctx`; `WithTx` installs `scope.queries = New(tx)`.
- [models/db/engine_init.go](../../models/db/engine_init.go): process engine publishes `queries: New(pool)`.
- [models/db/engine.go](../../models/db/engine.go): `db.New(conn)` delegates to `sqlc.New(conn)`.
- [models/db/sqlc/db.gen.go](../../models/db/sqlc/db.gen.go): SQLC generates `New(DBTX)` and `Queries.WithTx(pgx.Tx)`. Generator-owned; never edit as a refactor shortcut.

## Findings and counterarguments

**F1:** `GetEngine(ctx)` is exported and allows raw SQL through `DBTX`, but the inspected production callers use SQLC `GetQueries`. This **supports reviewing** raw-executor exposure; it does not prove external consumers or tests never use it.

**F2:** Existing transactional call sites inspected here consistently use `txCtx`, but contract correctness depends on all future callers doing so. `context_postgres_test.go` targets the deliberately wrong-context path.

**F3:** A single `CreateTeamForOwner` SQL statement is wrapped by `WithTx2` in [team.go:80–108](../../services/org/team.go#L80-L108) for *future* audit/outbox atomicity per source comment. Removal could save transaction overhead, but only after verifying enclosing caller semantics and SQL concurrency behavior; no change is authorized yet.

**F4:** `models/db/sqlc_types.gen.go` re-exports generated types via aliases; these do not add data copying. An API-size/import-cost audit is required before deletion.

**F5:** Existing `Makefile` and `.github/workflows/test.yml` already provide SQLC v1.30.0 drift checks and disposable migrations. Do not duplicate those controls.

## Next discriminating work

1. Complete and observe PostgreSQL wrong-context reproduction, and sticky nested rollback in CI.
2. Compare context-bound callback vs explicit transaction-bound Queries with the same behavior tests.
3. Measure source call-site changes and operational overhead; only then accept an API shape in the ADR.
4. Audit type aliases and `emit_interface` consumers with compile-only experiments before changing generation.

**Scope guard:** The outbox entry exists only as a database caller; it is not authorization to change F40 worker/queue behavior.
