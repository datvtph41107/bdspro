
> **2026-11-25 governance reset:** This content is a historical snapshot/reference, not an active mandate or verified current runtime. Recheck live source and requirements. Do not resume Gitea/F40/parallel plans or infer authorization from it. Current policy: https://github.com/datvtph41107/bdspro/blob/dev/AGENTS.md (on the reset branch, use its local root AGENTS.md).

# Source Evidence Map — pinned starting anchors

**Status:** navigation and evidence, NOT approval of any current architecture. Refreshed 2026-11-17. Commit SHAs below identify snapshots; refresh branch HEAD before new work.

## Primary reference: Gitea

Repository: https://github.com/go-gitea/gitea

Inspected snapshot: `bf5c39b310f99b4254266bebee535d3730daba0a`.

| Question | Source anchor | Observed fact (not a design verdict) |
| --- | --- | --- |
| CLI and process manager | [cmd/web.go#L261-L296](https://github.com/go-gitea/gitea/blob/bf5c39b310f99b4254266bebee535d3730daba0a/cmd/web.go#L261-L296) | `runWeb` initializes graceful manager, then enters serving path |
| Startup composition | [routers/init.go#L114-L175](https://github.com/go-gitea/gitea/blob/bf5c39b310f99b4254266bebee535d3730daba0a/routers/init.go#L114-L175) | Ordered `InitWebInstalled` calls across settings, models, services, queues and cron |
| Fatal boundary | [routers/init.go#L60-L76](https://github.com/go-gitea/gitea/blob/bf5c39b310f99b4254266bebee535d3730daba0a/routers/init.go#L60-L76) | `mustInit`/`mustInitCtx` name failing function and call `log.Fatal` |
| DB retry/migration policy | [routers/common/db.go#L21-L59](https://github.com/go-gitea/gitea/blob/bf5c39b310f99b4254266bebee535d3730daba0a/routers/common/db.go#L21-L59) | DB retries/backoff; migration/version checks are separate policy |
| Cache readiness | [modules/cache/cache.go#L21-L41](https://github.com/go-gitea/gitea/blob/bf5c39b310f99b4254266bebee535d3730daba0a/modules/cache/cache.go#L21-L41) | Cache creation with Ping retries before publication |
| Queue registration | [modules/queue/manager.go#L92-L116](https://github.com/go-gitea/gitea/blob/bf5c39b310f99b4254266bebee535d3730daba0a/modules/queue/manager.go#L92-L116) | Named queue from settings, creation error logged and nil returned |
| Queue activity | [modules/queue/workerqueue.go#L215-L253](https://github.com/go-gitea/gitea/blob/bf5c39b310f99b4254266bebee535d3730daba0a/modules/queue/workerqueue.go#L215-L253) | Queue tracks typed process context, channels and cancel state |
| Lifecycle coordination | [modules/graceful/manager.go#L53-L109](https://github.com/go-gitea/gitea/blob/bf5c39b310f99b4254266bebee535d3730daba0a/modules/graceful/manager.go#L53-L109) | Manager initialization, shutdown-aware running and hooks |
| Serving boundary | [cmd/web.go#L168-L244](https://github.com/go-gitea/gitea/blob/bf5c39b310f99b4254266bebee535d3730daba0a/cmd/web.go#L168-L244) | Installed init before `NormalRoutes` and `listen` |

These demonstrate *code mechanism*; they do not alone establish why upstream authors chose each trade-off.

## Requirements/history evidence: old BDSPro CRM v4

Repository https://github.com/datvtph41107/bdspro-backend ; inspected commit `0b16ec4b46b1c8813e93a1867f999f3aebd9c97a`.

| Anchor | What it tells us |
| --- | --- |
| [crm-service/initial/startup.go](https://github.com/datvtph41107/bdspro-backend/blob/0b16ec4b46b1c8813e93a1867f999f3aebd9c97a/crm-service/initial/startup.go) | Many real business usecases/handlers collected in `InitialApp`; scope pressure is real |
| [crm-service/wire/wire.go](https://github.com/datvtph41107/bdspro-backend/blob/0b16ec4b46b1c8813e93a1867f999f3aebd9c97a/crm-service/wire/wire.go) | Large compile-time constructor graph |
| [crm-service/wire/wire_gen.go](https://github.com/datvtph41107/bdspro-backend/blob/0b16ec4b46b1c8813e93a1867f999f3aebd9c97a/crm-service/wire/wire_gen.go) | Actual constructor order/cleanup generated from Wire |
| [crm-service/cmd/grpc/main.go](https://github.com/datvtph41107/bdspro-backend/blob/0b16ec4b46b1c8813e93a1867f999f3aebd9c97a/crm-service/cmd/grpc/main.go) | Explicit gRPC registration and background actors |
| [crm-service/cmd/payment-completed-consumer/main.go](https://github.com/datvtph41107/bdspro-backend/blob/0b16ec4b46b1c8813e93a1867f999f3aebd9c97a/crm-service/cmd/payment-completed-consumer/main.go) | Independently deployed process with DB and broker |

Historical scope is **NOT** approval to recreate Wire, microservices or an enormous application container.

## Audit target: BDSPro monolith dev

Repository https://github.com/datvtph41107/bdspro ; snapshot `440d885afdc60d6b92ed92bef258c7354bff2915` (at time of this documentation change).

| Anchor | Observation to investigate, not to defend |
| --- | --- |
| [cmd/web.go](https://github.com/datvtph41107/bdspro/blob/440d885afdc60d6b92ed92bef258c7354bff2915/cmd/web.go) | CLI + graceful bootstrap/serving path |
| [routers/init.go](https://github.com/datvtph41107/bdspro/blob/440d885afdc60d6b92ed92bef258c7354bff2915/routers/init.go) | `startup.Run`, `NormalRoutes`, `NormalHandler` |
| [modules/startup/startup.go](https://github.com/datvtph41107/bdspro/blob/440d885afdc60d6b92bef258c7354bff2915/modules/startup/startup.go) | Stage names, rollback and context checks |
| [models/db/engine_init.go](https://github.com/datvtph41107/bdspro/blob/440d885afdc60d6b92ed92bef258c7354bff2915/models/db/engine_init.go) | Process DB state and graceful termination hook |
| [modules/conn/redis_init.go](https://github.com/datvtph41107/bdspro/blob/440d885afdc60d6b92ed92bef258c7354bff2915/modules/conn/redis_init.go) | Redis owner and borrowed-client access |
| [modules/graceful/manager.go](https://github.com/datvtph41107/bdspro/blob/440d885afdc60d6b92ed92bef258c7354bff2915/modules/graceful/manager.go) | Shutdown and termination callback behavior |

**Investigation gap:** complete constructor/consumer graph, actual background process ownership, costs of composition alternatives, real startup/deploy observability and lifecycle contracts. Do not infer completeness from directory names or tests.

## Additional authoritative sources — use when a question demands them

- [Go memory model](https://go.dev/ref/mem), [Go context](https://pkg.go.dev/context), [Go net/http](https://pkg.go.dev/net/http), [database/sql](https://pkg.go.dev/database/sql), [Go testing](https://pkg.go.dev/testing).
- [PostgreSQL official documentation](https://www.postgresql.org/docs/) and relevant driver docs for transactions/pools.
- [OpenTelemetry documentation](https://opentelemetry.io/docs/) for instrumentation semantics; distinguish standard from actual BDSPro implementation.
- Platform documentation (Docker/Kubernetes/systemd) **only for the deployment model actually chosen**.

These are references, not assertions that BDSPro already uses a given deployment topology or monitoring stack.
