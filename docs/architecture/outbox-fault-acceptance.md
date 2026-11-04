# Outbox fault-injection acceptance: PostgreSQL / Asynq seam

BDSPro uses PostgreSQL as the durable job-intent owner and Redis/Asynq as the
task transport. The enqueue is intentionally executed while an outbox row is
locked. There is **no distributed transaction** across PostgreSQL and Redis.

## Verified scenarios

The disposable PostgreSQL acceptance tests in
`services/worker/outbox_fault_injection_test.go` create a unique schema,
copying the **migrated** `public.job_outbox` schema via PostgreSQL `LIKE ...
INCLUDING ALL`. A separate pgxpool uses that schema through `search_path`,
so parallel package tests cannot read or consume the same pending rows.

1. Simulated success in the task transport followed by a forced transaction
   rollback: `queued_at` must remain null, `Dispatch` returns zero, and
   retry of that pending row can accept `asynq.ErrTaskIDConflict` before
   committing `queued_at`.
2. Two dispatchers compete for one row. While the first holds a transaction
   lock and is blocked inside `Enqueue`, the second must return zero due to
   the existing SQLC `FOR UPDATE SKIP LOCKED`, not block or enqueue twice.

Task enqueuers are deterministic fakes at the existing transport boundary.
The database transactions and row locks are genuine PostgreSQL. Tests do
**not** prove distributed exactly-once execution, task ID retention after
Asynq's own expiry, or worker-side idempotency. Job handlers still require
at-least-once-safe semantics.

## Unresolved design tradeoff

Current `Dispatch` makes an external Redis request (up to three seconds per
task, ten tasks per batch) while holding PostgreSQL row locks. This is a
conscious consistency/latency tradeoff, not a new Gitea-style queue mechanism.
Do not move enqueue outside the lock without adding an explicit durable
claim/lease/recovery design and discriminating concurrency/failure tests.

Run with `BDS_TEST_MODE=disposable`, `BDS_TEST_DB_URL` targeting the migrated
`bdspro_test` database. Without that environment, integration tests skip.
