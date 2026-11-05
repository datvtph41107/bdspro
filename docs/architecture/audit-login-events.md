# Login audit events — source ownership and operational contract

BDSPro follows two distinct references: Laravel DRX's `UserConnectionHistory`
and Gitea's `models/audit` + `modules/setting` + periodic pruning. It does
not reproduce DRX's Passport middleware chain or create a general-purpose
event bus to perform SQL inserts.

## Boundary

* `routers/api/v1/api.go` declares HTTP admission. `LogLoginResponse`
  captures HTTP outcome only; it does not infer bad credentials from status 401.
* `services/auth.AuthService` owns password, phone OTP and OIDC decisions.
  `recordLoginRejected` runs only when the service knows a credential/identity
  was rejected; an upstream Redis admission failure is a separate event.
* `models/audit.RecordLoginRejected` computes a keyed identifier MAC and
  inserts one typed SQLC `audit_events` row. There is no parallel
  `authentication_failures` table, `LoginAuditService`, or event-bus wrapper.
* `models/auth.NewSession` commits a successful session and
  `account_connection_histories` together. Success is never duplicated in
  `audit_events`. The two datasets have different evidentiary contracts.

## Event schema

`audit_events` is a concrete audit-event stream. The initially supported
`action` is `auth.login.rejected`, with `method` (`password`, `phone_otp`,
`oidc`), constrained `reason`, optional `account_id`, `subject_mac`,
bounded normalized `ip_address` and `created_at`. Future audit actions
require a reviewed migration; do not write arbitrary event JSON.

Unknown accounts have NULL `account_id`. Existing accounts have an owning FK.
The account FK uses `ON DELETE CASCADE`: deleting the account removes its
linked security events; the privacy/incident-hold policy must be revisited
before adding legal holds. No raw passwords, OTPs, email addresses, phone
numbers, bearer tokens, OIDC claims or request body are stored here.
`subject_mac` is HMAC-SHA256 over method + normalized identifier and a
fixed versioned domain separator. Emails normalize to lowercase, phone numbers
retain E.164, and OIDC issuer/subject are **case-sensitive**.

## Settings and failure policy

In `app.ini`, `[audit] HMAC_KEY` is an independently managed secret at least
32 bytes long; non-development startup rejects an absent key. The committed
template key is **development-only** and must not be deployed to production.
The developer mode may temporarily derive the audit key from the JWT signing
key when the audit section is absent, solely for upgrading local configs.
Production overrides use `BDS__AUDIT__HMAC_KEY`.

`[audit] RETENTION_DAYS` defaults to 90, validated between 1 and 3650,
and is currently the shared window for successful connection history and
rejected audit events. It is *a project policy default*, not a legal standard.
One `services/worker.LoginRetention` lifecycle prunes both datasets at worker
startup and hourly; the same `Sweep` is available to the one-shot `bdspro
cleanup` command, which connects directly to PostgreSQL without Redis or
Asynq. Schedule this command externally if retention must continue during
a Redis outage. Do not introduce another Redis-backed queue for this task.
Each sweep is limited to 20 batches of 1000 per table.
PostgreSQL `SKIP LOCKED` handles multiple workers. One dataset's deletion
failure cannot starve the other. The only write index is for expiry until
a concrete investigation read contract and query plan justify more indexes.

A rejected login audit insert is **best-effort**. A 401 must not turn into a
500 because its secondary audit insert fails, and no credential is issued on
a failure path. Such a database failure emits a bounded structured warning
without raw identifier/IP/error text. Successful session/history writes are
**not** best-effort: their transaction must commit before JWTs are released.

## Explicit limitations

* Authentication protocol errors rejected before `LoginOIDC` and IPActive
  requests rejected before `AuthService` are not persisted as audit rows.
  They retain separate structured admission and request-outcome logs.
* Persisting every failed proof causes a DB write per admitted failed login;
  baseline failure-injection and p95/p99 load measurements remain required.
* Rotating `[audit] HMAC_KEY` breaks cross-key correlation of identifier MACs.
  Do not log the key, share it with the JWT signing key in production, or assume
  a stable digest across rotations.
* Retention deletes only primary PostgreSQL data. Backup expiry, restoration,
  DSAR/export policy, device approval and instantaneous JWT revocation are
  separate operational requirements.

## Independent maintenance command

```sh
./bin/bdspro --config /path/to/app.ini cleanup
```

`cleanup` is deliberately one-shot, not a second resident worker or
scheduler. Exit 0 means the bounded sweep completed without a database
error, **not** that all overdue records were removed: each table is capped
at 20,000 deletions per run. An error after partial deletion exits nonzero;
already committed batches are not rolled back and the next invocation can
resume. Use a periodic external scheduler and alert on nonzero exits and
unbounded overdue backlog. A worker crash or stopped scheduler means expired
data can remain past `RETENTION_DAYS`. Expiry of backups is separate.
