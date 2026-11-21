# Web bootstrap and resource lifecycle — current contract

Baseline: `dev@96a8146`. This document describes the typed bootstrap refactor
that supersedes the earlier Web initialization call graph; historical G02 notes
remain useful evidence but not an active implementation contract.

## Configuration boundary and caller

```text
cmd.NewMainApp       // register CLI only, no config or resource creation
  -> cmd.runWeb
       -> setting.LoadWebConfig(CLI Paths, process ENV)  // parse once, validate
       -> cmd.configureCommandLogger(cfg.Common.Log)
       -> setting.PublishLegacyWeb(cfg)                  // temporary HTTP bridge
       -> graceful.InitManager(ctx, cfg.Common.Graceful.HammerTimeout)
       -> cmd.serveInstalled(cfg)
            -> routers.InitWebInstalled(ctx, cfg)
                 -> startup.Run [PostgreSQL, Redis, JWT]
            -> routers.NormalHandler()
            -> cmd.listen(handler, cfg.Server.HTTPServerAddress)
                 -> graceful.HTTPListenAndServe -> net.Listen
```

`cmd.runMigrate` calls `setting.LoadMigrateConfig`, not Web settings; it
does not construct the HTTP handler, Redis client, JWT maker or graceful manager.
`cmd.config check` calls `setting.CheckWebConfig`, which calls the same typed
Web loader without publishing global settings or opening runtime resources.

## Package owner / lifecycle contracts

- `modules/setting`: INI/ENV/secret reading, paths, defaults, validation,
  typed `WebConfig`/`MigrateConfig`. It must not open runtime connections.
- `cmd`: CLI composition, console/logger setup, process signal handling,
  validated config handoff, graceful manager, HTTP listener invocation.
- `routers`: explicit Web startup steps, route/middleware composition.
  `routers/common` owns the **existing** development migration policy.
- `models/db`: owns one process PostgreSQL pool and its idempotent
  `CloseEngine`; `InitEngine(ctx, source)` does not read configuration globals.
- `modules/conn`: owns one process Redis manager/client; `InitRedis(ctx, address)`
  does not read setting globals. Consumers borrow, they do not close Redis.
- `modules/auth`: `Init(secretKey)` creates a process JWT maker. The maker is
  in-memory state, not a network resource; no generic `Close` interface is added.
- `modules/startup.Run`: validates ordered steps, unwinds **successful**
  stages in reverse on error or cancellation. A failing Init owns its partial cleanup.
- `modules/graceful`: coordinates signal-driven drain and termination.
  DB/Redis register idempotent termination callbacks after successful init.

`setting.PublishLegacyWeb` is a one-time compatibility bridge for HTTP request
middleware and service code still using `setting.API/Server/Security/IPActive`.
It is not a hot-reload API and should be removed only after the HTTP consumers
are migrated. Its mutable map/slices are copied when publishing.

## Failure matrix

| Failure | Expected behavior |
|---|---|
| INI/ENV/secret invalid | No resource initialization; no Web globals published |
| DB.Connect/Ping fails | Pool closes itself; no Redis or listener |
| DB initializes, migration fails | `routers/common` closes DB; later termination hook is idempotent |
| Redis initialization fails | Redis owner closes partial manager; `startup.Run` rolls back DB |
| Auth initialization fails | `startup.Run` rolls back Redis then DB |
| Handler composition/bind fails | `serveInstalled` invokes graceful termination hooks |
| SIGTERM while serving | HTTP Shutdown drains active requests up to HTTP timeout; then Close forces net/http conns closed; hammer remains fallback |
| Multiple Close calls | Process-owned DB/Redis closers remain idempotent |

The process-owned DB and Redis cannot be initialized again in the **same**
process after their Close functions set the closed flag. A failed deployment
must restart the process, not retry bootstrap in-process.

**HTTP drain:** `server.HTTP_SHUTDOWN_TIMEOUT` now flows from the typed
Web snapshot through `cmd.listen` and `cmd.runHTTP` to the HTTP adapter.
On shutdown, `http.Server.Shutdown(context.WithTimeout(...))` stops new
accepts and waits for active net/http requests; when its deadline expires,
`http.Server.Close()` force-closes net/http connections. The existing wrapped
listener and Graceful Manager still track accepted TCP connections, and the
hammer remains a final escalation deadline. Neither `Shutdown` nor `Close`
can guarantee that arbitrary uncooperative handler goroutines or hijacked
(WebSocket) connections have exited. Request handlers must respect their
context and all new background work must have an explicit drain contract.
Do not describe a forced deadline as a successful completion of an operation.

The Graceful Manager invokes termination callbacks concurrently, not in a
guaranteed dependency order. If future background workers depend on DB/Redis
during shutdown, their registered activity must stop and drain before those
resources terminate. Never infer reverse order from termination callbacks.

## Docker / Linux deployment

`docker/setup` ends with `exec "$@"` so the Go process receives SIGTERM.
Compose sets `stop_grace_period: 30s` for its 15s default hammer plus cleanup
allowance. If `GRACEFUL_HAMMER_TIMEOUT` is customized higher, the deployment
deadline must also be increased (and verified with an actual slow-request test).
No database schema or migration policy changes were included in this refactor.

## Acceptance

```bash
go test -race ./...
go vet ./...
go build ./...
docker compose config --quiet
```

CI also verifies migration on disposable PostgreSQL, Docker build and HTTP smoke.
The test suite includes partial-startup, reverse rollback, idempotent cleanup
and active-request drain/deadline tests. Repeat these with real load, Docker
stop and actual production-like deployment settings before production rollout. No test should touch production DB or secrets.

References:
- Gitea ordered initialization: https://github.com/go-gitea/gitea/blob/4a9a53d79abc8ab339b18d6c8c59004d7c9d5126/routers/init.go#L114-L150
- Caddy App.Start/Stop: https://github.com/caddyserver/caddy/blob/edcdfa0ed88f7dcc1c53720d25218fde9d63eb1e/caddy.go#L95-L114
- PocketBase constructor: https://github.com/pocketbase/pocketbase/blob/7f2773ec234b62d0b80ca47c508397cb3d0da3ea/core/base.go#L215-L245
- Go net/http Server.Shutdown/Close: https://pkg.go.dev/net/http#Server.Shutdown
- Docker stop_grace_period: https://docs.docker.com/reference/compose-file/services/#stop_grace_period
