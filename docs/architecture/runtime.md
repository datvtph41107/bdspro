# Runtime — process, PostgreSQL, Redis, HTTP and workers

`main.go` is the executable entrypoint. `run(ctx)` loads `config.Config`,
sets up logging, opens and verifies the PostgreSQL pool, runs migrations,
constructs `db.Store`, connects Redis, starts the Asynq worker, constructs
`core/router.Server`, and starts the HTTP listener. Shutdown follows
context cancellation. No `bootstrap/` folder is required or active.

`core/router/server.go` owns the one Gin engine, trusted-proxy policy,
global request middleware, CORS wrapper, idle/read/write timeouts and
explicit route composition. It is application composition inside `core`,
not a generic reusable routing framework.

`core/errors` carries classified safe public errors and retained internal
causes. `core/response` writes HTTP JSON. Structured request logs must
use the matched route pattern, never a raw URL containing verification
secrets. Security denial monitoring may use Redis independently from rate
limiting; monitoring failures cannot grant a request previously denied.

`worker/` owns Asynq Redis transport, retries and queue lifecycle;
`jobs/` owns verification-email execution. `api/service` expresses the
scheduling intent through the narrow VerificationTaskScheduler interface.
A PostgreSQL transaction does **not** atomically commit an Asynq enqueue.
The current register flow schedules during `RegisterTx` before PG commit;
that risk requires a separate consistency decision, not a style refactor.

Redis state for OAuth, admission policies and queues is security/operational
state, not automatically a disposable cache. No cache Facade or driver
registry should be introduced without concrete needs.
