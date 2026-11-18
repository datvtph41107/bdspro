# G02 — Gitea-aligned application composition contract v1 (proposal)

**Decision:** PROPOSED; applies to the current Web + Worker slice, not a universal subsystem interface.
**Change scope:** explicit startup sequence, typed handler registration, package-oriented business operations, resource ownership and managed runtime activities.
**Reference comparison:** Gitea `main` and BDSPro `refactor/g02-resource-boundaries-20261118`; use GitHub permalink/commit when reviewing changes (moving branch URLs are not immutable evidence).

## 1. The code pattern, not a naming convention

```text
main.go
  → cmd.NewMainApp        [register CLI commands only]
  → cmd.runWeb            [initialize graceful manager]
  → routers.InitWebInstalled
       → setting.LoadSettings
       → common.InitDBEngine
           → db.InitEngine(ctx, dsn)  [resource publication]
           → graceful.RunAtTerminate(db.CloseEngine) [composition policy]
       → common.InitRedis
           → conn.InitRedis(ctx, address)
           → graceful.RunAtTerminate(conn.CloseRedis)
       → auth.Init
  → routers.NormalHandler [compose ingress dependencies + route tree]
       → routers.NormalRoutes
         → apiv1.Routes
           → Group("orgs/{org}", common.APIAuth, orgAssignment)
           → Combo("/teams").Post(reqOrgOwner(), reqIPAdmission(...), bind(...), org.CreateTeam)
  → HTTP request
       → request APIContexter
       → APIAuth → orgAssignment → reqOrgOwner → reqIPAdmission → bind
       → org.CreateTeam
       → orgservice.CreateTeam
           → db.WithTx2(ctx, ...)
           → db.GetQueries(txCtx) → SQLC → PostgreSQL
```

**Timing:** Go package import links symbols and runs package initialization; it is not constructor injection. CLI registration is not resource initialization. Route registration compiles action chains and never calls the final business handler. Each request creates an APIContext and executes the registered actions. An Organization service is not instantiated at startup: `orgservice.CreateTeam` is a package-level function.

**Precise source trace in BDSPro:**
- `main.go`: `main → cmd.NewMainApp → cmd.RunMainApp`.
- `cmd/main.go`: `NewMainApp`, `PrepareSubcommandWithGlobalFlags`, `prepareConfig`.
- `cmd/web.go`: `runWeb → serveInstalled → InitWebInstalled → NormalHandler → listen`.
- `routers/init.go`: `InitWebInstalled`, `NormalRoutes`, `NormalHandler`.
- `routers/api/v1/api.go`: `Routes`, `bind[T]`.
- `modules/web/router.go`: `Group`, `Methods`, `wrapMiddlewareAndHandler`.
- `modules/web/handler.go`: `toHandlerProvider`, `prepareHandleArgsIn`, `shouldStopContinuation`.
- `services/context/http.go`: `APIContexter` produces a request-scoped context.
- `routers/api/v1/org/team.go`: HTTP `CreateTeam`.
- `services/org/team.go`: business `CreateTeam` and transaction.
- `models/db/context.go`: `WithTx2` / `GetQueries`, selecting transaction-local SQLC queries.

## 2. Gitea reference versus BDSPro adaptation

| Question | Gitea source and observable mechanism | BDSPro contract | Classification |
| --- | --- | --- | --- |
| How is bootstrap ordered? | `go-gitea/gitea/routers/init.go::InitWebInstalled` calls `mustInit/mustInitCtx` in code order | `routers.InitWebInstalled` calls `startup.Run` with required steps and rollback | ADAPTED: error propagation instead of fatal |
| Must each use case have a constructor? | `go-gitea/gitea/services/org/team.go::NewTeam` invokes `db.WithTx` directly | `services/org.CreateTeam` calls `db.WithTx2` and SQLC queries via context | ALIGNED |
| When do handlers run? | Gitea API routes register typed actions with `modules/web` | BDSPro route registration compiles typed actions; they run only at request time | ALIGNED |
| How is optional background work started? | `go-gitea/gitea/services/actions/init.go::Init` checks `setting.Actions.Enabled` and schedules queue activity through graceful | BDSPro worker is a separate CLI process, configured via `cmd/worker.go` and managed with graceful activity registration | ADAPTED to current deployment mode |
| Who registers DB/Redis termination? | Gitea uses global setting/lifecycle APIs within various initializers | BDSPro Web/Worker composition registers termination; resource owners only expose init/close | DELIBERATE TIGHTER BOUNDARY |

