# BDSPro source ownership audit — command, package, file and function

**Historical source-placement audit:** the earlier `WebConfig` handoff
and `PublishLegacyWeb` described below were superseded by
[Canonical Process Settings & Source Ownership](canonical-settings-and-ownership.md)
and [ADR-G02-02](decisions/ADR-G02-02-canonical-process-settings.md).
Keep this report as the evidence ledger for the file cleanup; do not treat
its configuration call graph as the current implementation.

**Status:** refactor implementation scope / compatibility-limited. Pinned
baseline: `dev@f809de461ba1b9bfec8c8867eb8442a99bcd2905` (2026-11-21).
Read this report with the final implementation commit, not as a claim that
all BDSPro directory decisions are already accepted. A source file is
evidence, not its own architectural justification.

## Design force and measurable question

Problem: `cmd/web.go` performed a flat chain of config read, output logger
wiring, Chi registration, process globals, graceful manager, and resource
bootstrap. Small cross-domain forwarding files made the real function
placement harder to discover. The question is not how many functions we can
split, but whether each file has an **owner, public or private contract,
independently named responsibility, observable side effect**, and real caller.

A structural change is justified only when it removes a fake abstraction,
restores a missing responsibility boundary or reduces traversal cost without
creating a new source of state or breaking the runtime contract.

## Evidence: Gitea-style entrypoints vs BDSPro

Gitea pinned SHA `4a9a53d79abc8ab339b18d6c8c59004d7c9d5126`:

- `cmd/web.go:261-298`, `runWeb`: process manager and process setup,
  delegates the server startup to `serveInstalled`.
- `cmd/web.go:168-242`, `serveInstalled`: Web-specific initialization and
  route construction; `cmd/web.go:332+`, `listen`: HTTP listener coordination.
- `modules/setting/setting.go:219-240`, `LoadSettings`: settings-driven
  `initAllLoggers()` and subsystem loaders.
- `modules/setting/log.go:192-217`, `initAllLoggers`: logging settings
  activate a resource manager. This is not a command-line helper.
- `services/context/api.go` and `routers/init.go`: HTTP request context and
  route tree belong outside the command package.

Gitea does **not** prove that more files are always better: its scale,
installation flow, template renderer, file storage and Git services create
different constraints. We adopt **command composition vs settings activation
vs resource ownership**, not Gitea's many global values or fatal policies.

Independent comparisons:

- PocketBase `core/base.go:215-251`: constructor sets config/defaults;
  `core/base.go:412-466`: `Bootstrap` owns the sequence; `475+` handles
  clearing previously opened resources. It is a different design from Gitea
  and demonstrates that a function may be long if all lines belong to one
  lifecycle phase.
- Caddy `caddy.go:98-114`: `App.Start/Stop` contract; `419-482`:
  provision/start, unwind started modules on failure. We do not copy its
  dynamic modules into BDSPro.
- Grafana `pkg/setting/setting.go:1574-1584`:
  `NewCfgFromINIFileWithoutGlobals` is a separate pure config read.
  BDSPro's `LoadWebConfig` and `config check` retain equivalent
  no-publication semantics.

Pinned external references:

- https://github.com/go-gitea/gitea/blob/4a9a53d79abc8ab339b18d6c8c59004d7c9d5126/cmd/web.go#L261-L298
- https://github.com/go-gitea/gitea/blob/4a9a53d79abc8ab339b18d6c8c59004d7c9d5126/modules/setting/log.go#L192-L217
- https://github.com/go-gitea/gitea/blob/4a9a53d79abc8ab339b18d6c8c59004d7c9d5126/modules/setting/setting.go#L219-L240
- https://github.com/pocketbase/pocketbase/blob/7f2773ec234b62d0b80ca47c508397cb3d0da3ea/core/base.go#L412-L485
- https://github.com/caddyserver/caddy/blob/edcdfa0ed88f7dcc1c53720d25218fde9d63eb1e/caddy.go#L419-L482
- https://github.com/grafana/grafana/blob/ab4ccc30c29023b0373e646dd6fd8a29348895ab/pkg/setting/setting.go#L1574-L1584

## File placement rules — based on responsibility, not line count

**Command file** `cmd/<command>.go`: defines CLI flags/Before/Action and
the process-level composition for that command. `cmd/main.go` owns CLI
root, signal wiring and common CLI contracts. Do not put config parser,
network client, domain service or a standalone logger manager in `cmd`.
A CLI `config check` action *does* belong under `cmd` because its caller
is the CLI and it writes command output; name the file `config_check.go`,
not an ambiguous `config.go`.

