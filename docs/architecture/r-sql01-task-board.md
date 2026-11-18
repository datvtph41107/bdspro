# R-SQL01 — SQL-first DB Core execution board

**State:** IN PROGRESS. Independent from F40 event/queue/notification.
**Base:** `dev` at `abe04e46269632ef671a92bab00319b52696e8f5`.
**ADR:** [SQL-first DB Core — PROPOSED](decisions/r-sql01-sql-first-core.md). Architectural API choice is not accepted.
**Evidence:** [Production DB caller inventory](r-sql01-callsites.md).

## Task gates

| ID | Task | Status and evidence |
| --- | --- | --- |
| RSQL-00 | Source comparisons, alternatives and invariants | Documented in ADR; review pending |
| RSQL-01 | Nested rollback-only / cancellation / closed-scope unit regression | VERIFIED at `55f44e2` by [CI run 44262702418](https://github.com/datvtph41107/bdspro/actions/runs/44262702418) |
| RSQL-02 | PostgreSQL wrong-context executor escape and nested failure | VERIFIED at `dc53e17` / `4a58d8f` by [CI run 44263287738](https://github.com/datvtph41107/bdspro/actions/runs/44263287738) |
| RSQL-03 | Explicit transaction-bound executor comparison | Prototype in test-only `models/db/explicit_executor_experiment_test.go`; final-HEAD CI pending; no production API changes |
| RSQL-04 | Count production executor API call sites | Source-audited in [r-sql01-callsites.md](r-sql01-callsites.md), `6c4893e`; review pending |
| RSQL-05 | Inspect `emit_interface`, alias generator, imports | OPEN |
| RSQL-06 | SQLC reproducibility/failure safety | OPEN; existing CI checks SQLC v1.30.0 drift |
| RSQL-07 | SQLSTATE + named query diagnostics | OPEN |
| RSQL-08 | Pool, migration and shutdown lifecycle | OPEN |
| RSQL-09 | Choose and migrate final public DB Core API | BLOCKED; requires design acceptance and comparative evidence |
| RSQL-10 | Final report and review at final PR HEAD | OPEN |

## Existing verified infrastructure — do not duplicate without evidence

`.github/workflows/test.yml` already checks Go format, `go mod tidy -diff`, `make sqlc-check`, `go build ./...`, `go test -race ./...`, vet, migrations on disposable Postgres, Docker build and HTTP smoke. `Makefile` owns the SQLC generation entry point. Generated files are never edited by hand.

## SQLC-first authority

- Schema source: `db/migration/*.sql` with go-migrate.
- Query source: `db/query/*.sql` compiled and generated with SQLC v1.30.0.
- Generated source: `models/db/sqlc/` plus `models/db/sqlc_types.gen.go`, verified by `make sqlc-check`.
- Runtime: explicit ownership of executor/pool/transaction is under evaluation; neither `GetQueries(ctx)` nor trial explicit scope is a yet-accepted final design.
- F40 outbox and workers are a separate workstream. No behavior or migration changes here.

## Local reproduction commands

These commands require the repository Go toolchain and a **disposable** PostgreSQL database. Never use production credentials.

~~~sh
go test ./models/db -run '^TestTransactionScope' -count=1
BDS_TEST_MODE=disposable BDS_TEST_DB_URL='postgres://user:pass@127.0.0.1:5432/disposable_test?sslmode=disable' \
  go test -race ./models/db -run '^(TestTransactionPostgresExecutorEscape|TestTrialExplicitExecutorPostgres)$' -count=1
go test -race ./...
go vet ./...
make sqlc-check
~~~

The PostgreSQL tests create unique scratch tables and remove them; they do not edit business migrations or production tables.

## What the tests do not prove

- They do not show that explicit executor is faster, simpler to maintain across the full service graph, or impossible to bypass through other dependencies.
- The current API permits selecting the process pool if a caller accidentally uses the original, non-transaction context. This behavior has been reproduced under disposable Postgres.
- The experimental API is deliberately **test-only**. It is not a new exported `models/db` symbol or a declaration that future production code must use it.
- Benchmarks, package import graph, type-alias cost and production telemetry are still unmeasured.

## Acceptance

- [ ] Current PR HEAD has all required GitHub Actions passing.
- [ ] ADR accepted by maintainer before choosing a new public DB API.
- [ ] Wrong-context failure, nested rollback-only and cancellation behavior protected after API changes.
- [ ] SQLC generate drift test passes on final HEAD.
- [ ] No unrelated F40, authorization, migration or user-visible HTTP behavior changed.
- [ ] Real benchmark and change-surface measurements accompany any claim of optimization.
