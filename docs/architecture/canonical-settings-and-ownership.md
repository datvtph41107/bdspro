# Canonical Process Settings & Source Ownership — BDSPro

**Decision owner:** BDSPro engineering (user-approved direction, 2026-11-21)
**Baseline:** `dev@b1ee221c807f5abdbdfd4f3bcd76d17375a04ad7`
**Scope:** canonicalize the **current** Web/migrate runtime settings and document
owners for `cmd/`, `modules/`, `models/`, `routers/`, `services/`. Do not
infer that a package must be moved simply because its name is unlike Gitea.

## 1. Fundamental requirement: one runtime authority

An INI file plus `BDS__...` overrides defines *one effective configuration*
for a **single BDSPro process**. Web, migrate and config-check are different
CLI commands and do not run together in that process.

`modules/setting` owns the full lifecycle of **configuration values**:

```text
INI file + ENV + __FILE
   -> ResolvePaths / provider / overrides / known keys
   -> LoadWebConfig (pure, typed and validated)
       |-> config check: report only, never publish/open resource
       |-> LoadWebSettings: publish once into setting.{Common,Server,API,
           Security,IPActive,Database,Redis,Paths} globals
                -> cmd.runWeb
                -> routers.PrepareWebRoutes reads setting globals
                -> graceful.InitManager reads setting.Common.Graceful
                -> routers.InitWebInstalled() [startup.Run]
                     -> common.InitDBEngine() -> db.InitEngine()
                          -> db.Connect(ctx, setting.Database.DBSource)
                     -> common.InitRedis() -> conn.InitRedis()
                          -> RedisManager.Client(ctx, setting.Redis.Address)
                     -> auth.Init() -> NewJWTMaker(setting.Security.TokenSymmetricKey)
                -> routers.NormalHandler(preparedRoutes) reads IP policy
                -> cmd.listen reads setting.Server and shutdown deadline

LoadMigrateConfig (pure DB+common subset)
   -> LoadMigrateSettings: publish Common/Database/Paths once
   -> cmd.runMigrate reads setting.Database, not Web/Auth/Redis resources
```

`WebConfig` and `MigrateConfig` are **ephemeral typed build results**, not
second live stores: no globals of these structs; they are not passed through
router/resource startup. `CfgProvider`, `InitWorkPathAndCommonConfig`,
`LoadSettings`, `LoadDBSetting` and `PublishLegacyWeb` were abandoned as
competing publication APIs. No code parses the config in DB/Redis/Auth.

`publishSettings` enforces one publication per process, shared by the
mutually exclusive Web/migrate command. It commits settings only **after**
all relevant parse/semantic validations succeed; repeated publication returns
an error and must not overwrite the process state. The Web input's mutable
IP policy map and server slices are copied on publication. Published exported
settings are a startup-time convention: **Go does not make these variables
immutable**. Runtime mutation/hot reload is explicitly unsupported.

Note: static route preparation runs **after** settings publication but
**before** DB/Redis resources open. A programming error in route registration
may leave published values and a logger inside the exiting process; it does
not leave a newly opened PostgreSQL/Redis client. This is the deliberate
tradeoff for a single settings authority. Do not claim the entire preflight
is side-effect-free (logger activation is an effect).

## 2. Why this design, not an ideology about DI

| Alternative | Real advantage | Actual cost / risk |
| --- | --- | --- |
| Gitea-style process settings (chosen) | `InitDBEngine(ctx)`, `InitRedis(ctx)` call signatures; settings remain with one package owner and no multi-hop config handoff | Hidden temporal dependency: `LoadWebSettings` must happen before resource Init; tests with process globals require isolation; not a multi-instance runtime |
| Pass `WebConfig` across `cmd → routers → common → resource` (previous) | Explicit startup input; different values can be constructed in a test | BDSPro already stores DB/Redis/JWT in global process singletons and HTTP still uses setting globals; two access paths, more parameter forwarding without end-to-end instance ownership |
| App/DI container per server instance | Multiple isolated Web/DB/Redis instances, easier substitution for some tests | Requires owning connection lifecycle, token/DB getters and HTTP request dependencies; premature for one-process BDSPro |
| Read INI directly in each subsystem | Very short setup call sites | Repeated I/O and secret handling; inconsistent validation and precedence; **rejected** |

A *primitive* is different from a process subsystem: `models/db.Connect(ctx,
source)` remains explicit, and `RedisManager.Client(ctx, address)` remains
explicit. They are reusable without process settings. Only process-owned
`InitEngine(ctx)`, `conn.InitRedis(ctx)` and `auth.Init()` read globals.

## 3. Source evidence and limitations

**Gitea pinned commit**
`4a9a53d79abc8ab339b18d6c8c59004d7c9d5126`:

