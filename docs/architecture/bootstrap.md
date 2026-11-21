# Web bootstrap and resource lifecycle — current contract

Baseline for typed bootstrap: `dev@96a8146`, followed by `31f5fb2` and
`45c82c2` for explicit constructors and HTTP drain. FLC-01 hardens route
preflight and startup cancellation; read
[its traced failure-boundary case](flc01-startup-failure-boundary.md)
for precise source, invariants, operational risks and verification gates.
Historical G02 notes are evidence, not the live implementation contract.
**Canonical current settings contract:** [Canonical Process Settings & Source Ownership](canonical-settings-and-ownership.md)
and [ADR-G02-02](decisions/ADR-G02-02-canonical-process-settings.md).
The earlier typed bootstrap was deliberately superseded at user request;
pure validation remains, but runtime resources read one published setting.

## Configuration boundary and caller

```text
cmd.NewMainApp                 // register CLI only
  -> cmd.runWeb
       -> cmd.prepareWeb
            -> setting.LoadWebSettings(args, envs)
                 -> LoadWebConfig                 // one parse, validate, transient typed data
                 -> publishSettings once        // canonical setting package globals
            -> setting.InitLoggers()             // modules/log writer owner
            -> routers.PrepareWebRoutes()        // static route contract; no DB/Redis
       -> graceful.InitManager(ctx)              // reads setting.Common.Graceful
       -> cmd.serveInstalled(preparedRoutes)
            -> routers.InitWebInstalled(g.ShutdownContext())
                 -> startup.Run [PostgreSQL → Redis → Auth]
                     -> common.InitDBEngine(ctx) -> db.InitEngine(ctx)
                          -> db.Connect(ctx, setting.Database.DBSource)
                     -> common.InitRedis(ctx) -> conn.InitRedis(ctx)
                          -> RedisManager.Client(ctx, setting.Redis.Address)
                     -> auth.Init() -> NewJWTMaker(setting.Security.TokenSymmetricKey)
            -> routers.NormalHandler(preparedRoutes) // reads setting.IPActive
            -> cmd.listen(handler)                    // reads setting.Server
                 -> graceful.HTTPListenAndServe -> net.Listen
```

`cmd.runMigrate` calls `setting.LoadMigrateSettings` and `InitLoggers()`;
it publishes only Paths/Common/Database. No HTTP listener, Redis client,
JWT maker or graceful manager is created.
`cmd.runConfigCheck` calls `setting.CheckWebConfig`; it uses the **same pure
loader** but never publishes globals, opens resources or activates a log
writer. Each command loads INI independently in its own process; no subsystem
re-parses the file. This is Gitea-style **runtime global settings**, not an
instance-based DI container.

## Package owner / lifecycle contracts

- `modules/setting` owns **all config discovery, INI/ENV/secret parsing,
  validation and one-time publication**. It does not own DB/Redis resources.
  `WebConfig` is a transient validated value, never a parallel runtime store.
- `cmd` owns CLI args, sequence of process operations and exit.
  `runWeb` stays short because `prepareWeb`, `serveInstalled`, `listen`
  are distinct command lifecycle stages, all kept in `cmd/web.go`.
- `routers` owns static route preflight, startup step composition and HTTP
  middleware admission. It **reads** canonical settings, does not parse them.
- `models/db` owns PostgreSQL pool, `CloseEngine`, and the explicit
  `Connect(ctx, source)` primitive; process `InitEngine(ctx)` reads
  `setting.Database`.
- `modules/conn` owns Redis, its idempotent Close and the reusable
  `RedisManager.Client(ctx, address)` primitive; process `InitRedis(ctx)`
  reads `setting.Redis`.
- `modules/auth` owns JWT maker; process `Init()` reads
  `setting.Security`, but `NewJWTMaker(secret)` stays explicit and testable.
- `modules/log` owns logger writers/file descriptors. `setting.InitLoggers`
  activates it from canonical log settings.
- `modules/graceful` coordinates signal-driven shutdown/drain/termination;
  `InitManager(ctx)` reads the validated hammer deadline.
- `modules/startup.Run` executes init steps, rolls back **successful**
  stages in reverse on error/cancellation. A failing Init cleans up its own
  partially created resource.
- `models/<domain>`, `services/<domain>`, `routers/api` keep domain
  data, transactions/business operations and HTTP mapping distinct.

Only the **configuration access path** changed, not the startup Run/Close,
migration policy, public API, resource singletons or database schema.

## Failure matrix

| Failure | Expected behavior |
|---|---|
| INI/ENV/secret invalid | No resource initialization; no Web globals published |
| DB.Connect/Ping fails | Pool closes itself; no Redis or listener |
| DB initializes, migration fails | `routers/common` closes DB; later termination hook is idempotent |
| Redis initialization fails | Redis owner closes partial manager; `startup.Run` rolls back DB |
| Auth initialization fails | `startup.Run` rolls back Redis then DB |
| Route registration / trusted-proxy preflight fails | Settings may have been published and logger activated, but no DB/Redis/Auth resource is opened |
| Runtime admission composition fails after DB/Redis | `serveInstalled` invokes graceful termination hooks |
| SIGTERM while serving | HTTP Shutdown drains active requests up to HTTP timeout; then Close forces net/http conns closed; hammer remains fallback |
| Multiple Close calls | Process-owned DB/Redis closers remain idempotent |
| SIGTERM during cooperative DB/Redis init | Startup context cancels immediately; completed stages roll back |

The process-owned DB and Redis cannot be initialized again in the **same**
process after their Close functions set the closed flag. A failed deployment
must restart the process, not retry bootstrap in-process.

**HTTP drain:** `server.HTTP_SHUTDOWN_TIMEOUT` is read from canonical
`setting.Common.Graceful` by `cmd.listen`, which passes the duration to
`graceful.HTTPListenAndServe`.
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

## Source placement decision (2026-11-21)

Read [File and Package Ownership Audit](source-ownership-audit.md).

- `cmd/runtime_config.go` had one command-specific adapter calling
  `modules/log` with fields read from `modules/setting`: no independent
  ownership. The explicit log activation is now in `setting/log.go`, where
  Gitea also locates settings-driven logger initialization. The writer itself
  remains in `modules/log`.
- `cmd/web_graceful.go` forwarded one call with no state, contract or error
  transformation; `cmd.listen` now calls `graceful.HTTPListenAndServe`
  directly. Graceful remains the HTTP protocol lifecycle owner.
- `cmd/config.go` was **not** a misplaced configuration parser:
  it implements the `config check` CLI command. It was renamed
  `cmd/config_check.go` to avoid a misleading, broad filename; it keeps
  calling pure `setting.CheckWebConfig`.
- `cmd/helper.go` mixed CLI argument checks, signals and console bootstrap
  in a catch-all. These entrypoint functions now reside in `cmd/main.go`
  with their only process-level owner; no `modules/util` migration.
- `modules/util/path.go` contained an `IsExist` abstraction used only by
  the INI adapter. `modules/setting/config_provider.go` now uses `os.Stat`
  and `errors.Is(err, os.ErrNotExist)` directly, with existing config
  provider file/missing-file tests.
- Only the file placement and explicit log activation adapter were
  modified. The singleton resource manager, rollback, run mode migration,
  and HTTP request contracts remain as they were.
