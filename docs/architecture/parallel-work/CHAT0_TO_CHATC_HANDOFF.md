# CHAT0_TO_CHATC HANDOFF — Independent P06 review

Date: 2026-11-10. Reviewed PR [#220](https://github.com/datvtph41107/bdspro/pull/220) at **exact HEAD `07287479864a33304c243913d5d99d29f1ba48c7`**, base `dev@8dea6dc9`; 17 commits ahead, 0 behind, 41 changed paths inside `cmd/**`, `modules/{setting,log,queue,cache}/**`. Exact-HEAD [CI 42779188456](https://github.com/datvtph41107/bdspro/actions/runs/42779188456) **COMPLETED/SUCCESS**.

## Independent decision: **CONDITIONAL ACCEPT — P06 Chat C source slice only. FREEZE OWNED SOURCE by default**

**G1: PASS scoped:** pinned Gitea `640cd615` `cmd/migrate.go::runMigrate`, `modules/setting/queue.go::GetQueueSettings`, `modules/queue/manager.go`, `modules/log/manager.go`; BDSPro `cmd/{helper,worker,migrate}.go`, `modules/{setting,queue,log}`. Gitea's level/channel workers have genuine consumers, BDSPro's channel backend did not; copying it would be wrong.

**G2: PASS:** 41 paths all C-owned, including deletion of test-only cache/channel packages. No production API/DB/route edits.

**G3: PASS for reported+reviewed scope with limitations:** `cmd/worker.go` initializes validated settings, DB, Redis, Asynq producer/processor and drains dispatcher/retention on shutdown; logger selection uses `--work-path`, file ownership/cleanup. `modules/setting/queue.go` validates positive bounded concurrency/weights; `modules/queue/config.go` validates the total and per-queue caps. Tests cover CLI, startup, HTTP drain, Asynq active-task shutdown, Redis-link recovery. **No proof of full Redis-server/PG outage failover, exactly-once effects, or termination of a Go handler ignoring cancellation.** Such system-level cases belong P07, not new unused modules.

**G4: PASS exact HEAD**, [run 42779188456](https://github.com/datvtph41107/bdspro/actions/runs/42779188456). Coordinator independently inspected code/CI but did not rerun live network failure matrix locally.

**G5: CONTRACT ACCEPTED for handoff, NOT yet integrated.** `setting.LoadWorkerSettings(filename)` supplies `QueueSettings`; `queue.NewRedisTaskProcessorWithConfig(options, queue.Config)` is consumed by `cmd/worker.go`; `cmd.configureCommandLoggers(setting.Paths,setting.LogGlobalConfig)` anchors file root; existing `routers.InitWebInstalled` signature unchanged. No `modules/cache` constructor remains; do not force placeholder cache or `modules/graceful`. Coordinator owns docs/config templates/go.mod and combined integration.
**G6: independent Chat 0 review completed — CONDITIONAL ACCEPT**. PR stays DRAFT, not merged, until P07 integrated tests and docs/code references pass.

## Findings / CHAT0_TO_CHATC

- **C-01 — CLOSED / owner C:** remove unused channel task backend with no production consumer; code in #220 already does this. Do not recreate to resemble Gitea.
- **C-02 — CLOSED / owner C:** unused cache module deleted; no production consumer. If new real consumer appears later, new decision/lease required; do not bring back adapters.
- **C-03 — CLOSED / owner C:** worker queue option bounds, missing vs empty app.ini validation, logger file root and CLI process dependencies implemented and covered by regression tests.
- **C-04 — CLOSED / owner C:** active Asynq handler is drained on shutdown in Redis integration test; no exactly-once guarantee.
- **C-05 — OPEN / owner Chat 0:** stale docs `docs/architecture/cache-queue.md`, `docs/architecture/modules.md`, `docs/operations/configuration.md`; replace removed cache/channel claims, ensure docs reflect Asynq/SQL outbox and failure policy.
- **C-06 — OPEN / owner Chat 0:** external `app.ini` templates and `go.mod/go.sum` import cleanup (especially memcache dependency): inspect after combining PRs, do not delete any dependency still used by other packages.
- **C-07 — OPEN / owner Chat 0/P07:** combined runtime/integration regression with A HTTP ingress, B DB/session authority and C startup/shutdown. Test CORS 404/405/HEAD, trusted proxies, Redis fail-closed, worker lease retry, DB failed transaction, context cancellation.
- **C-08 — DEFER / owner Chat 0/P07:** live Redis-server/PG failover, uncooperative handler shutdown, exactly-once impossibility. Document risk and validate if operational policy requires; do not imply it was already tested.
- **C-09 — DEFER / owner Chat 0:** `modules/graceful` is not needed absent more than one real lifecycle consumer; `cmd/http.go` and `cmd/worker.go` own actual lifecycle. No empty module.
- **C-10 — RECHECK at combined HEAD / owner Chat 0:** inspect package references to deleted `modules/cache`, `modules/setting/cache`, `queue.NewChannelTaskQueue` across runtime, tests, docs and external config. PR source diff is not alone a proof against other branches.

## Concrete counterexample controls

1. A consumer importing removed cache compilation fails after C #220 integrated: run full `go test ./...` and non-test imports after **combined** A/B/C.
2. An [queue] `CONCURRENCY=0` or malformed weight must abort before DB/Redis init, rather than create uncontrolled worker(s). Current setting tests validate inputs; verify command init contract.
3. Redis failure at startup must fail closed for required auth/throttle, with explicit exceptions only where designed. Worker lease recovery is at-least-once; preserve idempotency, no exactly-once claim.
4. If a handler blocks forever ignoring cancellation, Asynq/HTTP shutdown cannot guarantee memory/goroutine completion. Do not claim otherwise.
5. Logger double-init or file errors must never silently fall back to console without policy; preserve early stderr until configured sink works, close owned resources once.

## Direction to Chat C

**No new code by default.** Confirm this report against the latest branch, mark C-01..C-04 closed, and deliver an invariant/constructor matrix and P07 integration checklist. If a **new verifiable defect in C-owned source** is found, fix it narrowly with one failure-first test, re-run exact-head CI, send commit+finding+counterexample. No edits to `routers/**`, `services/**`, `models/**`, `db/**`, `docs/**`, `go.mod` or config templates without an approved lease. Do not merge #220 or claim whole-system P06 INTEGRATED.