- [setting/database.go:19-54](https://github.com/go-gitea/gitea/blob/4a9a53d79abc8ab339b18d6c8c59004d7c9d5126/modules/setting/database.go#L19-L54):
  package-level `Database`, `loadDBSetting(CfgProvider)`.
- [setting/setting.go:110-165,219-240](https://github.com/go-gitea/gitea/blob/4a9a53d79abc8ab339b18d6c8c59004d7c9d5126/modules/setting/setting.go#L110-L165):
  ordered common/config loading; `LoadSettings()` and logger activation.
- [routers/init.go:114-145](https://github.com/go-gitea/gitea/blob/4a9a53d79abc8ab339b18d6c8c59004d7c9d5126/routers/init.go#L114-L145):
  composition root calls setting load, then subsystem init.
- [models/db/conn.go:42-55](https://github.com/go-gitea/gitea/blob/4a9a53d79abc8ab339b18d6c8c59004d7c9d5126/models/db/conn.go#L42-L55):
  `GlobalConnOptions()` reads package settings.
- [models/db/engine_init.go:48-77](https://github.com/go-gitea/gitea/blob/4a9a53d79abc8ab339b18d6c8c59004d7c9d5126/models/db/engine_init.go#L48-L77):
  `InitEngine(ctx)` uses those options, initializes a process DB engine.
- [setting/redis.go:6-18](https://github.com/go-gitea/gitea/blob/4a9a53d79abc8ab339b18d6c8c59004d7c9d5126/modules/setting/redis.go#L6-L18):
  shared Redis configuration can act as a default across subsystems.

**Deliberate BDSPro differences**: no Gitea installation workflow, dynamic
ConfigProvider saving, XORM or extra Git/SSH/queues. Unlike Gitea's
`mustInit/log.Fatal`, BDSPro keeps `startup.Run` errors, reverse rollback,
idempotent DB/Redis Close, Graceful HTTP drain and Docker signal behavior.

Independent source contrasts:
- [Grafana pure INI parse](https://github.com/grafana/grafana/blob/ab4ccc30c29023b0373e646dd6fd8a29348895ab/pkg/setting/setting.go#L1574-L1584):
  pure configuration inspection without changing global state.
- [PocketBase BaseApp constructor](https://github.com/pocketbase/pocketbase/blob/7f2773ec234b62d0b80ca47c508397cb3d0da3ea/core/base.go#L215-L245):
  an instance-based alternative; not BDSPro's current one-process contract.
- [Caddy App.Start/Stop](https://github.com/caddyserver/caddy/blob/edcdfa0ed88f7dcc1c53720d25218fde9d63eb1e/caddy.go#L95-L114):
  owner-controlled lifecycle without implying the same config access pattern.

## 4. Directory/package/file ownership contract

Directory names follow the **code's actual producer/consumer**, not visual
symmetry with Gitea. For every function, ask: who calls it, what invariant,
what side effects, who owns Close, and why does this file import this package?

| Directory | Canonical owner | Representative code | Not its responsibility |
| --- | --- | --- | --- |
| `cmd/` | process entrypoints, CLI flags/signal/exit and lifecycle coordination | `main.go:RunMainApp`, `web.go:runWeb/prepareWeb/serveInstalled/listen`, `migrate.go:runMigrate`, `config_check.go:runConfigCheck` | INI parsing implementation, connection internals, domain workflow |
| `modules/setting/` | path resolution, INI adapter, env/secret precedence, static validation, canonical publication | `path.go:ResolvePaths`, `setting.go:LoadWebConfig/LoadWebSettings`, `database.go:LoadMigrateSettings`, `log.go:InitLoggers` | open/close DB, Redis, HTTP listener or business transaction |
| `modules/graceful/` | process manager phases, HTTP draining, termination callbacks | `manager.go:InitManager`, `server_http.go:HTTPListenAndServe` | business transaction rollback |
| `modules/log/` | writer creation, stderr/file outputs, Close; setting only activates it | `manager.go:ConfigureOutputs/Close` | decide CLI mode or parse INI |
| `modules/conn/` | shared Redis manager/client lifecycle | `redis_init.go:InitRedis/ProcessRedis/CloseRedis` | API status, schema migration or config parsing |
| `modules/auth/` | JWT maker cryptography and process maker | `runtime.go:Init/ProcessMaker`, `jwt.go:NewJWTMaker` | user/session SQL transaction, HTTP authorization response |
| `modules/web/` | generic route/handler contract and Chi adapter | `router.go`, `handler.go` | domain service operations, token DB validation |
| `modules/startup/` | ordered init + reverse rollback of completed steps | `startup.go:Run` | choose an application's subsystem list or close resources directly |
| `modules/util/` | truly shared safe error/optional helpers | `error.go:WithPublic/WithCause` | catch-all wrappers or one-use delegators |
| `models/db/` | pgxpool, SQLC queries, transaction primitives, migrations and process DB owner | `conn.go:Connect`, `engine_init.go:InitEngine/CloseEngine` | CLI flags or HTTP JSON |
| `models/<domain>/` | persistent domain data and invariants | `models/organization/team.go:NormalizeTeamName` | open listeners or HTTP response |
| `services/<domain>/` | business workflows, authorization, transaction boundaries and side effects | `services/org/team.go:CreateTeam`, `services/auth/access.go:ResolveAccessToken` | process signal, startup, INI parsing |
| `services/context/` | **existing HTTP request context adapter**, Gitea-like historical placement | `http.go:APIContexter`, `api.go:APIErrorAuto` | standalone business service or config owner |
| `routers/` | application HTTP composition and Web init sequence | `init.go:InitWebInstalled/PrepareWebRoutes/NormalHandler` | DB/Redis Close implementation |
| `routers/common/` | Web-specific shared startup/HTTP policies | `db.go:InitDBEngine`, `redis.go:InitRedis` | duplicate config loader or generic util |
| `routers/api/v1/<domain>/` | HTTP input, guard, status and DTO mapping to service | `org/team.go:CreateTeam` | SQL transaction implementation |

**Naming rules**

- Use `<domain>.go` for cohesive domain behavior, e.g. `team.go`;
  use `<mechanism>.go` for real technical owner, e.g. `server_http.go`,
  `config_provider.go`, `engine_init.go`.
- Do not create `helper.go`, `runtime_config.go` or `base_manager.go`
  merely because a function is long. A private helper with one caller should
  stay next to it; an adapter belongs in its actual consumer or provider.
- Keep `cmd/config_check.go` for the CLI **action**; the parser remains in
  `modules/setting/config_provider.go`.
- `services/context` is a documented transport-facing exception following
  Gitea; a directory move requires a measured import/caller benefit.
- No `routers/web`, `routers/private`, queue/worker or extra domain
  directories without a real active consumer.

## 5. Developer trace: where to write a new line

**New configuration key**: `modules/setting/schema.go` permits it,
corresponding section loader parses/validates it, and the consuming
process-owner reads `setting.<Section>.<Field>`. Test `config check`
without side effects and one startup. Do not read INI in DB/Redis/Auth.

**DB initialization**:
`cmd.runWeb → routers.InitWebInstalled(ctx) → common.InitDBEngine(ctx)
→ models/db.InitEngine(ctx) → db.Connect(ctx, setting.Database.DBSource)`.
`models/db` owns Close; `startup.Run` rollback and Graceful termination
may both call the idempotent closer. Never close the pool from a request handler.

**Business transaction**:
`routers/api/v1/org/team.go:CreateTeam → services/org/team.go:CreateTeam
→ models/organization/team.go:NormalizeTeamName → models/db.WithTx2 → SQLC`.
The router owns status/response; service owns transaction/business policy;
model owns normalized domain value; DB owns SQL execution.

**HTTP request**:
`routers.NormalHandler(preparedRoutes) → services/context.APIContexter
→ routers/api/v1 → services/<domain> → models/db`.
API request context and response are not general-purpose background context.

## 6. Lifecycle / deployment risks preserved

- Fail-fast is a boundary decision; `config check` validates without DB.
  Web refuses startup on invalid settings; DB/Redis failures roll back
  previously completed steps with `startup.Run`.
- Graceful manager is initialized **after publication** and before DB/Redis,
  because their Init functions register process termination callbacks.
- One process publication does **not** imply clean service restart in-place:
  DB/Redis process singletons mark themselves closed after Close. A new
  process is needed to reinitialize after that lifecycle ends.
- `Pool.Close` waits for checked-out connections. HTTP forced close is not
  proof an uncooperative handler has exited; worker/transaction drain tests
  remain OPEN.
- `migrate` is an independent CLI mode: publishes Common/Database only and
  does not open HTTP/Auth/Redis. It still validates common graceful settings;
  revisiting command-scoped validation requires a separate policy decision.
- External exported globals are not language-level immutable; this decision
  establishes **initialization-only** mutation. If hot reload, multiple
  app instances/process-local tests or concurrent config updates become
  requirements, reconsider instance-based configuration.
- Source/line and CI proof must be pinned to the **final implementation SHA**.
  This document's baseline is not a claim that CI or staging has passed.

## 7. Acceptance gates and decision triggers

1. Code compiles and `go test -race ./...`, `go vet ./...` pass.
2. Web config validates once, publishes once; a second publication fails
   without changing global state.
3. `config check` and `LoadWebConfig` do not mutate settings, connect DB/Redis
   or create log files.
4. `migrate` publishes only DB/Common/Paths and preserves migration policy.
5. Route registration still precedes resource init; trusted proxy/Swagger
   behavior and API smoke remain unchanged.
6. DB/Redis/Auth/Graceful resource-owned startup uses one settings authority;
   primitive client constructors remain explicitly parameterized.
7. CI rejects new direct config parsing outside `modules/setting` and
   reverse imports from `modules/setting` into routers/models/services.
8. Full Docker build, SQL migration, HTTP smoke after final commit.
9. No claim of runtime performance improvement without benchmarks.
10. Revisit the choice if a new command needs isolated settings concurrently,
    an instance-based server architecture becomes real, or hot reload is
    explicitly required.
