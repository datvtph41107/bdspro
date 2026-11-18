# R-SQL01 — SQL-first Core execution board

**State:** IN PROGRESS, parallel to F40; never conflate the two.
**Base:** `dev` @ `abe04e46269632ef671a92bab00319b52696e8f5`.
**Decision record:** [R-SQL01 DB Core ADR](decisions/r-sql01-sql-first-core.md) — PROPOSED, not accepted.

The user asked for a dedicated PR with discrete commits and *implementation of already evidenced behavior*. This board separates implemented regression protection from future architectural work. Do not mark tasks VERIFIED without checking the relevant commit and final-HEAD CI/test results.

## Scope and order

| ID | Work | Gate / evidence | Current status |
| --- | --- | --- | --- |
| RSQL-00 | Record architecture alternatives, citations, risks, invariants | Pinned BDSPro + Gitea + SimpleBank + Coder + River code links, explicit PROPOSED status | IMPLEMENTED (documentation; review pending) |
| RSQL-01 | Protect nested rollback-only, closed scope, canceled context and result behavior | Unit tests exercising current `models/db/context.go` policy; `go test -race ./models/db` on final HEAD | IMPLEMENTED (tests added; CI pending) |
| RSQL-02 | Test wrong-context pool escape against disposable Postgres | Two distinguishable writes via pool vs tx; rollback; verify committed data from independent connection | OPEN (source-level counterexample only) |
| RSQL-03 | Compare explicit `*Queries` callback vs `TxScope` vs current `WithTx` | Same failure tests, nested semantics, call-site change-count, benchmark under identical workload | OPEN — no runtime API swap authorized |
| RSQL-04 | Inventory all `GetEngine` / `GetQueries` / `New` / `WithTx` / `WithTx2` call sites | File:line caller/callee matrix and native SQL exception rationale | OPEN |
| RSQL-05 | Audit SQLC `emit_interface`, aliases and generated ownership | Usage count, compile-time API delta, any required compatibility bridges | OPEN |
| RSQL-06 | Verify generator reproducibility and failure safety | `make sqlc-check`, pinned sqlc v1.30.0 and clean worktree; assess temp-output design before rewriting scripts | OPEN — existing CI already performs sqlc-check |
| RSQL-07 | Runtime diagnostics: SQLSTATE, named-query latency and txn result | A real HTTP/DB operation trace without leaking arguments, query-label cardinality budget | OPEN |
| RSQL-08 | Connection lifecycle, pool behavior, migrations | Startup/cleanup/reinit tests, migration dirty-state + compatibility checks in disposable DB | OPEN |
| RSQL-09 | Decide and implement API after evidence | Accepted ADR; differential PostgreSQL tests; migration plan preserving domain/security behavior | BLOCKED by RSQL-02/03/04 |
| RSQL-10 | Final acceptance report | Actual commit SHA, CI checks, test output, open questions and follow-on PR(s) | OPEN |

## Existing mechanisms that must not be reimplemented without proof

- `.github/workflows/test.yml` already runs `go mod tidy -diff`, formatting, `make sqlc-check` using SQLC v1.30.0, `go build ./...`, `go test -race ./...`, `go vet ./...`, disposable PostgreSQL migrations, Docker build and HTTP smoke.
- `Makefile` already defines `sqlc`, `sqlc-check` and migration commands.
- `models/db/context.go` already contains nested rollback-only policy. Preserve it unless a tested replacement is explicitly accepted.
- `models/db/sqlc/` and `models/db/sqlc_types.gen.go` are generator-owned. Never hand-edit them as a refactor shortcut.
- `docs/architecture/f40-*.md` govern a different workstream. No event/queue migration belongs in R-SQL01.

## Local / CI validation

~~~sh
go test ./models/db -run '^TestTransactionScope' -count=1
go test -race ./models/db
go test ./...
go vet ./...
make sqlc-check  # requires sqlc v1.30.0, changes generated files before diff check
~~~

For PostgreSQL-backed tests use disposable `BDS_TEST_MODE=disposable` and `BDS_TEST_DB_URL`; never point failure-injection or migration rollback tests at production. The local toolchain/environment must match `go.mod` and the CI image before interpreting build failures.

## Review checklist

- [ ] Evidence links and design statuses accurate
- [ ] Unit tests confirmed passing at final PR HEAD
- [ ] SQLC drift checks confirmed passing at final PR HEAD
- [ ] No changes to user-visible HTTP/security/domain behavior
- [ ] No schema migration, data reset, or outbox/F40 changes
- [ ] CI checks and open test gaps documented in PR
- [ ] Architectural acceptance decision explicitly recorded by maintainer

## Follow-up decision rule

Select the smallest implementation that satisfies executor ownership and rollback-only invariants. Do not claim explicit transaction APIs are better on performance without controlled measurements. Defer Store/driver frameworks and package moves without demonstrated cost-benefit.
