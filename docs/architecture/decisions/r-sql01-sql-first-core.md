# R-SQL01 — SQL-first DB Core: authority, generation and lifecycle

**Status:** PROPOSED (not ACCEPTED, not a claim of production verification)  
**Workstream:** R-SQL01, independent of F40 event/queue/notification migration  
**Branch:** `refactor/r-sql01-core-contract`, based on `dev` at `abe04e46269632ef671a92bab00319b52696e8f5`  
**Review owner:** BDSPro maintainers  
**Evidence baseline:** BDSPro `434df714ae0adde03e1246be3b86665a05d8058a` (rechecked on `dev` before branch creation; source paths listed below).

## Fundamental problem

A SQL-first application needs one documented authority for each of: schema evolution, generated-query ownership, connection/pool lifetime, transaction lifetime, error interpretation and observability. BDSPro currently offers both context-selected SQLC queries and raw `DBTX`, plus generated `Queries.WithTx`, `db.New`, and process-global engine state. Those APIs overlap in *how callers select an executor*, even though not all of their responsibilities are redundant.

The question is **not** whether SQLC, pgx or `WithTx` exists. It is whether a business operation can accidentally bypass the transaction or query conventions, and whether extra abstractions justify their maintenance and misuse costs.

## Evidence: observed code, not architectural approval

