# Historical register / Asynq reliability checkpoint

> **Historical, pre-outbox behavior.** This document no longer describes the
> active registration process. Current implementation is in
> `internal/db/sqlc/store.go` and `internal/worker/verification_dispatcher.go`.
> Do not copy the pre-commit enqueue implementation below.

Baseline: dev@928f029. This change deliberately retains Asynq and the existing
RegisterTx enqueue callback; it does **not** implement a PostgreSQL outbox.

## Execution

1. HTTP POST /users validates and hashes credentials; RegisterTx inserts party,
   person and account in PostgreSQL.
2. The AfterCreate callback schedules task:send_verify_email in Redis **before**
   PostgreSQL COMMIT. If enqueue fails, PostgreSQL rolls back.
3. Asynq moves scheduled -> pending -> active, calls worker handler, and records
   retry/archive or completion based on the returned error.
4. The handler now reuses an existing unexpired, unused verification token.
   A row lock on accounts serializes token creation for the same account.
5. SMTP failure still returns an error to Asynq. The same verification token
   is reused on retry while valid; a new token is created after expiration.
6. GET /users/verify_email consumes the token and marks the account verified.

## Non-atomic Postgres / Redis boundary

Because Redis enqueue occurs before PostgreSQL COMMIT, a task can survive a
failed COMMIT. ProcessIn(10s) is **not** a correctness guarantee.
Moving Enqueue after COMMIT would instead permit a committed account with no
task when Redis is unavailable. Closing this gap requires an agreed-on durable
handoff (e.g. an outbox), which is outside this change.

SMTP acceptance followed by a network error can produce duplicate email
delivery. Reusing the same token prevents unnecessary records while valid;
it does **not** guarantee exactly-once delivery.

## Redis persistence deployment warning

The Compose change mounts /data and enables AOF with appendfsync everysec.
This is a bounded-loss policy, not zero-data-loss durability.

**Before first applying the new Compose configuration to an existing Redis
container, inspect Asynq scheduled / pending / active / retry / archived tasks.**
Recreating an old Redis container that had no persistent volume can discard
its queue state. Schedule a maintenance window or explicitly migrate the Redis
dataset before recreation; simply adding the volume does not migrate old data.

Read-only checks:

```bash
docker compose exec -T redis redis-cli INFO persistence
docker compose exec -T redis redis-cli LLEN 'asynq:{critical}:pending'
docker compose exec -T redis redis-cli LLEN 'asynq:{critical}:active'
docker compose exec -T redis redis-cli ZCARD 'asynq:{critical}:scheduled'
docker compose exec -T redis redis-cli ZCARD 'asynq:{critical}:retry'
docker compose exec -T redis redis-cli ZCARD 'asynq:{critical}:archived'
```

## Acceptance

```bash
go test -count=1 ./...
go test -race -count=1 ./api ./db/sqlc ./worker
go vet ./...
go build ./...
docker compose config --quiet
```

Verify with a fake SMTP sender: the first failure schedules a retry, repeated
handler executions reuse the same active verification record, and an already
verified account causes no outgoing email. Do not log verification secret codes.

## ENG-GOV-03 — Controlled handoff failure evidence (25/10/2026)

**Problem / source:** `auth.Register` calls `db.SQLStore.RegisterTx`, whose `AfterCreate` callback queues verification with Redis/Asynq **before** the PostgreSQL transaction commits. Redis and PostgreSQL have independent durability boundaries.

**Existing evidence:** `db/sqlc/store_test.go::TestRegisterTxEnqueueFailureRollsBack` uses an injected callback error and proves PG rollback. `worker/processor_test.go` tests successful processing, reused verification tokens on replay, and skipping already-verified accounts.

**New targeted gates:**

- `db/sqlc/register_handoff_integration_test.go::TestRegisterTxRedisTaskPersistsAcrossPostgresRollback`: on disposable GitHub Actions PostgreSQL + real Asynq Redis, a dedicated unique queue receives one `task:send_verify_email` event. Compare a normal commit with an injected callback failure **after successful enqueue**. Read back the task through Asynq Inspector, inspect account/person/party rows, then remove the test queue. This establishes that Redis acceptance and PostgreSQL rollback can coexist; **it does not simulate actual PG COMMIT failure or a network partition**.
- `worker/processor_test.go::TestProcessTaskMissingAccountReturnsRetryableError`: a missing account makes verification handler return `pgx.ErrNoRows`, rather than `nil` or `asynq.SkipRetry`. This is a worker unit test with a fake Store; it does not prove a real Asynq retry has executed.

**Safety:** Integration gate only runs if `GITHUB_ACTIONS=true` and `REDIS_TEST_ADDRESS` is configured, using `openTestDB` which enforces a disposable PostgreSQL test DB. It never touches WSL/production volumes. Every queued task goes to a random unique test queue scheduled one hour out, not the production critical queue; cleanup removes the queue. No production Go logic or migration is changed.

**Interpretation:** Redis task existence is **not** equivalent to account commit. Because worker failures are retryable only within configured Asynq retry policy, orphaned tasks may ultimately be archived; do not claim guaranteed delivery. `asynq.ProcessIn(10s)` is a timing choice, not transactional ordering.

**Decision:** Retain current queue/transaction design pending an explicit requirement for stronger atomic handoff. Do not add outbox, generalized event bus, scheduler or new abstraction based only on this reproducible failure mode. If the product requires `committed account ⇒ durable verification intent`, first specify how to handle duplicate tasks, eventual delivery, backfills and operator recovery, then evaluate an outbox with tests.

**Acceptance scope:** CI must show both named tests executed (not skipped), existing RegisterTx tests, Go tests, migration guard, build, smoke, vet and race detector successful. A green CI does not establish WSL or provider E2E acceptance.
