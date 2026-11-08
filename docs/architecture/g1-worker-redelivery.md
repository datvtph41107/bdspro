# G1.1 — Worker redelivery and SMTP ambiguity contract

Baseline: G0.1 merged as PR #162; review target is Go 1.27.3, PostgreSQL 18,
Redis 7, Asynq and the existing SQLC-backed outbox. No extra delivery
framework, database migration, or HTTP response change.

## Delivery boundaries

PostgreSQL durably records the intent in `job_outbox`; `services/worker`
dispatches it to Asynq while holding its PostgreSQL row lock. The Redis
enqueue and PostgreSQL commit are not one distributed transaction. Asynq
retries handlers on an error and may redeliver after worker interruption.
SMTP acceptance and Asynq acknowledgement are also not atomic.

Consequently **exactly-once email delivery is not a system guarantee**.
A mail provider can accept a message even if a worker sees a timeout, loses
the response or crashes before recording successful task execution. Retry may
send the message again. Deduplicating only by Asynq task ID does not solve
this second delivery boundary.

## Contract by business operation

| Job | Retry proof behavior | Duplicate external side effects |
| --- | --- | --- |
| Verification email | Existing `verify_emails` record reuses the same secret while valid | May resend the same verification message |
| Team invitation | Signing key + invite ID derives a stable proof; accepted/revoked/expired invites are skipped | May resend the same invitation message |
| Password reset | Each worker attempt creates a new high-entropy proof and persists its hash before SMTP | May send multiple **different** valid reset links; each expires after 15 minutes |

Password recovery consumes the presented reset proof and deletes *all*
other outstanding password-reset proofs belonging to that account in one
PostgreSQL transaction. The same transaction updates credentials, deletes
refresh sessions and revokes the OAuth delegations specified in R38.6.
This protects against a second, already-issued reset link remaining usable
**after a successful recovery transaction**.

This is *bounded safety of the current implementation*, not idempotent
password-reset delivery and not an assertion that an undelivered email can
never be lost. A later redelivery after recovery may create another proof:
a durable attempt-level completion/tombstone would be needed to rule that out.
Do not silently change secret derivation or insert conflict handling without
defining expiry, key rotation, redemption races and backward compatibility.

## Evidence and scope

- `services/jobs/redelivery_test.go` simulates an SMTP provider that accepts
  a message but returns an error to the worker. Verification and invitation
  must retain identical proofs on subsequent execution.
- `services/jobs/redelivery_integration_test.go` runs against migrated,
  disposable PostgreSQL and proves a password-reset retry creates two
  persisted proofs; successful recovery with the first invalidates both.
- `services/worker/outbox_fault_injection_test.go` verifies the
  PostgreSQL/Redis handoff, including real Asynq enqueue recovery (G0.1).

These deterministic tests do not kill an operating worker process, prove
Asynq crash redelivery timing, guarantee SMTP exactly-once behavior, or test
Redis restarts with real in-flight jobs. Those require an isolated process
kill/restart harness, a controlled mail provider and bounded retry deadlines,
not claims inferred from unit mocks.

## Follow-on gates

1. Exercise task execution interrupted after handler side effects but before
   acknowledgement with a disposable worker process, Redis and mail sink.
2. Verify stopping a worker drains or retries in-flight tasks without losing
   the database intent.
3. Determine whether multiple simultaneously valid reset links are an
   accepted recovery UX/security policy. Do not equate regenerating a secret
   with exactly-once safety.
4. Verify time-bounded recovery and queue growth under a real Redis outage.

The G1.1 tests are guarded by `BDS_TEST_MODE=disposable` and a
`bdspro_test` PostgreSQL database. They do not run against production.
