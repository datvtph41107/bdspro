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
3. With `BDS_TEST_REDIS_ADDR` configured, the production Asynq Redis producer
   first targets a deliberately unavailable port. PostgreSQL must leave the
   row pending. Retrying against live disposable Redis (DB 15, unique queue
   and task ID) must commit `queued_at`, and the next dispatch must do nothing.

Scenarios 1–2 use deterministic fake task enqueuers at the existing transport
boundary; scenario 3 uses a real Asynq/Redis enqueue. PostgreSQL transactions
and row locks are genuine. Tests do
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
`bdspro_test` database, and `BDS_TEST_REDIS_ADDR` targeting disposable Redis
for the real-transport scenario. Without those settings, integration tests skip
rather than mutate non-test databases. The live Redis task in DB 15 is test
fixture data; run against disposable Redis only.
