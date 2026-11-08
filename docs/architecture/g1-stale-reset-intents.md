# G1.1c — Reject password-reset jobs from completed recovery generations

## Invariant

A password-reset job requested before a successful account recovery **must
never create a fresh reset proof after that recovery commits**. Outbox TaskID,
Redis task deduplication, SMTP acknowledgement, and clearing existing reset
tokens do not enforce this ordering.

Every account has a PostgreSQL `password_reset_version` counter, initially
zero, introduced by canonical migration 000028. The password-reset outbox
records `{account_id, reset_version}` at request time. The worker generates
the random proof **only at execution time**, then issues its hash through the
SQLC `IssuePasswordResetTokenIfCurrentVersion` query. The query inserts a
token only if the current account version matches the job's version, and
locks the account row `FOR SHARE` until the statement commits.

The existing `models/auth.ResetPassword` transaction holds that account
`FOR UPDATE`; it consumes the proof, updates the password, advances the
version, deletes all outstanding reset tokens and sessions, and revokes
OAuth delegated credentials in a single commit. A concurrent worker proof
insertion must serialize with this transaction:

- **Worker first:** token INSERT commits before recovery obtains FOR UPDATE,
  then recovery deletes the token while advancing the version.
- **Recovery first:** worker waits for the lock, observes the advanced
  version, inserts no proof, and sends no SMTP email.
- **New request after recovery:** captures the new version under the account
  lock and can create a valid proof normally.

Unverified accounts must not create durable reset jobs or reset proof rows.
A zero-row proof insert is a consumed, obsolete job and returns success,
not an Asynq retry storm.

## Compatibility and migrations

The new version 28 is an additive, explicitly versioned migration. Migration
version 27 remains valid for existing installations, but the new worker and
queries require the version-28 schema before deployment. The canonical
migration guard now accepts 27 existing tables at schema version 28 and checks
that the new column exists. It refuses older ledger mismatches.

Legacy Asynq payloads containing only `account_id` cannot establish
whether a password recovery completed *before this migration existed*.
The worker therefore treats a **missing** `reset_version` as an obsolete
task, records a privacy-safe warning and acknowledges it without issuing a
proof. This intentionally drops pending pre-migration reset emails; users can
request a fresh link. Explicit version zero from a new, versioned outbox
intent remains valid until the account's first versioned recovery. No
secret is stored in the payload.

Do not roll back the schema while newer workers or pending versioned jobs
are running; restoring an older worker also restores its unsafe stale-task
behavior. Database migration rollback is not an operational security rollback.

## Acceptance evidence

- `services/jobs/stale_reset_intent_integration_test.go`: disposable
  PostgreSQL proves a stale job sends nothing after recovery, while a newly
  requested job using the advanced version is deliverable.
- The same test holds an account FOR UPDATE while a stale proof INSERT
  races; INSERT must block and return zero rows after the version bump.
- Existing password reset, OAuth revocation, outbox fault injection,
  Asynq process-crash and API acceptance tests must remain green.
- `sqlc-check`, canonical schema checks, `go test -race`, Docker and
  HTTP runtime smoke remain required.

## Deliberate exclusions

SMTP can still accept duplicate messages if its acknowledgement is lost.
A proof issued just **before** recovery may be sent as an unusable email
after recovery; that is safe for credential authority but can confuse users.
There is no immediate revocation of existing stateless access JWTs.
A privileged replay of an old task outside the normal Asynq transport also
cannot bypass the PostgreSQL generation check.
