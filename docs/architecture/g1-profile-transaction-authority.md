# G1.2c.2 — Gitea-style profile service ownership and transaction-ordered authority

Baseline: BDSPro `dev@6f7a00b055090afac35f0fd0d915a63f5c3f5eb5`.
Gitea source comparison: `main@edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76`.

References:
- [Gitea services/user/update.go](https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/services/user/update.go)
- [Gitea routers/api/v1/user/settings.go](https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/routers/api/v1/user/settings.go)
- [BDSPro services/profile/profile.go](https://github.com/datvtph41107/bdspro/blob/6f7a00b055090afac35f0fd0d915a63f5c3f5eb5/services/profile/profile.go)
- [PostgreSQL 18 explicit locks](https://www.postgresql.org/docs/18/explicit-locking.html)

## The real gap

Previously `PATCH /users/me/profile` and `PATCH /users/me/settings`
used first-party JWT + live-session verification only at the HTTP boundary.
`services/profile.Update` then ran `UPDATE` and `GetProfile` as two
independent pool operations, outside a shared transaction.

If revocation committed between HTTP admission and the subsequent SQLC UPDATE,
a previously authorized in-flight PATCH could still mutate the account.
The transaction could also return a profile projection read after an
interleaving write by another request.

## Implemented ownership and authority

```text
Gin route (existing Authenticate and RequireSession)
    -> user.ProfileHandler.Update / UpdateSettings
       (bind JSON, use ONLY signed AccountID + SessionID)
    -> services/profile.Service.Update / UpdateSettings
       (validate PATCH and coordinate identity + account models)
    -> db.WithTxResult (ONE PostgreSQL transaction)
       -> existing LockActingSession FOR UPDATE (owner, live, expiry)
       -> models/account.UpdateProfile / UpdateSettings (SQLC write)
       -> models/account.GetProfile (SQLC projection in same transaction)
       -> COMMIT; publish response only if commit succeeds
    -> services/convert + APIContext response
```

This follows the **reason** for Gitea's `services/user.UpdateUser`:
a service owns a real cross-model lifecycle and response outcome, while
HTTP remains focused on binding/responding. We **do not** change Gin to Chi,
XORM, or use Gitea process-global database access.

`services/profile.Show` remains a small read delegate for the existing
handler to use consistently. There is no new global service, dependency
registry or response wrapper. Account model methods still own the
account-ID-scoped write predicates; the existing `models/auth` error
`ErrSessionUnavailable` remains a sanitized 401 public domain error.

## Commit-order guarantees (and limits)

A PATCH whose transaction starts its session check after a revoke commits
is rejected without applying the write. A PATCH which obtains the session
row lock first may finish and commit; revocation then continues and takes
effect. Revoke-one, revoke-all and recovery write/delete the session row,
so PostgreSQL enforces this order with conflicting row locks.

The acting-session lock is the **existing** SQLC `LockActingSession`
using `FOR UPDATE`; it is not a newly invented middleware check.
We do not acquire account-level `FOR UPDATE` in this profile transaction.
The response projection is read in the same transaction, then emitted only
after the transaction commits. The operation still uses PostgreSQL's default
READ COMMITTED semantics, not serializable isolation across unrelated rows.

Natural session expiration is checked at lock time, not continuously until
commit. A just-started PATCH may finish after wall-clock expiry. The
read-only GET endpoints have the G1.2b admission contract; this PR does
not add row locks to ordinary GETs.

## Cost and performance budget

| Case | Before | After |
| --- | --- | --- |
| Profile/settings PATCH after HTTP admission | 1 SQL UPDATE + 1 SQL profile SELECT, on separate acquisitions | 1 session-lock SELECT + 1 UPDATE + 1 profile SELECT, in one transaction |
| SQL statements inside PATCH service | 2 | 3 |
| Pool acquisitions inside PATCH service | 2 | 1 transaction acquisition |
| Additional DB transaction | none | one, with lock duration through commit |
| Invalid payload (before DB) | 0 | 0 |
| Stale/revoked acting session | MAY write when HTTP gate was passed earlier | session lock fails; no profile/settings SQL write |

The service reduces pool acquisition count but **adds SQL and a lock**.
No assertion is made that latency or throughput improved. Benchmark p95/p99,
pool saturation and concurrent revocation workload before production SLO claims.

## Tests and acceptance

- HTTP unit tests simulate real SQLC DBTX inside a staged `ExecTx` fake,
  validate allowed PATCH, invalid inputs, cross-account body injection,
  revoked and foreign sessions, rollback on simulated commit failure, and
  sanitized infrastructure errors.
- Disposable PostgreSQL integration test uses pinned backend connections
  and `pg_blocking_pids` to prove both interleavings:
  revocation wins -> profile update rejected; profile lock wins ->
  update commits first and revocation waits. No sleeps stand in for proof.
- Preserve existing first-party HTTP, OAuth, membership, Asynq SIGKILL,
  canonical migration/SQLC, race, Docker and HTTP smoke gates.

## Deliberate non-goals

This is **not** a global request transaction or a blanket rule to lock
every session on every query. Team ownership changes and targeted revoke-one
need their own transactional authorization policy and tests; profile PATCH
alone does not resolve those races. Nor does it introduce account suspension,
a JTI denylist or a generic service layer around every model function.
