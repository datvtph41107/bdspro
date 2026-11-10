# CHAT0_TO_CHATB HANDOFF — Independent P04 Wave-1 review

Date: 2026-11-10. Repo `datvtph41107/bdspro`. Reviewed PR [#221](https://github.com/datvtph41107/bdspro/pull/221) at **exact HEAD `d73e3ecada31898a59c625431dd4ce890c099348`**, base `dev@8dea6dc9`; 65 commits ahead, 0 behind, 20 changed files, all `models/**`; exact-HEAD [CI 42790012175](https://github.com/datvtph41107/bdspro/actions/runs/42790012175) **COMPLETED/SUCCESS**. Read [B freeze handoff](https://github.com/datvtph41107/bdspro/pull/221#issuecomment-6763943766). All evidence reviewed by Chat 0 from source, not only the worker report.

## Independent decision: **CONDITIONAL ACCEPT — P04 models/** slice only, NOT accepted for integrated transaction architecture

**G1: PASS within models.** Gitea pinned `640cd615` `models/db/context.go::{GetEngine,WithTx,WithTx2,txWithNoCheck}` uses context-scoped transaction owner; BDSPro `models/db/{context,store,transaction}.go` maps authority onto SQLC/pgx instead of copying XORM. `services/org/team.go::NewTeam` is supporting Gitea source for service-driven transaction boundaries.

**G2: PASS.** Exact GitHub diff contains 20 `models/**` paths only, no out-of-lane writes.

**G3: PASS for reviewed transaction scope, with explicit limitations.** `WithTx` / `WithTx2` propagate txCtx; rollback-only state prevents swallowed nested error commits; `transactionDBTX` validates state, pool and contextual authority for SQLC Exec/Query/QueryRow, including escaped `*Queries`; `contextDBTX` chooses the scoped executor when txCtx is passed; raw read/write `queryExecutor` uses the same owner. pgx SQLC, OAuth, nested, cancellation, cross-pool tests were included in CI. **This is not proof that caller misuse of the original parent context is impossible**: a standalone `WithTxResult(ctx,...fn(*Queries))` callback cannot supply txCtx, and a direct ingress Store query with unscoped parent ctx can run on its pool. That is a consumer contract risk, not evidence of two separate PG transaction owners.

**G4: PASS**, exact HEAD matching [run 42790012175](https://github.com/datvtph41107/bdspro/actions/runs/42790012175). No independent local CI rerun by coordinator; result established from GitHub Actions.

**G5: PARTIAL/BLOCKED for integrated contracts.** `WithTxResult` is still exported and used by production `services/user` in dev (though historical A PR #218 already migrates update/settings to `WithTx2`); worker `services/worker/outbox_dispatcher.go::Dispatch` uses `ExecTx`; raw OAuth refresh replay deliberately commits family revocation *before* returning `ErrOAuthGrantReplay` and password-reset transaction locks account-first while atomically revoking sessions and grants. These **must not** be rewritten with a generic error-returning `WithTx` without preserving commit-on-replay/invalid-code counters and lock order. `SQLStore.ExecTxRaw` remains the only explicit Begin/Commit/Rollback owner; multiple entry functions are not automatically multiple transaction authorities, but legacy public API should be retired only after consumers migrate.

**G6: Independent Chat 0 review performed; conditional outcome.** Reviewed real changed source + file diff + GitHub Actions and tested counterexample assumptions against API contracts, not rerun independent adversarial test executables. **Not unconditional ACCEPTED; do not merge PR #221 or open P05 yet.**

## Concrete counterexamples / remaining blockers

1. **Wrong parent Context:** `db.WithTxResult(ctx, store, func(q *db.Queries) (T,error) { _ = db.WithTx(ctx, ...); ... })` cannot inherit derived txCtx. Counterexample if different pool transaction commits inside outer rollback. Do not mask this with a process-global map; remove `WithTxResult` only after all production and test references are either migrated to `WithTx2(txCtx)` or formally justified. Guarded `*Queries` rejects wrong context; direct unscoped ingress Store still has its normal pool semantics.
2. **Direct `ExecTx` legacy worker consumer:** verify `services/worker/outbox_dispatcher.go` locking, claim/ack/lease semantics, and transaction failure. Adapt with `WithTx(ctx)` and `TxQueries(txCtx)` only when preserving exact retry semantics; model code is not allowed to patch worker source without lease.
3. **OAuth replay:** in `models/db/oauth_server.go::RotateOAuthRefreshToken`, consumed token sets `replay=true`, revokes family inside `ExecTxRaw`, callback returns nil to COMMIT, then public error is returned. Replacing this with `return ErrOAuthGrantReplay` inside tx would **rollback security revocation**. This is a deliberate distinct write-result policy, not automatically a duplicate DB engine.
4. **Reset password:** `models/auth/auth.go::ResetPassword` locks account before proof/session mutation and revokes OAuth in same transaction. Preserve account-first lock order, single-use reset token, whole rollback on any failure, session and grant revocation.
5. **Actor/session:** `models/auth/session.go::RequireActingSession` relies on `LockActingSession` inside same tx; at P07 test concurrent revoke/password reset against organization/team write to prove no TOCTOU commit.
6. **Overengineering watch:** `contextDBTX` and `transactionDBTX` are *two executor selectors/guards* for ingress and tx-bound SQLC, not two pgx Begin owners. Keep only if each keeps a concrete security invariant and tests. Avoid another `TxQueriesResult`, context registry, or duplicated raw selector.

## Ownership / lease disposition

- **No Chat B lease for `services/user/**` yet.** PR #218 already edits `services/user/{user,update,settings}.go`. To prevent competing writers, Chat 0 owns reconciliation after A cutover; Chat B remains reviewer/test author in `models/**`, and may not touch `services/user/**` until a new exact-file lease (if still needed). User update/settings migrations on A PR must be independently evaluated as B transaction consumers.
- **`services/worker/outbox_dispatcher.go`** is assigned to **Chat 0 integration** for an isolated worker migration or a separately leased worker owner after contract freeze; not automatically Chat C merely because it handles queue runtime. No parallel edit with Chat A/B.
- **`models/db/transaction.go::WithTxResult`** deletion is B-owned **after verified zero production AND test call sites on combined source**, not before. Coordinator will approve exact zero-caller evidence.
- **P05 remains CLOSED.** Open only once G5 consumer inventory, actor/session/OAuth boundaries and integrated source + CI have been accepted. Then designate one scoped model domain with no API/runtime edits (first candidate `models/organization/**`), not unbounded refactor.

## Merge dependency (not permission to merge)

1. Integrate docs/lease rules or carry them into integration review; A #218 cutover + coordinator stacked #222 final HTTP ingress in one tested composition.
2. Reconcile A historical `services/user/**` transaction changes against B P04 model contract, and worker outbox under coordinator lease.
3. Integrate B #221 and C #220 in an isolated integration branch (B/C file paths do not overlap, but call semantics may); order of B/C can be swapped **only when combined tests pass**. Never merge unverified overlapping consumers to dev.
4. Run Go1.27.3, SQLC, PostgreSQL18, Redis/Asynq, HTTP Auth/OAuth and runtime failure tests on final combined HEAD; only then deliberate dev merge.

**Next Chat B action:** acknowledge CONDITIONAL ACCEPT, freeze scope, supply compact caller inventory and post-cutover contract advice; no additional speculative models refactor and no P05 until Chat 0 grants it.