**Source links:**
- Gitea bootstrap: https://github.com/go-gitea/gitea/blob/main/routers/init.go
- Gitea package service: https://github.com/go-gitea/gitea/blob/main/services/org/team.go
- Gitea optional Actions runtime: https://github.com/go-gitea/gitea/blob/main/services/actions/init.go
- BDSPro bootstrap: https://github.com/datvtph41107/bdspro/blob/refactor/g02-resource-boundaries-20261118/routers/init.go
- BDSPro router action composition: https://github.com/datvtph41107/bdspro/blob/refactor/g02-resource-boundaries-20261118/routers/api/v1/api.go
- BDSPro business operation: https://github.com/datvtph41107/bdspro/blob/refactor/g02-resource-boundaries-20261118/services/org/team.go

## 3. Explicit contracts

### CLI and bootstrap

- CLI constructors only register commands and hooks; they do not open DB/Redis or run business operations.
- `cmd.runWeb` initializes graceful manager before invoking application initialization.
- Required initializer runs in declaration order, returns a contextual error and aborts later stages.
- Failed stage owns partial cleanup; previously completed stages are rolled back in reverse order.
- Resource owners can expose `Init(ctx, requiredConfig)` + idempotent `Close()`; Web/Worker composition must register termination *after successful initialization* and *before exposing consumers*.
- Initialization, readiness, runtime drain and cleanup are distinct phases. Do not infer shutdown order from ordering of graceful hooks.

### HTTP / service boundary

- Router only registers scoped guards, binding and endpoint action functions.
- A typed handler receives one request-scoped `APIContext`, translates transport input to an operation and translates operation results/errors into HTTP.
- Package-level business service accepts `context.Context`, input and domain identifiers; no `APIContext` or HTTP response in the business operation.
- Authorization guard is defense-in-depth; SQL write authorization remains authoritative under transaction/locking. Do not remove `CreateTeamForOwner`/SQL checks.
- `db.GetQueries(txCtx)` must receive the transaction context from `db.WithTx2`; using the outer context can escape the transaction.
- Shared process resources do not imply universal global dependencies. If a component owns per-instance state, external transports or alternative implementations, evaluate a constructor explicitly.

### Runtime subsystem boundary

- Do not require stateless Organization operations to implement `Run`/`Close`.
- Use managed graceful activity registration for actual long-lived workers, queue subscribers and cron loops.
- Feature disablement is a composition policy. Define whether disabled means “no routes”, “capability unavailable” or “no background runner” per feature.
- Never introduce a generic `Subsystem` interface simply to unify names.

## 4. Validation and actual costs

Added tests in `modules/startup/contract_test.go` verify: required order; only successful stages rolled back in reverse; error identity via `errors.Is`; cancellation after a successful stage; invalid pipeline detected before any side effect; nil context/empty pipeline rejected.

Run:

```bash
go test ./modules/startup
go test ./tools/archcheck
go run ./tools/archcheck
go test -race ./...
```

These tests establish only the contract they exercise. They do not prove live SIGTERM with an in-flight transaction, the Docker runtime cost, or dispatcher retries. See `g02-resource-import-boundaries.md` for the import-edge measurements and unverified failure matrix.

**Extra engineering cost:** one source-aligned contract document, one small startup test suite; no new runtime interfaces, DI container, DB migration, route or query change in this increment.

**Open:** notification service composition, optional feature enablement across API/worker, failure injection after handler construction, graceful shutdown of in-flight SQL transactions and Asynq jobs, comparative DX measurement versus v4/Wire.

## 5. Review gate

For every new BDSPro subsystem, link its **Gitea reference → target code entrypoint → owner/import rule → required behavior → executable test**. Keep decision state PROPOSED until a reviewer accepts the contract. Preserve a direct package service when it is sufficient; use constructor injection if the specific service owns state, external transports or testable implementation variants and can justify its wiring cost.
