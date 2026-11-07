# BDSPro runtime architecture — R01

**Current** source: `main.go`, `cmd/`, `modules/`, `routers/`,
`services/`, `models/db/`, `db/`, `templates/`.
Former `internal/` references live in [historical records](../archive/README.md).

## Execution paths

```text
main.go -> cmd/main.go (CLI flags and command dispatch)
  ├── cmd/web.go
  │   ├── modules/setting (paths, INI, validation)
  │   ├── models/db.InitStore -> SQLC store
  │   ├── models/db.MigrateUp (development only)
  │   ├── modules/conn.RedisManager.Client
  │   └── routers.InitWebInstalled
  │       └── routers/api/v1.Routes -> routers/api/v1/api.go
  │           ├── routers/common (request ID, CORS)
  │           ├── services/context (JWT principal, HTTP errors, ingress)
  │           ├── services/security (throttle, monitoring, login outcomes)
  │           ├── modules/ipactive (per-action admission)
  │           ├── routers/api/v1/user and org (HTTP handlers)
  │           └── services/auth and models/organization -> models/db
  ├── cmd/worker.go
  │   ├── modules/conn (Redis startup readiness)
  │   ├── modules/queue (Asynq transport, processor, logger)
  │   └── services/worker (outbox dispatcher, job adapters)
  │       └── services/jobs (actual email workflows)
  └── cmd/migrate.go
      └── setting.LoadDatabaseSettings -> models/db.MigrateUp/Down
```

`routers/`, `services/` and `modules/` are **different kinds of ownership**.
Their code is not interchangeable just because they are siblings.

- [Cache, queue and Redis lifecycle](cache-queue.md): Gitea comparison, backend ownership and failure policies.
- [JSON serialization policy](json.md): used JSON v2 APIs and behavior compatibility for cache, OAuth2 and queued jobs.
- [R30 JWT, JSON, timeutil and util ownership](r30-auth-json-timeutil.md): Gitea comparison and implemented consumers.
- [Technical modules](modules.md): existing modules, actual Gitea counterparts, and explicit conditions for future modules.
- [Business services](services.md): business operations, queue adapters and transaction ownership.
- [Routers and transports](routers.md): Gin protocol, handlers and API public contracts.
- [Database](data.md): SQLC, migrations and connection lifecycle.
- [HTTP request path](http.md): admission, middleware and response.
- [Process runtime](runtime.md): web/worker/migrate, availability and failures.
- [Gitea evidence and boundaries](gitea-ownership-r01.md): source comparison.

No generic repository registry, routing facade, plugin loader, or empty
`routers/web` is assumed by this architecture.
