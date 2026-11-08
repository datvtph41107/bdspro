# G1.2c.7 — Organization lifecycle: actor authority and lock order

Source baseline: BDSPro dev@1a84b4de6a17349a8fd04ca5618604c9b042b701.
Gitea reference: edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76
- https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/models/organization/org.go
- https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/services/org/org.go
- https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/routers/api/v1/org/org.go

## Evidence and choices

The existing model had three organization mutations:
- `CreateOrganization` already wrote party, organization and initial owner
  atomically, guarding the account with `LockOrganizationCreator FOR KEY SHARE`.
  HTTP admission did not hold actor session through commit.
- `UpdateOrganization` used a SQLC owner-guarded UPDATE with
  `FOR SHARE OF m`, but no atomic session check.
- `AddOrgUser` used an owner-guarded INSERT with `FOR SHARE OF owner`,
  but no atomic session check.

Do not move SQLC or domain policy to router. Gitea puts rich organization
lifecycle in services where needed, but models own persisted organization
state. BDSPro retains GIN, SQLC, explicit db.Store injection and its
existing owner-predicate SQLC. A stateless service package function is
sufficient; no new `OrganizationService` object or middleware.

## Account-first creation — important exception

Credential reset calls `db.LockAccountForCredentialRecovery` (`FOR UPDATE`)
before `DeleteSessionsForAccount`. If organization creation locked
acting session FIRST and then the creator account, while recovery
locked the account FIRST and then the session rows, they could deadlock.

Therefore, creation always obtains `FOR KEY SHARE` on account first
(via model `LockCreator`), then `FOR UPDATE` on acting session, then
creates Party, Organization, and initial owner membership. This
serializes account deletion and password reset consistently with their
existing account-first paths. The opaque `LockedCreator` has an
unexported person ID, making it impossible for other packages to forge
a positive authorized creator value. The token is only valid in the
same transaction that acquired the account lock; callers MUST NOT
retain it across transactions.

Model `CreateOrganization` retains its original standalone transaction
API and uses the same helper. This avoids a duplicated creator-lock
SQL query and avoids nested transactions.

## Update/add-member operations

A transaction locks only the acting session, then invokes the existing
SQLC UPDATE/INSERT. Both SQLC operations still lock owner membership
`FOR SHARE`; adding an account-level lock without evidence would
increase contention and isn't required for these paths. The existing
member-revocation concurrency tests are retained. The narrow
`OrganizationWriter` interface allows `*db.Queries` and the store
without duplicating SQL.

## Testing and cost

Disposable PostgreSQL with pinned connections must prove:
1. Session revoke wins for all 3 writes: no mutation commits.
2. Write wins for all 3: session revoke waits for action commit.
   Creation is staged on the creator Person FK row after acquiring its
   account and session locks; update/add-member are staged on owner
   organization_members row.
3. Password reset/account FOR UPDATE wins: creation must block
   *before* acquiring the session lock; subsequent removal of session
   denies creation with no orphaned Party/Organization.
4. Invalid names fail without DB; deleted actor, revoked session,
   foreign actor/session and SQLC owner denial remain distinct.
5. Existing HTTP acceptance and model-level tests remain green.

Three HTTP paths add one session-lock SELECT. Create retains its
transaction; Update and AddMember gain transactions. No p95/p99
claims without load and lock-wait measurement.

No schema changes, generated SQL, new route, generic transaction
abstraction, global database or new middleware.

PostgreSQL 18 row-lock guide:
https://www.postgresql.org/docs/18/explicit-locking.html