| Repo and pinned source | Observed mechanism | What this does / does not prove |
| --- | --- | --- |
| [BDSPro context.go:50–105](https://github.com/datvtph41107/bdspro/blob/434df714ae0adde03e1246be3b86665a05d8058a/models/db/context.go#L50-L105) | `GetEngine(ctx)` exposes native `DBTX`; `GetQueries(ctx)` chooses scope-bound `Queries` when context contains scope, otherwise global pool-bound `Queries`. | An outer context, mistakenly reused inside a transaction callback, can select the pool. Source-level risk, **not** a reproduced production incident. |
| [BDSPro context.go:118–204](https://github.com/datvtph41107/bdspro/blob/434df714ae0adde03e1246be3b86665a05d8058a/models/db/context.go#L118-L204) | `WithTx` starts pgx tx, installs context scope and guards nested failures with sticky rollback-only; `WithTx2` adds typed result. | Preserves useful nested-failure semantics; does not prove context propagation is optimal. |
| [BDSPro generated db.gen.go:14–32](https://github.com/datvtph41107/bdspro/blob/434df714ae0adde03e1246be3b86665a05d8058a/models/db/sqlc/db.gen.go#L14-L32) | SQLC `DBTX`, `New(db)`, `Queries.WithTx(tx)` are already generated. | Binding a query executor does not itself require an additional custom store or transaction framework. |
| [BDSPro engine_init.go:11–95](https://github.com/datvtph41107/bdspro/blob/434df714ae0adde03e1246be3b86665a05d8058a/models/db/engine_init.go#L11-L95) | Process-global engine publishes `pool` plus `Queries`, uses graceful cleanup, and rejects reinit after `CloseEngine`. | Defines process ownership; multi-instance testability and close/reinitialization tradeoffs remain open. |
| [BDSPro sqlc.yaml](https://github.com/datvtph41107/bdspro/blob/434df714ae0adde03e1246be3b86665a05d8058a/sqlc.yaml) / [Makefile:30–45](https://github.com/datvtph41107/bdspro/blob/434df714ae0adde03e1246be3b86665a05d8058a/Makefile#L30-L45) | SQLC v1.30.0, pgx/v5, generated output in `models/db/sqlc` plus `tools/dbalias`; CI runs `sqlc-check`. | Existing generation process should be evaluated before replacing or duplicating it. |
| [SimpleBank exec_tx.go:9–25](https://github.com/techschool/simplebank/blob/97f000fe58ad01a0774179ffa8884ac7784cf263/db/sqlc/exec_tx.go#L9-L25) | Transaction callback receives `*Queries` bound by `New(tx)`. | Explicit executor reduces dependence on forwarding the correct transaction context, but has no equivalent built-in nested rollback-only policy. |
| [Coder db.go:160–243](https://github.com/coder/coder/blob/da0bd0750f8501d25cc9555b8de7ba1d6fc000ae/coderd/database/db.go#L160-L243) | `Store.InTx` passes transaction-bound Store and implements retry for serializable failures. | Operational policy is useful; Coder uses sqlx/database/sql in this code and its wide Store is not the default BDSPro target. |
| [Coder querymetrics.go:34–67](https://github.com/coder/coder/blob/da0bd0750f8501d25cc9555b8de7ba1d6fc000ae/coderd/database/dbmetrics/querymetrics.go#L34-L67) | Generated query wrappers emit labeled histograms/counters. | Name-level SQLC metrics need a name-mapping/instrumentation mechanism; pgx tracing alone does not supply Go method identity. |
| [River AGENTS.md:15–27](https://github.com/riverqueue/river/blob/1f19003d1b65e32206f18bd73b222d61b1773972/AGENTS.md#L15-L27) / [pgx driver](https://github.com/riverqueue/river/blob/1f19003d1b65e32206f18bd73b222d61b1773972/riverdriver/riverpgxv5/river_pgx_v5_driver.go#L117-L142) | SQLC-first query discipline, explicit pgx transaction-bound executor and conformance tests. | Driver abstraction is appropriate for River's library needs, not automatically for a single-database BDSPro monolith. |
| [Gitea context.go:110–175](https://github.com/go-gitea/gitea/blob/0c1ebb630ead184a007ddebd02eb8eae6e70352e/models/db/context.go#L110-L175) | Context-bound `WithTx` and compatibility-era `TxContext`; nested error closes shared session. | An API can coexist for compatibility without being recommended for new code; Gitea's XORM mechanism must not be copied to pgx. |

## Target invariants (proposal)

1. **Schema authority:** `db/migration/*.sql` with go-migrate is the only versioned schema authority. No ORM automigrate.
2. **SQL access:** `db/query/*.sql` + SQLC is the default business query path; native pgx execution requires an explicit exceptional use case and review.
3. **Executor ownership:** every transactional operation must use the executor bound to its intended transaction. Never silently swap to the process pool inside the same operation.
4. **Transaction ownership:** exactly one outermost layer commits/rolls back. Nested failure cannot be ignored and still result in success; preserve sticky rollback-only behavior or an equally strong contract.
5. **Resource ownership:** app boot initializes the pool, owns its cleanup and exposes actionable readiness errors. Migrations have their own CLI/bounded lifecycle even if code shares a package.
6. **Diagnostics:** preserve SQLSTATE and error wrapping; use bounded labels, redact arguments, and define mapping from named SQLC query to traces/metrics before claiming query-level visibility.
7. **Developer workflow:** generated source is reproducible, committed, never hand-edited, and `sqlc-check`/Go tests must pass on final PR HEAD.

## Alternatives and failure cases

**A — Keep context-bound `WithTx/GetQueries` unchanged.** Lowest migration cost; existing nested-failure semantics. Risk: passing original context to `GetQueries` resolves global pool. Test this against a disposable PostgreSQL schema before accepting it.

**B — Explicit `*Queries` transaction callback (SimpleBank).** Small API, visible executor provenance, minimal dependency surface. Costs: existing service signatures change; nested rollback-only and composition must be redesigned/tested.

**C — Explicit transaction scope with `Queries()` + sticky failure.** Promising compromise; introduces scope state and threading through business calls. Reject if it becomes a gratuitous `UnitOfWork`/Store framework or duplicates generated SQLC.

**D — Coder-wide Store / River-wide driver interfaces.** Useful in respective upstream contexts; not justified for single pgx + SQLC backend without concrete additional needs.

**Counterexample to A:** inside `WithTx(ctx, func(txCtx ...){...})`, `GetQueries(ctx)` returns pool queries while `GetQueries(txCtx)` returns transaction queries; a write performed by the former may commit outside the outer rollback. This is reasoned from the function's branch conditions, **not yet a run test**.

**Counterexample to naïve B:** nested callback returns an error that parent swallows; an explicit `*Queries` alone cannot enforce rollback-only. Dropping BDSPro's `transactionScope.fail` without replacement would remove an existing safety invariant.

## Provisional direction — not yet an accepted DB Core API

- Retain PostgreSQL/pgx v5/SQLC/go-migrate stack.
- Treat present BDSPro behavior as a baseline to **challenge**, not an authority.
- Prefer evaluating explicit transaction-bound executor as design B/C; do not migrate call sites until PostgreSQL integration tests prove equivalence or desired intentional differences.
- Keep SQLC's generated `DBTX`/`Queries` rather than implementing an ORM or a generic repository layer.
- Audit `GetEngine`, `db.New`, `sqlc_types.gen.go` and `emit_interface` consumers before removing/changing them.
- No F40 event/worker/outbox migration in R-SQL01. Outbox remains an optional failure scenario only.

## Verification and approval gates

See [R-SQL01 task board](../r-sql01-task-board.md). Each implementation task needs a before/after file list, executable tests or explicitly stated blockers, and final-head CI evidence. A passing test establishes tested behavior only; it does not imply architectural acceptance or performance improvement.

**Acceptance required:** Maintainer review of transaction authority choice after comparative PostgreSQL tests. Do not label an unselected API or speculative benchmark `ACCEPTED`/`VERIFIED`.
