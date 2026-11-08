# G1.2c.1 — Transaction-ordered first-party revoke-all

## Scope and source evidence

G1.2b guards authenticated Gin requests by looking up the access token's
persisted session. This rejects a request starting **after** revocation commits,
but a request already admitted may reach a model operation later. Prior
`models/auth.RevokeAllSessions` accepted only account ID and used one SQLC
`UPDATE sessions`; a revoked caller that crossed HTTP admission earlier
could still revoke all of the account's remaining sessions.

By contrast, BDSPro already serializes owner/team **membership-changing
writes** in SQL `FOR SHARE` predicates, and the OAuth authorization code
consent transaction locks its authorizing session before issuing authority.
Existing acceptance tests cover those invariants; rewriting them is not part
of this PR.

## Invariant

**An account-wide session revocation may commit only if the session that
initiated it is active in the same transaction.**

`models/auth.RevokeAllSessions` now accepts the signed acting session ID
alongside the account ID. The transaction takes two locks in order:

1. `LockSessionRevocationAccount` locks `accounts(id)` **FOR UPDATE**.
   This serializes concurrent account-wide revocations and session issuance
   through the existing `sessions.account_id` FK KEY SHARE behavior.
2. `LockActingSession` locks the specific session **FOR UPDATE** and
   enforces account ownership, not blocked, not expired, at query time.
3. `RevokeAllAccountSessions` updates all remaining live refresh sessions
   before commit. PostgreSQL rolls back the operation if a step fails.

If another transaction revokes the acting session first, the second step
waits for its row lock and returns no row after the first transaction
commits. The transaction returns `ErrSessionUnavailable`, resulting in
the existing 401 public response, and never modifies other sessions.

The account lock is necessary: locking only each acting session first would
create a deadlock pattern between simultaneous revoke-all requests from
sessions A and B (A holds A while updating B; B holds B while updating A).
It is intentionally not implemented as a Redis distributed lock.

## Evidence

- `models/auth/session_revoke_all_race_integration_test.go`
  uses disposable PostgreSQL and two pinned backend connections,
  `pg_blocking_pids` to prove waiting on revocation, then confirms an old
  actor cannot block another session.
- A concurrent two-device revoke-all test requires exactly one commit and
  one `ErrSessionUnavailable`, with no deadlock.
- Existing HTTP G1.2b, OAuth, credential recovery, and Asynq crash tests
  must remain green; SQLC regeneration, Go race, Docker, HTTP smoke are
  full acceptance gates.

## Deliberate boundaries

This change **does not** add a transaction lock to every API mutation.
An already admitted `PATCH /users/me/profile` still has an admission/commit
race. A separate policy and model transaction check are required for each
write that must be linearized with revocation. The targeted first-party
`DELETE /users/me/sessions/:id` operation still relies on G1.2b admission
and its existing audited transaction; this checkpoint does not claim that
operation has actor-session serialization.

Revoke-all does not revoke previously issued third-party OAuth grants.
Those have separate consent/grant lifecycle. A PostgreSQL connection outage
fails closed without turning infrastructure errors into "invalid session".
No schema migration, JWT format change, new middleware registry, or routing
framework is introduced.

PostgreSQL references:
- https://www.postgresql.org/docs/18/explicit-locking.html
- https://www.postgresql.org/docs/18/transaction-iso.html