**Module file** `modules/<capability>/<subject>.go`: reusable mechanism.
`modules/setting/setting.go` is the typed configuration entrypoint;
`config_provider.go` adapts INI; `path.go` resolves paths;
`log.go` owns parsing the log section plus an *explicit* InitLoggers
handoff to the logger. `modules/log` owns writer creation, file descriptors
and Close; `modules/graceful` owns process/HTTP shutdown. A one-line
forwarder with no contract or adaptation is not automatically a module.

**Model file** `models/<domain>/<entity>.go`: domain values and domain
invariants. `models/db` intentionally contains PostgreSQL/SQLC primitives,
transaction context and schema migration tools. It is not an HTTP router
or settings parser. Distinguish `models/organization/team.go` from
`services/org/team.go` by reading the bodies rather than relying on names.

**Service file** `services/<domain>/<operation>.go`: business orchestration,
authorization-sensitive data access and transaction work; it accepts
`context.Context` and does not decide HTTP status or process signal.
`services/context` is an established **HTTP request context adapter**
following the Gitea naming convention, not a domain service package.
It should be tracked as a special transport-facing exception; do not
blindly move it to `modules` without tracing its many consumers.

**Router file** `routers/api/v1/<domain>/<entity>.go`: request input,
authentication/authorization guard and mapping service errors to HTTP.
`routers/init.go` composes route tree and Web resource startup steps;
`routers/common` houses shared route policies, not generic helpers.

**Util file** `modules/util/<name>.go`: only a useful shared behavior with
multiple concrete consumers or a cross-domain contract.
`modules/util/path.go` had `IsExist` called solely by
`modules/setting/config_provider.go`, and simply wrapped `os.Stat`.
It added a file and call hop without a new invariant. The file and its
test can be removed; the INI adapter now checks file existence directly,
with its existing file/missing-file parity tests.
`modules/util/error.go` is different: it defines a shared safe error
classification consumed by domain models and HTTP adaptation. It has a
real cross-domain contract and is **not** removed.

**Names and file count**: do not create `helper.go`, `common.go`,
`runtime_config.go`, `web_graceful.go` merely to make the main file
shorter. Retain such files only when they own a distinct capability,
platform difference, state, invariant or independently used API.
An internal helper should usually be in the file of its only caller.
A long cohesive file is preferable to several unowned wrappers.

## Concrete source-to-target move ledger

| Before | Evidence | After / decision |
| --- | --- | --- |
| `cmd/runtime_config.go` | Sole `configureCommandLogger` copied five validated fields into `modules/log.GetManager().ConfigureOutputs` | Remove file; explicit `setting.InitLoggers(cfg.Common.Log)` in `modules/setting/log.go` forwards to `modules/log` owner. Validation of the configured log section remains with `loadLogGlobalFrom`; `modules/log` checks its own writer input contract |
| `cmd/web_graceful.go` | Sole `runHTTP` delegates with unchanged args and errors | Remove file, call `graceful.HTTPListenAndServe` from `cmd.listen`. Manager and server retain their ownership |
| `cmd/helper.go` | Mixed signal setup, root stderr CLI Before, argv validation; all called in cmd | Move functions to `cmd/main.go` (one CLI root owner); remove generic filename, preserve signatures |
| `cmd/config.go` | `newConfigCommand` and `runConfigCheck` are CLI actions, not INI parsing | Rename to `cmd/config_check.go`; **keep under cmd** |
| `cmd/config_test.go` | Only tests config check CLI action | Rename to `cmd/config_check_test.go` |
| `cmd/web.go` flat `runWeb` | Loaded config, created logger, compiled routes, published legacy globals and initialized process in one sequence | Private `prepareWeb` **in the same file** owns Web command preflight, `runWeb` owns process lifecycle, `serveInstalled` owns resource and listener orchestration, `listen` owns process tracking and HTTP invocation |
| `modules/util/path.go` | `IsExist` has one in-repo call from INI adapter | Remove, use `os.Stat` and `errors.Is(err, os.ErrNotExist)` within setting adapter |

## After refactor, the readable runtime trace

