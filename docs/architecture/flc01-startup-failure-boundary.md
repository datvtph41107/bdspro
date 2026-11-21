> **Historical FLC-01 implementation evidence:** the typed `cfg` arguments
> shown in the earlier call graph were superseded by the user-approved
> [canonical process settings](canonical-settings-and-ownership.md).
> Route registration still runs before DB/Redis initialization, but
> validated settings are published once before route preparation.

# FLC-01 — Startup failure boundaries, route preflight and cancellation

Status: IMPLEMENTATION TARGET for `dev`, reviewed against
`45c82c20603e59195e7cdae8c650a58dc8f63bb8` as the baseline.
This report must be read together with the current implementation commit and
its CI run. Do not treat a passing test as proof that all production incidents
are impossible.

## Problem (not a pattern name)

Before this change, `cmd.serveInstalled` called
`routers.InitWebInstalled` (PostgreSQL, migration policy, Redis, JWT)
*before* `routers.NormalHandler`, which registered routes and could panic
for a malformed handler signature. In the same startup scope it passed
`g.HammerContext()`, which is canceled later than
`g.ShutdownContext()` during shutdown.

The risk is unnecessary resource creation before a deterministic programmer
error is exposed, and delayed cancellation of work which has never begun
serving requests. This is not a claim that the old code necessarily leaked
OS file descriptors or caused a production incident.

## Source chain and change boundary

| Owner | Before | FLC-01 contract |
| --- | --- | --- |
| `modules/setting` | `LoadWebConfig` | Unchanged: parse and validate once |
| `routers.NormalRoutes` | Called during `NormalHandler` after DB/Redis | Called by `PrepareWebRoutes(cfg)` before resource startup |
| `modules/web.Router.Methods` | Compiles typed actions and panics for programming errors | Unchanged; these errors now surface during preflight |
| `routers.PrepareWebRoutes` | Did not exist | Build Chi route tree, optional Swagger, trusted proxies; return handler; no DB/Redis/Manager |
| `cmd.runWeb` | Load, logger, legacy globals, graceful, resource init, route compile | Load, logger, **route preflight**, legacy globals, graceful, resource init, attach runtime admission, listener |
| `routers.NormalHandler` | Borrow Redis then compile routes | Borrow Redis and attach limiter to prepared handler; it cannot register more routes |
| `modules/graceful.Manager` | Shutdown and Hammer phase contexts | Unchanged |
| `modules/startup.Run` | Rollback successful steps in reverse on returned errors | Unchanged |
| `models/db` / `modules/conn` | Process-owned resources with idempotent closers | Unchanged |

New call graph:

```text
cmd.runWeb
  -> cmd.prepareWeb
       -> setting.LoadWebConfig(args, envs)
       -> setting.InitLoggers(cfg.Common.Log)
       -> routers.PrepareWebRoutes(cfg)
       -> routers.NormalRoutes()
            -> apiv1.Routes() / web.Router.Methods()  // program error can panic
       -> api.SwaggerHandler(cfg.Common.RunMode)       // only if enabled
       -> routes.SetTrustedProxies(cfg.Server.TrustedProxies)
       -> routes.WrapIngress()
  -> setting.PublishLegacyWeb(cfg)                     // compatibility bridge
  -> graceful.InitManager(ctx, cfg.Common.Graceful.HammerTimeout)
  -> cmd.serveInstalled(cfg, preparedRoutes)
       -> graceful.Manager.RegisterServer()
       -> routers.InitWebInstalled(g.ShutdownContext(), cfg)
            -> startup.Run(PostgreSQL, Redis, JWT)
       -> routers.NormalHandler(preparedRoutes, cfg.IPActive.Policies)
            -> conn.ProcessRedis()                     // borrowed, not closed here
            -> ipactive.NewRedis(client, policies)
       -> cmd.listen(handler, address, HTTPShutdownTimeout)
            -> graceful.HTTPListenAndServe()
```

## Error semantics: do not confuse fail-fast with killing the process

1. Invalid config returns an error *before* route preflight or resource init.
2. Invalid trusted proxy returns a descriptive preflight error.
3. Bad route handler signatures remain programmer panics (no catch-all recovery
   and no silent fallback). They now occur before DB, Redis and JWT are opened.
4. During startup, SIGTERM cancels `g.ShutdownContext()`; cooperative
   DB/Redis/migration operations unwind through `startup.Run`.
5. When cancellation was requested and startup returns `context.Canceled`,
   `cmd` treats this as intentional shutdown. Unrelated failures remain
   non-nil errors for process failure, even if shutdown races with the error.
6. A failure **inside** `Step.Init` must clean up its own partial resources.
   Completed steps are rolled back in reverse order. Resource-owner Close
   functions must tolerate a later termination callback.
7. Once HTTP is serving, a separate drain contract applies. Active request
   contexts must not be canceled merely because startup should be canceled;
   existing HTTP shutdown/hammer semantics are retained.

