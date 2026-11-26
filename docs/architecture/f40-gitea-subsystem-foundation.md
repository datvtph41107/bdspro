
> **2026-11-25 governance reset:** This content is a historical snapshot/reference, not an active mandate or verified current runtime. Recheck live source and requirements. Do not resume Gitea/F40/parallel plans or infer authorization from it. Current policy: https://github.com/datvtph41107/bdspro/blob/dev/AGENTS.md (on the reset branch, use its local root AGENTS.md).

# F40 — Gitea-aligned Event, Queue, Realtime and Audit Foundation

Status: **DIRECTION ACCEPTED / IMPLEMENTATION NOT STARTED**. Requested 2026-11-18. This is a documentation-only implementation program: **remove Asynq and PostgreSQL `job_outbox` after a verified replacement**, preserve business/security contracts, and adopt Gitea's subsystem model. Do **not** equate documentation approval with permission to drop production data or lose delivery guarantees.

Pinned source snapshots:
- Gitea `go-gitea/gitea@0c1ebb630ead184a007ddebd02eb8eae6e70352e` (main on 2026-11-18).
- BDSPro `datvtph41107/bdspro@434df714ae0adde03e1246be3b86665a05d8058a` (dev on 2026-11-18).
- Refresh both SHAs at the start of every implementation task. Evidence is version-bound.

## Problem, architectural targets and exclusions

Current BDSPro: `cmd/worker.go` starts PostgreSQL + Redis + Asynq; `services/worker/outbox.go` transfers `job_outbox` to Redis; `services/jobs` owns task business logic. `db/query/job_outbox.sql` defines email verification, password reset, team invitation job intent. `models/audit` and `audit_events` already exist. A Gitea-aligned refactor is **not only** a dependency deletion: it changes startup, storage, delivery, retries, failures, tests and migration paths.

Desired boundaries:

```text
routers -> domain services -> models/db (SQLC/pgx transaction)
                       |                 + required audit INSERT in same transaction
                       + after-commit notify.TypedEvent(...) [in-process dispatch]
                            + UI notification consumer -> notification DB
                                   -> notify.NotificationCountChange
                                   -> websocket wsNotifier -> pubsub.Broker -> subscriber -> WS client
                            + mail consumer -> Queue.Push(Task) -> WorkerPoolQueue -> mail sender
                            + other consumers (explicitly approved, no blanket fan-out)

cmd/web -> settings -> DB -> broker/queue/consumer initialization -> routes/listen -> graceful shutdown
cmd/worker (if retained) -> queue backend + worker lifecycle, no Asynq
```

A domain event is not a queue job, pubsub message, websocket frame, audit record or log line. The same Redis installation may host independent queue and realtime infrastructure with different transport semantics. Do not turn realtime best-effort events into durable job records.

**Technical decisions to resolve in F40-01, not pretend already implemented:**
1. Queue backend: Gitea supports channel (ephemeral), LevelDB (local persistent) and Redis (multi-instance). BDSPro already requires Redis for auth; prefer a **Gitea-style Redis-backed queue with bounded worker pool** if multi-instance and recovery requirements remain. No Asynq dependency. A single-instance in-memory queue is permitted only if documented loss on process crash is accepted and security workflows have recovery.
2. Queue ownership: bounded `Push`, typed item/handler, worker concurrency, shutdown drain, retry/requeue and backoff, dedupe and idempotency; prove capabilities from code, do not claim exact-once.
3. Transaction-to-queue gap: dropping outbox loses atomic `DB COMMIT + durable enqueue intent`. **No Gitea mechanism magically replaces that.** Approval must explicitly choose: (a) acceptance of lost jobs + secure resend/recovery, or (b) a domain-state-based durable reconciliation mechanism that is *not* a generic outbox. Do not delete table before decision and acceptance evidence.
4. Redis Pub/Sub messages are best-effort, subscriber channels can drop on full buffers; client reconnect obtains an authoritative DB snapshot.
5. Required security audit remains within the DB transaction; best-effort audit notifier is optional, not a substitute.
6. Keep business handlers and security invariants; replace queue adapter/transport, not domain behavior. No blind new generic framework or hidden global initialization.

## Gitea evidence map — mechanism, not labels

