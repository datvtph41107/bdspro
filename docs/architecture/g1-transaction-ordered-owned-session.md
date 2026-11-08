# G1.2c.3 — Transaction-ordered targeted session revocation

## Gap reproduced

At `dev@589bdb4`, `DELETE /users/me/sessions/:id` validated the access
session only at Gin admission, then called `RevokeOwnedSession` with the
account ID and target session ID. The SQLC owner predicate protected the
target but did not re-check the **acting** session during its transaction.
An admitted, subsequently revoked actor could therefore mutate another
session.

Browser logout also used this model function; it now supplies its own
authenticated session as both actor and target.

## Locking contract

The model transaction performs, in exactly this order:

1. `LockSessionRevocationAccount(accountID)`: `accounts FOR NO KEY UPDATE`.
2. `LockActingSession(actorID, accountID)`: `sessions FOR UPDATE`, active
   and unexpired at `clock_timestamp()`.
3. `RevokeOwnedSession(targetID, accountID)`: owner-scoped SQLC UPDATE.
4. `RecordSessionRevoked`: audit evidence in the same transaction; commit.

The account lock is shared with `RevokeAllSessions` and conflicts with
password recovery's account lock. This order avoids the deadlock where
session A locks A then updates B while session B locks B then updates A.
It serializes targeted and account-wide revocations within one account,
an explicit throughput tradeoff. Existing `FOR NO KEY UPDATE` permits
account foreign-key `KEY SHARE` for session issuance and OAuth consent.

`ErrSessionUnavailable` means the signed actor is no longer authorized.
`ErrSessionNotFound` masks absent, foreign, expired or blocked targets.
Infrastructure errors are never converted into authorization errors.
An audit failure rolls back the actual session change.

## Acceptance

- PostgreSQL pinned-connection test proves actor revocation blocks the
  targeted action, then rejects it with no target mutation.
- Simultaneous A->B and B->A targeted revocations yield one success and
  one rejected actor, with no deadlock.
- Targeted and account-wide revocations share lock order.
- Foreign target is invisible; self-revoke succeeds; repeated actions
  from a revoked actor are rejected.
- Existing audited rollback, OAuth consent, password reset, Gin route
  inventory, race, SQLC and Docker gates must remain green.

Cost: each targeted revoke now adds one account-lock query and one
acting-session-lock query within the existing transaction. This is an
authorization guarantee, not a measured p95/p99 improvement.

No new router abstraction, middleware, SQLC statement, migration, or
OAuth grant revocation is introduced. The browser logout's stale actor
failure is reported as unauthorized instead of an infrastructure failure.

Reference: https://www.postgresql.org/docs/18/explicit-locking.html
