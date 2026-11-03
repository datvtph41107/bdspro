# Process lifecycle and failure policy

`main.go` initializes process metadata, installs the CLI, and closes logs.
`cmd/main.go` selects one of `web`, `worker` and `migrate`.

- `web`: settings → PostgreSQL → dev migration → Redis → Gin → HTTP listen.
- `worker`: settings → PostgreSQL → Redis → Asynq processor and SQL outbox dispatcher.
- `migrate`: config path → database-only INI settings → versioned SQL migration.

`models/db.Connect` and `modules/redis.Connect` own connection and ping
cleanup, **not** domain-specific availability policy. Web startup currently
requires Redis because authentication throttling depends on it.
Separating the worker does not remove this web dependency.

`modules/queue` owns Asynq client, processor and logging adapter.
`services/worker` owns outbox delivery policy and task adapters;
`services/jobs` owns the job's business operation.
`SIGTERM`, task execution, duplicate enqueue, failure between Redis and
PostgreSQL commits, multiple worker replicas and graceful drain require
integration/failure-injection tests beyond ordinary unit/HTTP smoke tests.

`modules/log` supports structured events and console/file output.
Console output is stderr; file output requires a writable configured
directory. Logs must not contain OTP tokens, mail secrets or full
authorization headers.

CLI help/version should not require config or external services.
See [configuration](../operations/configuration.md) and
[Docker](../operations/docker.md).
