# P07 — Runtime queue and Redis ownership (post-P06)

**Canonical source:** Chat C PR [#220](https://github.com/datvtph41107/bdspro/pull/220) at `07287479864a33304c243913d5d99d29f1ba48c7`, integrated in draft P07 PR #223. Gitea reference `go-gitea/gitea@640cd615`: `modules/setting/queue.go`, `modules/queue/manager.go`, `modules/log/manager.go`, `cmd/migrate.go`. The objective is the same package responsibility and lifecycle; BDSPro does **not** copy Gitea's in-process queue engine.

## One active queue transport and one durable intent owner

```text
cmd/worker.go::runWorker
  ├── setting.LoadWorkerSettings(app.ini)
  │    └── setting.LoadQueueSettings(provider)
  ├── models/db.InitStore(ctx, DB_SOURCE)
  ├── conn.RedisManager.Client(ctx, REDIS_ADDR)     # PING preflight
  ├── queue.NewRedisEnqueuer(asynq.RedisClientOpt)
  ├── queue.NewRedisTaskProcessorWithConfig(opt, queue.Config)
  │    └── Asynq handlers: verification, password reset, team invite
  ├── queue.Manager.Register("tasks", processor)
  ├── services/worker.OutboxDispatcher.Start(ctx)
  └── services/worker.LoginRetention.Start(ctx)

services/worker.OutboxDispatcher.Dispatch(ctx)
  └── db.WithTx(db.WithStore(ctx, store), callback(txCtx))
       ├── db.TxQueries(txCtx)
       ├── ListPendingJobOutbox(txCtx)      # FOR UPDATE SKIP LOCKED
       ├── queue.TaskEnqueuer.Enqueue(...)  # Redis / Asynq, timeout from txCtx
       └── MarkJobOutboxQueued(txCtx)       # after enqueue / duplicate TaskID
```

`modules/queue` contains the **Asynq** producer, processor, config and manager. `modules/conn` owns shared go-redis client creation, PING and close; the Asynq producer and processor own their own Redis connections. No process-global worker manager is necessary. The channels in `modules/queue/base_channel.go` and `channel_worker.go` were deleted by P06 because no production consumer used them.

**At-least-once, not exactly-once:** PostgreSQL commit and Redis enqueue are not atomic. An enqueue may succeed and PostgreSQL may roll back; the durable outbox retries and the deterministic Asynq TaskID handles a duplicate acceptance as success. Consumers must be idempotent. The dispatcher retains row locking, enqueue-before-mark and retry semantics. The real PostgreSQL and Redis/fault-injection tests are `services/worker/outbox_fault_injection_test.go`, alongside process-level Asynq lease recovery.

## Optional worker queue configuration

Missing `[queue]` keys use existing defaults; **explicitly empty, malformed or zero values fail before opening DB/Redis in the worker command**:

```ini
; Optional worker-only section in app.ini.
[queue]
CONCURRENCY = 10
CRITICAL_WEIGHT = 10
DEFAULT_WEIGHT = 5
```

Limits are deliberate **BDSPro operational caps**, not Asynq or Gitea framework limits: concurrency 256, any queue weight 100, aggregate weight 1000. Both `setting.LoadQueueSettings` and direct `queue.Config.Validate` enforce relevant bounds. Names `critical` and `default` must match durable outbox `queue_name` producers. Worker-only configuration is not required by `migrate`, `help` or normal HTTP routes.

## Cache intentionally absent from current production source

P06 removed `modules/cache/**` and `modules/setting/cache.go` because there was **no production consumer**. This is different from Gitea, which uses cache for actual read-heavy projections. There is no `cache.NewStringCache`, `LoadCacheSettings`, Memcached backend, `[cache]` configuration or cache startup path in BDSPro's current P07 source. Older R28 architecture notes and tests described an implementation that was removed; treat those as historical, not active contracts.

A future cache needs an identified read-heavy consumer, TTL/invalidation, miss/outage behavior, security risk and ownership. Do not cache OTP attempts, live session revocation, OIDC one-time state or rate-limit counters as general-purpose TTL values.

## Operational failure and cleanup

- Missing/bad worker configuration aborts before DB and Redis connections.
- PostgreSQL or Redis PING failure at startup aborts web/worker; `migrate` and `help` do not depend on Redis.
- Redis/Asynq enqueue failure **must not mark** an outbox record queued; retry uses durable intent.
- `runWorker` cancellation first waits for outbox dispatcher and retention sweep; `queue.Manager.Shutdown` drains Asynq active tasks; then producer/Redis/DB are closed in reverse resource order. Main process closes logging sink. Cooperative active-task drain is covered; uncooperative Go handlers and real Redis/PG server failover have **not** been proven.
- `cmd/http.go` owns HTTP graceful shutdown; do not introduce `modules/graceful` without a real shared lifecycle consumer and a failing integration case.

**P07 gate:** full Go1.27.3 test/race/vet, PostgreSQL18 outbox rollback, Redis outage/retry, worker SIGKILL/lease recovery, deleted-import audit and Docker smoke must pass on the **combined** HEAD, not merely on PR #220 alone.