| Concern | Pinned source | Observed mechanism / important caveat |
|---|---|---|
| Typed event dispatch | [services/notify/notify.go](https://github.com/go-gitea/gitea/blob/0c1ebb630ead184a007ddebd02eb8eae6e70352e/services/notify/notify.go) | `RegisterNotifier`, `[]Notifier`, synchronous iteration; panic recovery at dispatcher level does not resume later callbacks after panic |
| Queue contract | [modules/queue/queue.go](https://github.com/go-gitea/gitea/blob/0c1ebb630ead184a007ddebd02eb8eae6e70352e/modules/queue/queue.go) | `HandlerFuncT[T] func(...T)(unhandled []T)`, workers, backend selection, unique/simple queue caveats |
| Retry/backoff | [modules/queue/backoff.go](https://github.com/go-gitea/gitea/blob/0c1ebb630ead184a007ddebd02eb8eae6e70352e/modules/queue/backoff.go), [workergroup.go](https://github.com/go-gitea/gitea/blob/0c1ebb630ead184a007ddebd02eb8eae6e70352e/modules/queue/workergroup.go) | Bounded backoff waits and requeue of unhandled items; flush/shutdown special case |
| Mail lifecycle | [services/mailer/mailer.go](https://github.com/go-gitea/gitea/blob/0c1ebb630ead184a007ddebd02eb8eae6e70352e/services/mailer/mailer.go) | Notifier registration -> mail queue -> sender; default mail queue handler logs send failure and returns nil, so don't assume every failed mail is automatically retried |
| Pub/Sub | [services/pubsub/broker.go](https://github.com/go-gitea/gitea/blob/0c1ebb630ead184a007ddebd02eb8eae6e70352e/services/pubsub/broker.go), [memory.go](https://github.com/go-gitea/gitea/blob/0c1ebb630ead184a007ddebd02eb8eae6e70352e/services/pubsub/memory.go), [redis.go](https://github.com/go-gitea/gitea/blob/0c1ebb630ead184a007ddebd02eb8eae6e70352e/services/pubsub/redis.go) | `Subscribe(topic) (<-chan []byte, cancel)`, `Publish`, best-effort bounded buffers; Redis broker coordinates across instances |
| WebSocket | [routers/web/websocket/websocket.go](https://github.com/go-gitea/gitea/blob/0c1ebb630ead184a007ddebd02eb8eae6e70352e/routers/web/websocket/websocket.go), [services/websocket/websocket.go](https://github.com/go-gitea/gitea/blob/0c1ebb630ead184a007ddebd02eb8eae6e70352e/services/websocket/websocket.go) | Authenticated Subscribe before initial snapshot, per-connection read/send loop, ping/write deadlines, graceful cancel |
| Realtime notifier | [services/websocket/notification.go](https://github.com/go-gitea/gitea/blob/0c1ebb630ead184a007ddebd02eb8eae6e70352e/services/websocket/notification.go) | `NotificationCountChange` -> query count -> per-user topic message |
| Audit | [models/audit/audit_event.go](https://github.com/go-gitea/gitea/blob/0c1ebb630ead184a007ddebd02eb8eae6e70352e/models/audit/audit_event.go), [services/audit/event.go](https://github.com/go-gitea/gitea/blob/0c1ebb630ead184a007ddebd02eb8eae6e70352e/services/audit/event.go) | One `audit_event` table, action/actor/scope/origin/metadata; recorder logs DB insert failure without propagating it, unlike BDSPro required audit |

## Database decision / migration register

| Table/data | Baseline | Target | Migration rule |
|---|---|---|---|
| `job_outbox` | migration `000010`; pending dispatch rows | **Remove after cutover** | New forward migration only; never edit applied migration history. Inventory `queued_at IS NULL` and old task processing; migrate/drain or explicitly compensate before DROP |
| `audit_events` | existing `000020` + `000021` evolution | **Keep and normalize** | Preserve legacy rejected-login evidence, privacy constraints, actor/scope references; mandatory audit shares tx |
| `notifications` / `notification` | not established as an active canonical inbox table in this research | **Proposed only** | New reviewed migration only when inbox read/unread has an approved API/consumer; suggested `id,user_id,source_type,source_id,status,created_at,updated_at`, uniqueness/idempotency & indices require verified keys; do not claim existing table |
| queue storage | Asynq Redis internal data | **Queue backend selected in F40-01** | Redis queue or LevelDB storage is *transport*, not a replacement PostgreSQL table by default; define crash recovery, TTL, duplicate & ordering |
| app logs / metrics | `modules/log` existing | **Retain** | Do not create log tables for technical logs; use structured logs/metrics, correlation IDs, redact secrets |
| domain source of truth | accounts, sessions, invitations, teams etc. | **Preserve** | Recovery state machine, idempotency constraints and resends may require narrowly scoped domain columns/queries; reviewed migrations only |

## Acceptance invariants (non-negotiable until explicitly changed)

- No plaintext reset proofs, JWTs, OTPs, invitation secrets or unsafe PII in queue payload, audit or logs.
- Email verification, password reset and team invitation still operate correctly with real PostgreSQL + Redis + SMTP test double.
- Password reset worker validates current reset version, issues proof securely, persists only hash; stale jobs have safe no-op semantics.
- Team invitation reuses an opaque durable invite identifier; proof generation remains security-reviewed.
- Session revoke and team-member removal: if required audit INSERT fails then business mutation rolls back.
- WebSocket auth, user-topic isolation, reconnect/snapshot, no cross-user data disclosure.
- Worker shutdown bounded and safe; Redis outage/push full, retry exhaustion, duplicates, SIGTERM and SIGKILL are tested.
- No `asynq` imports, Asynq config, `job_outbox` SQL, dispatcher, cron or generated SQLC references remain in final combined source (except historical docs/migrations and expressly allowed tests).
- No destructive DROP without pending-row report, release rollback/compensation plan and production migration approval.

## Task execution protocol

Single task at a time, work on `dev` unless explicitly redirected. Task board: [F40 task board](f40-task-board.md). Every task is **OPEN → IN_PROGRESS → VERIFIED** (or BLOCKED). Do not mark VERIFIED merely because source was written.

Required task artifact: base HEAD, Gitea pin, BDSPro file/line caller->callee trace, accepted decision/alternative, exact changed files, compatibility impact, schema migration/backfill, unit/integration/fault-injection commands, real observed output, test/CI at **final** HEAD, limits, next dependency. Tests cannot be inferred as passing. Do not mix migration DROP with initial refactor.

Execution gates:
- **G0 baseline**: inventory every live Asynq/outbox caller, configs, docker/cron, migrations, SQLC generated files, test ownership, Redis usage not related to jobs.
- **G1 architecture**: queue backend, job retry and no-outbox recovery policy accepted in ADR. This includes explicit crash-gap risk.
- **G2 replacement**: new queue producer+consumer, startup and lifecycle proven with real transport.
- **G3 cutover**: all three security email flows and failure scenarios pass with new path, old path disabled.
- **G4 cleanup**: remove Asynq + outbox code/config; forward-only DB DROP after drain/approval.
- **G5 post-cutover**: CI, race, vet, integration, restart/downtime and real deployment smoke on final head; docs reconciled.

Existing [cache-queue.md](cache-queue.md), [runtime.md](runtime.md), and [audit-actions-r361.md](audit-actions-r361.md) describe the **pre-cutover** Asynq/outbox implementation and remain historical/current baseline until verified work changes them. This plan is the forward-looking target, not a claim those files have been superseded by code.

## Sources in BDSPro baseline

- [cmd/worker.go](https://github.com/datvtph41107/bdspro/blob/434df714ae0adde03e1246be3b86665a05d8058a/cmd/worker.go) — startup / Asynq runtime
- [modules/queue/workerqueue.go](https://github.com/datvtph41107/bdspro/blob/434df714ae0adde03e1246be3b86665a05d8058a/modules/queue/workerqueue.go) — `asynq.Server`, `ServeMux`
- [services/worker/outbox.go](https://github.com/datvtph41107/bdspro/blob/434df714ae0adde03e1246be3b86665a05d8058a/services/worker/outbox.go) — outbox dispatcher / Redis enqueue
- [db/query/job_outbox.sql](https://github.com/datvtph41107/bdspro/blob/434df714ae0adde03e1246be3b86665a05d8058a/db/query/job_outbox.sql) — three job intent types
- [services/jobs/password_reset.go](https://github.com/datvtph41107/bdspro/blob/434df714ae0adde03e1246be3b86665a05d8058a/services/jobs/password_reset.go) — password-reset handler
- [docs/architecture/audit-actions-r361.md](audit-actions-r361.md) — required vs best-effort audit
