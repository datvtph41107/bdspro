# G1.2c.5 — Team invitation lifecycle and source ownership

Baseline BDSPro dev@bf4f24e733644523f9e1b597cfd3f1d2088760a2.
Gitea reference: main@edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76
- https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/services/org/team_invite.go
- https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/models/organization/team_invite.go

## Explicit differences from Gitea

In the reference, `services/org.CreateTeamInvite` calls a model
operation and then the mailer. BDSPro already has a *durable outbox*
created atomically with the invitation; replacing it with direct mail
would weaken crash recovery. Keep the outbox, hashed proof, deterministic
secret derivation for worker delivery, and existing Asynq delivery.

Like Gitea, `routers/api/v1/org` owns the HTTP protocol. The existing
`TeamHandler` persists because an injected store and signing key are
real per-server dependencies; removing the receiver would not
demonstrably improve dependency ownership. `TeamService` owns
cross-model action composition, not HTTP response creation or SQL.
`models/organization` keeps SQLC permission predicates, proof use,
and outbox persistence.

## Bug addressed

`RequireSession` validates the signed actor at HTTP ingress.
It does not establish a commit-time guarantee if the session gets
revoked after admission. Previously invite creation, acceptance and
revocation each wrote through model functions without locking the
acting session in those transactions. Thus session authority and
resource ownership were independently checked, without a common
commit boundary.

`TeamService` now acquires the actor session row `FOR UPDATE` through
`models/auth.RequireActingSession`, then performs each operation in
the **same transaction**. The model supplies transaction-bound
`CreateTeamInviteTx` and `AcceptTeamInviteTx` variants, while
retaining standalone APIs for pre-existing model callers. A minimal
`TeamInviteRevoker` query interface accepts both a store and a
transaction-bound `db.Queries`; no wrapper transaction is introduced
for ordinary standalone revocation.

## Invariants to verify

1. Revocation holds the actor session before issue/accept/revoke starts:
   invitation state, membership and outbox must remain unchanged.
2. Issue/accept/revoke holds session and is blocked on resource row:
   session revocation must wait; the action commits first.
3. Invalid/mismatched session denied. Wrong recipient, replay and
   owner-denied cases still rejected by SQLC. Malformed proofs do not
   acquire a transaction.
4. Outbox contains only invitation ID, never raw bearer token/email.
5. Existing HTTP contracts and Gin routes unchanged. Model tests and
   request-level E2E must still pass.

## Tradeoffs and non-goals

Issue and accept retain their existing transaction, adding one
session-lock query. Revoke now adds an explicit transaction and
session-lock query. Do not claim better p95/p99 without measurements.
Session expiration is evaluated when the lock is obtained, not
continuously through commit. Organization membership and invitation
proof checks remain database-owned; HTTP claims alone grant neither.

This PR does **not** redesign `TeamHandler`, rename `User` to `Person`,
change schema or migrations, or add a new generic transaction engine.
Those need separate callsite and domain-model reviews. The named
`TeamMemberWriter` replaces an anonymous exported parameter type
to make existing SQLC model dependencies explicit.

PostgreSQL row lock behavior:
https://www.postgresql.org/docs/18/explicit-locking.html
