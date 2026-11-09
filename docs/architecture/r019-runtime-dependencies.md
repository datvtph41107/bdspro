# R01.9 — CLI dependency ownership and settings profiles

## Source evidence

- Gitea `routers/init.go` coordinates installed subsystem initialization with `mustInit`/`mustInitCtx`. These wrappers **fatal on failure**; they do not own rollback or dependency injection.
- Gitea `modules/setting` owns config loading and domain sections, while `cmd` selects which command needs which dependencies.
- BDSPro uses CLI error propagation instead of killing a process in `mustInit`; this preserves the existing deferred PostgreSQL/Redis/Asynq cleanup. A copy of Gitea's reflection-based fatal wrapper would *reduce* this property.

References:
https://github.com/go-gitea/gitea/blob/main/routers/init.go
https://github.com/go-gitea/gitea/blob/main/modules/setting/setting.go

## R01.9 actual owner graph

| Entry | Settings | External resources | Cleanup owner |
|---|---|---|---|
| `cmd web` | `LoadConfig`, `InitLoggers` | DB → optional dev migration → Redis → router/HTTP | `cmd.runWeb`, `serveHTTP`, then reverse-order defers |
| `cmd worker` | `LoadConfig`, `InitLoggers` | DB → Redis → Asynq enqueue/server → outbox/retention jobs | `cmd.runWorker`, stop dispatch, wait, queue shutdown, enqueuer, Redis, DB |
| `cmd migrate` | `LoadDatabaseSettings` | migration connection only | migration command |
| `cmd doctor` | `LoadDatabaseCommandSettings` | PostgreSQL only | `cmd.runDoctor` |
| `cmd cleanup` | `LoadCleanupSettings` | PostgreSQL only | `cmd.runCleanup` |

Maintenance commands should remain executable with no `[server]`, `[redis]`, `[security]` or `[oauth2]` settings, and do not require a migration URL. `cleanup` alone validates audit retention. The shared `LogGlobalConfig.Validate` establishes one source of log-mode/level/format validation for all commands.

## HTTP lifecycle follow-on

- [R01.9c HTTP readiness and bounded shutdown](r019-http-shutdown.md): reject pre-canceled startup, force-close active connections when graceful drain times out, and capture real TCP negative tests.

## Invariants / not claimed

- `loadSettings` still drives web and worker and validates all current security prerequisites. Command scope is not permission to weaken web auth admission.
- Connection release remains in CLI entrypoints. The Redis manager is now deferred **before** obtaining its first client, so partial failures never skip the owner's release.
- No new global dependency container, import-time side effects, fake `mustInit`, or unusable `modules/graceful` wrapper is introduced.
- `routers.NewServer` still composes several handlers. This PR does not claim that its constructor graph, OAuth initialization or every services/models operation has been normalized.
- Next gate: isolate per-subsystem initialization and verify HTTP ready/shutdown/failure recovery with a dependency-level test harness.

Acceptance: Go 1.27.3 CI, settings tests, command compile, SQLC/route parity, worker recovery and HTTP smoke tests.
