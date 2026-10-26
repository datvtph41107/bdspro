# INFRA-01 — Bootstrap, errors, worker/jobs and cache

## Bootstrap / main

`main.go` is the OS entrypoint only. `bootstrap/` owns construction and shutdown ordering: config/logging → PostgreSQL/pool ping/migrations → Redis ping → Store/queue adapters → HTTP + worker runtime → graceful shutdown. It must not contain request/controller business decisions. Do not replace a God main file with one God `bootstrap/run.go`; split connection/migration, active runtime and legacy transports according to actual ownership.

Existing HTTP+worker co-location is preserved; don't force separate microservices until independent deployment/scale warrants it. Legacy gRPC/Gateway code must be identifiable as not part of active `Run`, not silently removed or advertised as running.

## Error language

The catalog in `app/errors` names known application failures and stores safe public messages/statuses. Public error definitions are not SQL errors. `app/http/responses` writes a safe envelope and logs unknown internal failures; new HTTP controllers should not copy `if err -> ctx.JSON` branches. Underlying error causes are retained via wrapping so internal diagnosis is possible, but they must not be emitted to users. Avoid a universal `switch` mapping `pgx.ErrNoRows` to 404 — missing rows may express forbidden mutation, invalid credential, stale verification or missing resource depending on the use case.

`panic` is not a business error channel. Framework recovery is reserved for unexpected panics; expected failures return Go `error`.

## Queue / jobs / scheduling

`worker/` owns Asynq server/client, queue names, retry/delay, task registration and operational logging. `jobs/` owns a task's actual work (currently verification email) so new image, notification, search-index, report and other jobs need not duplicate queue machinery. Service exposes intent through a small interface (e.g. `auth.VerificationTaskScheduler`) when it actually prevents importing Asynq into application code.

A task must identify its payload/version, retryability, idempotency boundary, execution owner and error handling. Do not assume exactly-once execution or a PG+Redis atomic commit.

## Redis state is not automatically Cache

Redis currently supports rate limits and OAuth state as security infrastructure plus Asynq broker. Those are not disposable read caches. A `cache/` component is warranted when there is a real key space, TTL, invalidation and failure policy for derived data. A catch-all `CacheManager`/Facade creates hidden dependencies and security errors.

## Operational contract

Maintain request context and cancellation into SQL, queue and external calls. Deny protected routes before DB operations; do not leak token, OTP, raw URL verification secrets in logs. Actual verification requires disposable integration environments, HTTP smoke, CI and WSL runtime acceptance separately.
