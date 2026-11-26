
> **2026-11-25 governance reset:** This content is a historical snapshot/reference, not an active mandate or verified current runtime. Recheck live source and requirements. Do not resume Gitea/F40/parallel plans or infer authorization from it. Current policy: https://github.com/datvtph41107/bdspro/blob/dev/AGENTS.md (on the reset branch, use its local root AGENTS.md).

# F40 — Task Board: Gitea-based Subsystem Migration

**Program status:** DIRECTION ACCEPTED, **0 implementation tasks verified**. Work sequentially on `dev`. Baseline BDSPro `434df714ae0adde03e1246be3b86665a05d8058a`; Gitea `0c1ebb630ead184a007ddebd02eb8eae6e70352e`. Update evidence SHAs for every task. Architecture: [F40 foundation](f40-gitea-subsystem-foundation.md).

Status vocabulary: `OPEN`, `IN_PROGRESS`, `BLOCKED`, `VERIFIED`. Only `VERIFIED` satisfies a gate; record the final SHA, commands and observed test output with each transition.

| Task | Status | Dependency | Deliverable and concrete acceptance |
|---|---|---|---|
| **F40-00** Baseline call graph & usage inventory | OPEN | — | Find **every** `asynq` import, enqueue producer, job type, consumer, worker CLI, outbox SQL, migration, generated SQLC, test, config and docker reference. Map email verify/reset/team invite end-to-end; identify non-job Redis dependencies. Record exact paths/lines and effective config; no code changes. |
| **F40-01** Queue backend and crash-gap ADR | OPEN | F40-00 | Compare Gitea channel/LevelDB/Redis backend against single-/multi-instance BDSPro; choose Redis queue vs LevelDB vs memory; define crash-gap (DB commit before enqueue) policy, recovery/resend, dedup, retention, retry/backoff/poison handling. Acceptance recorded; no claim of exactly-once. |
| **F40-02** Queue contract & backend tests | OPEN | F40-01 | Implement typed `Push`, worker handler `unhandled`/error policy, bounded capacity, backend with concurrency-safe stop, timeout and retry; tests for push full, Redis outage, duplicate, cancellation, backend restart. No Asynq dependency in new path. |
| **F40-03** Worker lifecycle and jobs cutover | OPEN | F40-02 | Wire new queue/worker into `cmd/worker.go` or approved web-owned worker runtime; keep domain `services/jobs` behavior; prove startup readiness, fail-fast, graceful stop, SIGTERM, kill/restart. Assert no double consumer during cutover. |
| **F40-04** Email verification cutover | OPEN | F40-03 | DB commit -> enqueue or reviewed domain-state reconciliation; email verification flows, resend, provider outage, request retry and duplicate delivery integration acceptance. Keep secrets out of payload. |
| **F40-05** Password reset cutover | OPEN | F40-04 | Prove reset version concurrency, secure token hashing, stale-task no-op, replay prevention, outage/retry, kill-between-commit-and-enqueue recovery policy. |
| **F40-06** Team invitation cutover | OPEN | F40-05 | Preserve invite UUID, signer, expiry, accept/revoke semantics, duplicate/restart/outage cases and email delivery. |
| **F40-07** Typed Notify foundation | OPEN | F40-01 | Minimal explicit registration and typed `services/notify` with caller trace, sync dispatch and per-consumer panic/error policy; tests prove conditional consumers and no accidental early publish. No generic global event bus. |
| **F40-08** Audit contract normalization | OPEN | F40-00 | Audit persistence `audit_events`, actor/action/scope/origin and retention; mandatory writes remain atomic with session/member operation; optional best-effort audit notifier only with explicit scope and tests. |
| **F40-09** User inbox schema + consumer | OPEN | F40-07 | Review notification table design, SQLC query/migrations, user scoped read/unread, notification idempotency; only implement if approved real UI contract. Test concurrent updates, authorization and duplicates. |
| **F40-10** Pub/Sub & message foundation | OPEN | F40-07 | Gitea-inspired `Broker`, bounded subscriber channels and idempotent unsubscribe, MemoryBroker tests; Redis broker readiness and fan-out across processes; message envelope/version/topic auth, disconnect/drop behavior and cleanup tests. |
| **F40-11** wsNotifier + authenticated WebSocket | OPEN | F40-09, F40-10 | Websocket notifier publishes user topic, WS handler authenticated subscribe-before-snapshot, ping/write deadline, cancel/reconnect. Verify no cross-user leak, slow subscriber, Redis failure, reconnect snapshot and shutdown. |
| **F40-12** Remove Asynq and obsolete runtime/config | OPEN | F40-03 through F40-06 | Delete old Asynq producer/processor/ServeMux code and dependencies via `go mod tidy`; remove obsolete task config, old schedules/dispatchers and adjust docker/CI/docs. Keep Redis for auth/pubsub/queue where required. `rg 'asynq'` returns only historical references. |
| **F40-13** Retire PostgreSQL `job_outbox` | BLOCKED | F40-01, F40-04 through F40-06, F40-12 | Verify zero unprocessed rows, archive/migrate/compensate live jobs, agree production rollout/restore path. New forward migration drops table/index, remove query and regenerate SQLC. Do **not** rewrite applied migration `000010`; never auto-run DROP during research. |
| **F40-14** Final combined verification | OPEN | F40-07 through F40-13 | `go test ./...`, race/vet where practical, sqlc regeneration, PostgreSQL/Redis/SMTP integration, failure injection, Docker SIGTERM/SIGKILL, browser reconnect, CI on final combined HEAD, reconcile docs and mark only proven tasks VERIFIED. |

## Per-task closure record (copy template)

```text
TASK:
STATUS: OPEN | IN_PROGRESS | BLOCKED | VERIFIED
BASE HEAD:
FINAL HEAD:
GITEA SOURCE SHA + file:line:
BDSPro old caller->callee:
BDSPro new caller->callee:
CONTRACT / FAILURE POLICY:
MIGRATION + ROLLBACK / COMPENSATION:
FILES CHANGED:
COMMANDS RUN + ACTUAL OUTPUT:
TESTS PASSED / SKIPPED / NOT RUN:
KNOWN LIMITATIONS:
NEXT TASK:
```

## Current blockers and non-negotiables

- **F40-13 is BLOCKED by design.** User has requested migration away from outbox, not unconditional destruction of pending jobs.
- Pre-cutover architecture docs describing Asynq remain true of baseline until replacement code exists.
- The current audit contract for `auth.session.revoked` and `team.member.removed` is transaction-mandatory; do not make it best-effort just to align with Gitea.
- Realtime Pub/Sub is not durable, and queue retry does not make PostgreSQL commit + enqueue atomic.
- Security-sensitive outputs and API behavior must be regression-tested before retiring existing transport.
