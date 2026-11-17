# Web dependency bootstrap

BDSPro follows Gitea's explicit bootstrap approach: application dependencies
are initialized in source order, rather than discovered by a DI container.
The Web composition root is `routers.InitWebInstalled(ctx)`, called by
`cmd.serveInstalled()` after `graceful.InitManager()`.

## Startup order

1. The CLI Before hooks prepare the configuration provider and initialize
   `modules/log`. The CLI action initializes the graceful manager.
2. `routers.InitWebInstalled` calls `setting.LoadSettings()`.
3. `routers/common.InitDBEngine(ctx)` initializes the PostgreSQL engine.
   Development migration policy stays in `routers/common/db.go`.
4. `routers/common.InitRedis(ctx)` initializes Redis through `modules/conn`.
5. `routers.NormalHandler()` compiles HTTP routes only after dependencies are ready.
6. `cmd.listen()` starts the HTTP server under the graceful manager.

No `Init()` method is required for stateless modules such as
`modules/json` or `modules/util`. A package is initialized only when it
owns startup state, a resource, or a long-running activity.

## Dependency ownership and failure

- `cmd`: process entrypoint, config preparation, logger, graceful startup.
- `routers`: Web startup order and route composition.
- `routers/common`: Web-specific infrastructure initialization policy.
- `models/db`: PostgreSQL pool, transactions, and termination hook.
- `modules/conn`: Redis client and termination hook.
- `services/*`: business operations; no implicit initialization unless necessary.

Each initializer returns errors to its caller; none invokes `log.Fatal`.
If migration fails, `common.InitDBEngine` rolls back the database engine.
If Redis fails after PostgreSQL is ready, `routers.InitWebInstalled`
closes PostgreSQL. A canceled Redis startup closes Redis as well.
Resources register termination hooks with the graceful manager, and the
corresponding Close functions are idempotent. A successful bootstrap leaves
normal cleanup to graceful termination.

This is not a claim that termination hooks have a deterministic ordering:
the manager currently starts termination callbacks concurrently. Any
future cleanup dependency between resources requires an explicit contract.

The Worker process has its own entrypoint and must not start its queue or
cron automatically from Web initialization.

## Gitea reference

- https://github.com/go-gitea/gitea/blob/main/routers/init.go
- https://github.com/go-gitea/gitea/blob/main/cmd/web.go

BDSPro intentionally returns initialization errors to the caller instead
of adopting Gitea's fatal-on-initialization-failure helper.
