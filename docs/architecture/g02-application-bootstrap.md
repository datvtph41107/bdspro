# G02 — Application bootstrap, mustInit semantics and resource lifecycle

Reference Gitea: bf5c39b310f99b4254266bebee535d3730daba0a.
Baseline BDSPro: dev@ff4411d2e8cc374276065abafbc7d9965da98872.

## Root problem

Gitea routers/init.go:60-76 uses mustInit/mustInitCtx to label failing init functions and halt the bootstrap. Its installed-Web initialization at lines 114-175 is explicitly ordered: settings, storage/cache, database, models/auth, services/queues/background work, then cron. cmd/web.go:261-268 owns graceful.InitManager before installed-Web startup.

BDSPro had two independent startup paths: routers.InitWebInstalled initialized Web settings, PostgreSQL, Redis and auth with hand-coded rollback; cmd/worker initialized separate DB/Redis/Queue settings, PostgreSQL and Redis before task handlers. The two paths lacked one auditable convention for stage errors, completed-step rollback and cancellation. This refactor changes orchestration, not database, authentication, or queue business logic.

## Decision and implementation

- NEN: modules/startup.Run executes named required steps in order, stops on the first error, preserves errors.Is through wrapping, and rolls back completed resources in reverse order on failure or cancellation.
- NEN: resource-owning packages create/close their resources. models/db.InitEngine and modules/conn.InitRedis register graceful.RunAtTerminate; their CloseEngine/CloseRedis are idempotent and are also legal startup rollback actions.
- NEN: the CLI process owns graceful.InitManager and a final DoGracefulShutdown / Done wait. The Web composition root remains routers.InitWebInstalled; Worker composition remains cmd/worker.
- KHONG NEN: copy Gitea's log.Fatal into initialization. In BDSPro, returning errors allows graceful termination callbacks to run and makes failure tests practical.
- KHONG NEN: load every service in both binaries, create an unused cache abstraction, or start queues implicitly because Gitea has them.
- CHUA NEN: introduce a cache service without a concrete cache consumer and invalidation contract; introduce another process-wide queue singleton; or wire placeholder delivery dependencies.

## Dependency graph (actual runtime)

CLI Before: setting.InitWorkPathAndCommonConfig -> setting.InitLoggers

Web: cmd.runWeb -> graceful.InitManager -> routers.InitWebInstalled -> startup.Run [setting.LoadSettings -> common.InitDBEngine -> common.InitRedis -> auth.Init] -> routers.NormalHandler (IPActive, Swagger, Chi routes) -> cmd.listen.

Worker: cmd.runWorker -> graceful.InitManager -> startup.Run [setting.LoadWorkerSettings (DB, Redis, Queue) -> db.InitEngine -> conn.InitRedis] -> newWorkerHandlerDependencies -> worker.TaskHandlers -> worker.RunProcessor (Asynq Stop then Shutdown).

modules/graceful coordinates process shutdown; modules/process tracks managed activities; modules/queue owns Asynq processor lifetime. A Web process does not consume worker tasks implicitly. The Asynq worker owns its own Redis transport connections; modules/conn owns the separate shared Redis client borrowed by IPActive.

## Known gaps that this patch does not conceal

- cmd/worker_deps.go currently returns an explicit error: worker: business delivery dependencies not configured. The worker therefore cannot serve real business tasks until VerifyEmailDelivery, ResetProofIssuer, PasswordResetDelivery and TeamInviteDelivery have actual production providers. Do not replace them with nil handlers or mocks.
- BDSPro currently has no modules/cache; Gitea's modules/cache/cache.go:20-30 initializes a concrete cache, which is not an automatic requirement for BDSPro.
- Gitea's modules/queue/manager.go:92-114 creates managed queues from named settings. BDSPro intentionally uses its existing Asynq-backed worker process instead.
- A stage failing after partially creating its own resource must clean up that partial state. startup.Run only rolls back stages whose Init returned nil.

## Verification gates

1. No resource initialization may occur before required settings are validated; Gitea-style fail-fast error contains the failing stage name.
2. DB succeeded, Redis failed -> DB is released; DB and Redis succeeded, auth failed -> Redis then DB are released; a cancelled bootstrap must not start a listener/processor.
3. Successful startup retains resources until graceful termination; no component borrowing Redis may close the process client.
4. Worker must report unconfigured delivery handlers as a startup failure instead of a false-ready state; task drain happens before resource termination.
5. Run gofmt, go build ./..., go vet ./..., CI HTTP/DB/Redis smoke and shutdown/failure-injection checks. Standalone startup logic was tested separately; full repository integration must be verified by CI.

Source links:
- https://github.com/go-gitea/gitea/blob/bf5c39b310f99b4254266bebee535d3730daba0a/routers/init.go#L60-L175
- https://github.com/go-gitea/gitea/blob/bf5c39b310f99b4254266bebee535d3730daba0a/cmd/web.go#L261-L268
- https://github.com/go-gitea/gitea/blob/bf5c39b310f99b4254266bebee535d3730daba0a/modules/queue/manager.go#L92-L114
