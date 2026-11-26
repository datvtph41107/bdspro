# Current navigation — 2026-11-25

Start with [root AGENTS](https://github.com/datvtph41107/bdspro/blob/dev/AGENTS.md) (use local branch version), [charter](engineering-charter.md), [current phase](research-governance/CURRENT_PHASE.md). Historical architecture and run instructions below must be verified against source before use.

> **2026-11-25 governance reset:** This content is a historical snapshot/reference, not an active mandate or verified current runtime. Recheck live source and requirements. Do not resume Gitea/F40/parallel plans or infer authorization from it. Current policy: https://github.com/datvtph41107/bdspro/blob/dev/AGENTS.md (on the reset branch, use its local root AGENTS.md).

# BDSPro runtime architecture — R01


## Current architecture research entry point (2026-11-20)

Read [Research Governance](research-governance/README.md), [Current Phase](research-governance/CURRENT_PHASE.md), and [Session Handoff](research-governance/SESSION_HANDOFF.md) first. The older Gin-focused execution paths and F40 implementation notes on this page are historical snapshots; they must not be used as today's runtime description or to authorize unrelated implementation. Live BDSPro `dev` source and explicit current scope win when these notes conflict. The research policy requires evidence before accepting any target package tree or design decision.

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

## Active target migration: F40 (requested 2026-11-18)

- [F40 — Gitea-aligned Event, Queue, Realtime and Audit Foundation](f40-gitea-subsystem-foundation.md): pinned mechanisms, replacement of Asynq/outbox, database decisions, lifecycle/error/retry and safety gates.
- [F40 sequential task board](f40-task-board.md): F40-00 through F40-14, per-task closure record, verified status and migration blockers.

**Status:** documentation-only implementation plan. The execution paths above describe the existing pre-cutover source; they must not be interpreted as the F40 target already running. Never drop job_outbox until verified task cutover, backlog reconciliation, and an approved forward migration.
