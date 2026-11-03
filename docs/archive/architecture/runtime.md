# Runtime — process, PostgreSQL, Redis, HTTP and workers

`main.go` is the executable entrypoint. `run(ctx)` loads `config.Config`,
sets up logging, opens and verifies the PostgreSQL pool, runs migrations,
constructs `db.Store`, connects Redis, starts the Asynq worker, constructs
`internal/api.Server`, and starts the HTTP listener. Shutdown follows
context cancellation. No `bootstrap/` folder is required or active.

`internal/api/server.go` owns the one Gin engine, trusted-proxy policy,
global request middleware, CORS wrapper, idle/read/write timeouts and
explicit route composition. It is application composition inside `core`,
not a generic reusable routing framework.

`internal/errors` carries classified safe public errors and retained internal
causes. `internal/api/response` writes HTTP JSON. Structured request logs must
use the matched route pattern, never a raw URL containing verification
secrets. Security denial monitoring may use Redis independently from rate
limiting; monitoring failures cannot grant a request previously denied.

`internal/worker/` owns Asynq transport, task handler registration and queue lifecycle.
`internal/jobs/` owns the email verification business operation. `main.go` registers
each task type with its handler. Registration persists a durable outbox row in
the account transaction. The dispatcher enqueues committed rows to Redis.
Delivery is at-least-once and does not guarantee exactly-once SMTP delivery.

Redis state for OAuth, admission policies and queues is security/operational
state, not automatically a disposable cache. No cache Facade or driver
registry should be introduced without concrete needs.
