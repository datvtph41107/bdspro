# BDSPro Register / Asynq reliability checkpoint

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