This change intentionally does not add a generic panic recovery mechanism
to `startup.Run`. Any unrelated panic after resource initialization remains
an OPEN risk. Routes are statically precompiled because that is the
deterministic production-relevant programmer error established by the source.

## Failure-injection evidence gates

| Scenario | Invariant | Test or verification |
| --- | --- | --- |
| Valid route tree | Can compile and serve `/api/v1/health` without DB/Redis | `routers.TestPrepareWebRoutesWithoutRuntimeResources` |
| Bad trusted proxy | Rejected before resources are opened | `routers.TestPrepareWebRoutesRejectsInvalidTrustedProxyBeforeResources` |
| Invalid handler signature | Rejected at route registration | `routers.TestRouterRejectsProgrammingErrorDuringRegistration` |
| Missing prepared handler | No fallback to a second late route build | `routers.TestNormalHandlerRejectsMissingPreparedRoutes` |
| Shutdown during startup | Startup returns wrapped `context.Canceled` before hammer | `graceful.TestShutdownContextCancelsStartupBeforeHammer` |
| Redis fails after DB | Completed DB step is rolled back | existing `startup.TestRunRollbackReverseOrder` |
| HTTP request in progress | HTTP drain still runs before termination | existing `graceful.TestHTTPShutdownDrainsActiveRequestBeforeResourceTermination` |
| Live signal/deployment | Must be run in disposable Docker or staging | **NOT VERIFIED by the new unit tests alone** |

Run `go test -race ./...`, `go vet ./...`, `go build ./...`,
Docker Compose integration and HTTP smoke at the **final** commit before marking
FLC-01 source integration VERIFIED.

## Go package responsibilities and import boundaries

- `cmd` imports `routers`, `setting` and `graceful` because it coordinates
  the entrypoint. It must not implement Chi routing or PostgreSQL connection code.
- `routers` owns route registration and HTTP composition; it may import
  `modules/web` and use typed config values. It must not close DB/Redis.
- `modules/web` owns generic router/handler contract and has no DB/Redis
  import; it preserves compile-time (registration-time) handler validation.
- `modules/startup` owns ordered execution and reversal of completed steps,
  not the definition of business-critical resources.
- `models/db` and `modules/conn` own connections and idempotent close,
  without reading `modules/setting` globals.
- `modules/graceful` owns phase contexts, registration and termination, not
  business transaction rollback.
- No `modules/util` helper, DI container, extra retry manager or lifecycle
  framework was introduced by this change.

## External source comparison (pinned, not copied)

1. Gitea `routers/init.go` `mustInit`, fatal on required startup failure:
   https://github.com/go-gitea/gitea/blob/4a9a53d79abc8ab339b18d6c8c59004d7c9d5126/routers/init.go#L60-L76
2. Gitea `routers/common/db.go` bounded DB retry and schema policy:
   https://github.com/go-gitea/gitea/blob/4a9a53d79abc8ab339b18d6c8c59004d7c9d5126/routers/common/db.go#L20-L57
   BDSPro does **not** blindly copy `log.Fatal` or retry. Existing
   Docker restart-on-failure and Compose healthy dependencies must be
   benchmarked against an in-process retry proposal first.
3. Caddy's app Start/Stop contract:
   https://github.com/caddyserver/caddy/blob/edcdfa0ed88f7dcc1c53720d25218fde9d63eb1e/caddy.go#L97-L114
   BDSPro retains its existing manager and startup coordinator.
4. PocketBase constructs the application from explicit config:
   https://github.com/pocketbase/pocketbase/blob/7f2773ec234b62d0b80ca47c508397cb3d0da3ea/core/base.go#L211-L247
   BDSPro keeps its smaller typed `WebConfig` and explicit constructors.
5. Go context contract: https://pkg.go.dev/context — cancellation is
   cooperative and cannot preempt a function that ignores the context.

## Explicitly OPEN / not changed

- How to recover and clean up from *arbitrary* panics during later
  startup stages; no broad recover/re-panic wrapper was added.
- Whether DB connection retry is justified after measuring readiness,
  restart count and time to service under actual deployment conditions.
- Request/transaction cancellation and `pgxpool.Close` delay if code
  retains connections; requires separate integration tests.
- Availability of DB/Redis after startup, liveness/readiness semantics,
  HTTP error logs, release-level zero downtime.
- Single-process DB/Redis globals and compatibility settings bridge.
- Deployment-specific SIGTERM tests under load beyond current CI smoke.

**Decision discipline:** These remain hypotheses until a reproducible test
shows a violated invariant and a narrower mechanism cannot solve it.

Update (2026-11-21): `cmd.runWeb` now delegates settings/logging/route
preflight to private `cmd.prepareWeb` in `cmd/web.go`. This is a source
ownership cleanup after FLC-01, not a change to rollback or shutdown logic.
See [Source Ownership Audit](source-ownership-audit.md).
