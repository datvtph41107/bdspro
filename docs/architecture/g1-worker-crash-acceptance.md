# G1.1b — Real Asynq process-crash recovery acceptance

Baseline: G1.1 is merged on `dev@fa786d363f19e873a5cc4e2110f7a3b1eb05adb8`.

## Invariant and experiment

Use disposable PostgreSQL 18 (`bdspro_test`), disposable Redis 7 (DB 15),
and the production `OutboxDispatcher`, `RedisTaskProcessor`, and
`NewPasswordResetHandler`. A unique queue/task ID and a cloned, isolated
PostgreSQL outbox schema prevent contact with other tests.

The test's **file-backed mail sink** represents a mail provider which has
accepted a message. It is deliberately not an SMTP-protocol test and must
never be described as proof of SMTP provider guarantees.

1. Register and verify a disposable account and atomically dispatch one
   password-reset outbox intent to real Asynq/Redis.
2. Start a child test process running the production Asynq processor.
   Its password-reset handler stores a real proof hash in PostgreSQL and
   writes a message to the file-backed mail sink.
3. Block the handler **after the mail side effect** but before returning.
   Confirm the task is active, then SIGKILL the entire child process.
4. Confirm the task still exists in Redis. Start a fresh worker process
   listening on the same unique queue and wait for Asynq's automatic
   lease-expiration recovery, retry scheduling and second execution.
5. Confirm the retained task is completed, the mail sink records two
   deliveries, two independent reset hashes exist, and using the first
   proof revokes both in the real PostgreSQL recovery transaction.

`hibiken/asynq` v0.23.0 creates 30-second leases and its recoverer
scans at 1-minute intervals, with an additional 30-second allowance
for clock skew. The test uses a bounded 215-second wait to cover the
normal recovery cycle. This is why the test runs once in a dedicated CI
step, not in every `go test ./...` or race invocation.

## Explicit scope exclusions

- No claim of exactly-once email delivery or SMTP acknowledgement safety.
- No claim that a password reset job redelivered **after a completed
  recovery** cannot create a fresh reset proof. That is a separate policy
  risk and requires job-intent generation/credential epoch design before
  making a backward-incompatible fix.
- No Redis process restart, full machine power loss or PostgreSQL failover.
- No proof of bounded queue growth, delivery latency SLO or zero lost
  emails with an external provider.
- No change to Go/Gin routing, SQLC schema, OAuth2 contracts, Redis key
  prefixes, or worker production implementation.

## Run manually

Against **disposable test services only**, after canonical migrations:

```bash
BDS_TEST_MODE=disposable \
BDS_TEST_DB_URL='postgres://root:secret@127.0.0.1:5432/bdspro_test?sslmode=disable' \
BDS_TEST_REDIS_ADDR='127.0.0.1:6379' \
BDS_G1_PROCESS_TEST=1 \
go test ./services/worker -run '^TestG1WorkerCrashAfterMailDeliveryAndAutomaticRecovery$' -count=1 -timeout=5m -v
```

The test owns only its UUID-namespaced queue, Redis task and isolated
PostgreSQL schema. A future stricter test should use a dedicated Redis
server instance, not just DB-level isolation, when testing Redis shutdown
or persistence loss.