```text
cmd.NewMainApp                   [main.go]     CLI registration only
  -> cmd.runWeb                  [web.go]      process entry
       -> cmd.prepareWeb         [web.go]      Web command preflight
          -> setting.LoadWebConfig              pure parse+validate; INI read once
          -> setting.InitLoggers                explicit side effect, modules/log owns file
          -> routers.PrepareWebRoutes           validate route shape, no external resources
          -> setting.PublishLegacyWeb          temporary one-time HTTP compatibility
       -> graceful.InitManager                  process lifecycle owner
       -> cmd.serveInstalled     [web.go]      register activity and init resources
          -> routers.InitWebInstalled           startup.Run DB → Redis → JWT
          -> routers.NormalHandler              attach borrowed Redis admission
          -> cmd.listen             [web.go]
             -> graceful.HTTPListenAndServe     protocol/server owner
```

`cmd.runMigrate` independently reads `LoadMigrateConfig` and calls
`setting.InitLoggers` for command output, then performs migration.
`cmd.runConfigCheck` only calls pure `setting.CheckWebConfig`, never
`InitLoggers`, never opens network resources, and never publishes globals.

## Why these exact lines belong where they do

1. `cfg, err := setting.LoadWebConfig(...)` in
   `cmd.prepareWeb`: CLI knows flag+environment inputs; setting owns parse.
   It must not read files itself.
2. `setting.InitLoggers(cfg.Common.Log)`: only after validation; explicit
   side effect, no logger construction inside CLI. `modules/setting/log.go`
   maps validated config to `modules/log.ConfigureOutputs`. This mirrors
   Gitea settings-driven logger activation, but BDSPro keeps the pure
   `LoadWebConfig` for `config check`.
3. `routers.PrepareWebRoutes(cfg)`: static HTTP program contract checked
   before publishing HTTP globals or initializing DB/Redis.
4. `setting.PublishLegacyWeb(cfg)`: not a new architecture, a transition.
   It is still one-time mutable global state and should eventually be removed.
5. `graceful.InitManager(...)`: must happen before resource constructors
   that register termination callbacks. It is the process lifecycle owner.
6. `cmd.serveInstalled`: preserves `startup.Run`, DB/Redis idempotent
   owner closers, cancellation and HTTP drain, with no new retry manager.
7. `os.Stat(file)` in INI adapter: here a direct standard-library call
   expresses the intent better than a one-use helper at `modules/util`.
   Existing `TestConfigProviderRejectsMissingFile` preserves
   `errors.Is(err, os.ErrNotExist)`.

## Decisions we did NOT make

- We did not remove global HTTP settings yet; `services/context` and
  `routers/api/v1/utils` still read them. Deletion needs a separate
  request-context contract and full route integration tests.
- We did not remove `ConfigProvider` just because it has one INI backend.
  Evaluate real code hop reduction and `IsSet` behavior against the
  existing provider contract first.
- `LoadMigrateConfig` still loads common Graceful settings even though
  the migrate command does not need HTTP. That command-specific validation
  coupling is real, but is **OPEN** until command policy is decided.
- `modules/util/random.go` and `modules/validation` appear to have no
  active runtime in-repo callers in current code search; deleting exported
  helpers is a separate compatibility decision, not included in this patch.
- `services/context` is transport-facing but currently placed beneath
  services as in Gitea. A directory move across many importers has not
  earned its migration cost based only on path aesthetics.
- Do not infer that file count, call graph depth, throughput or resource
  overhead improved until measured at the final checked commit.
- Changing configuration/stop policy, migrations, business API and
  resource singleton ownership is **outside this layout refactor**.

## Acceptance gates and developer experience tests

1. `go build ./...`, `go test -race ./...`, `go vet ./...`.
2. `config check`: same output, no logger/file resource side effects.
3. `web`: same config source, log writer and route contracts; existing
   PostgreSQL/Redis startup, Docker build and HTTP smoke unaffected.
4. `migrate`: same run mode policy and migration source; disposable DB CI.
5. `NewConfigProviderFromFile`: same missing-file error identity and
   file/INI parity tests after inlining the sole util wrapper.
6. Review call graph in `cmd/web.go` and count meaningless forwarding
   functions/files removed; confirm no alternative config authority was
   introduced.
7. Final CI must run against the **final refactored commit** before
   marking VERIFIED.

The source is the arbiter. If tests pass but the package still hides a
different side effect, report the remaining cost; do not equate CI pass
with an architectural proof.
