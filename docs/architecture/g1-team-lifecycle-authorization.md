# G1.2c.6 — Team lifecycle: authorization and transaction ownership

Source baseline: BDSPro dev@359798303ff04b2e1c7332e12ef9f7cb89c7e2a9.
Gitea reference main@edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76:
- https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/services/org/team.go
- https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/routers/api/v1/org/team.go

## Inventory and problem

`models/organization.CreateTeam` and `RenameTeam` are called
directly by the Gin team router and by model/integration tests.
Before this checkpoint, SQLC guarded each INSERT/UPDATE with an
owner-scoped active `organization_members` predicate using `FOR SHARE
OF m`. This is necessary resource authorization; it does **not**
serialize a previously-admitted HTTP request with revocation of the
JWT's acting session.

## Why this service boundary has value

Gitea's `services/org/team.go` owns a team lifecycle that coordinates
model writes and other business effects. BDSPro's equivalent
`TeamService` now coordinates first-party session authority with
team create/rename in the same transaction. The same service already
owns membership mutations and invitation lifecycle.

`routers/api/v1/org.TeamHandler` owns request binding, signed actor
extraction and HTTP status. A receiver remains justified because
a server-injected store and invitation signing key are real immutable
dependencies. No extra handler package or middleware executor.

`models/organization` owns domain validation and SQLC permission/error
mapping. `TeamWriter` describes only two SQLC write operations,
available both on `db.Store` and transaction-scoped `*db.Queries`.
The original standalone model functions remain usable by domain tests
and trusted internal callers. The service adds commit-time authority
for the two first-party HTTP entry points; callers must not assume the
standalone model API provides the same session boundary.

`ValidTeamName` is the single name policy used for preflight before
opening a transaction and by standalone model calls. Duplicate checks
are cheap and intentional: model functions cannot rely on a service
having validated inputs when invoked independently.

## SQLC and lock order

Team service order:
1. Preflight name (no DB).
2. Start a transaction and lock acting session `FOR UPDATE`.
3. Execute existing owner-scoped `CreateTeamForOwner` or
   `RenameTeamForOwner`, which holds `FOR SHARE OF m` until commit.
4. Commit, then expose the created/renamed team to HTTP.

Revoke-one/revoke-all lock the account row before their session rows;
team lifecycle operations only lock their own acting session and
member rows, and do not mutate other sessions. This avoids adding an
account-wide writer lock to every team action. PostgreSQL lock and FK
behavior must still be evaluated for new domain workflows.

## Evidence required

Run disposable bdspro_test PostgreSQL with pinned connections and
`pg_blocking_pids`:
- revoke first: create and rename block on acting session then reject,
  with no persisted change.
- write first: create and rename hold acting session while waiting on
  an owner-membership lock; session revoke waits; write commits first.
- invalid and cross-account acting sessions rejected. Invalid name
  rejected without a DB. Existing role/duplicate-name constraints
  unchanged.

No migration, generated SQLC, Gin route, response, queue, or API-schema
changes. The CREATE gains a transaction and session-lock SQL statement;
RENAME gains the same. p95/p99 changes are **not** asserted without
contention benchmarks.

## Deliberate scope

Organization creation, organization rename, organization membership
addition and the related authorization edges still need a separate
lock-order review. This checkpoint closes only Team create and rename,
not all organization writes.

PostgreSQL 18 row-lock reference:
https://www.postgresql.org/docs/18/explicit-locking.html
