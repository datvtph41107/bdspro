# G1.2c.9a — Account deletion must not orphan an active owner

BDSPro baseline: dev@bd7d9f72a8c7f4be083d45429deb81104b510739
Gitea comparison:
https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/services/user/delete.go

## Evidence and model distinction

Gitea implements user deletion in the services/user domain and cleans
cross-domain dependencies before removing the User. BDSPro has no
first-party DeleteAccount or account-deletion HTTP endpoint at this
baseline. It does have an Account-to-Person relationship, while
organization_members reference only Person. Consequently, ordinary
account deletion can leave an owner membership with no Account. A
future deletion service alone does not protect migration/raw SQL.

## Migration 31: relational integrity, not a trigger

Add nullable `organization_members.owner_person_id`. Backfill it for
all live owners, then apply two CHECK clauses:

- `owner_person_id IS NOT NULL` exactly when role is `owner` and
  `ended_at IS NULL`.
- When non-null, `owner_person_id = person_id`.

The owner person column is a foreign key to the **unique**
`accounts.person_id` with ON DELETE RESTRICT. Thus a raw delete of
an active owner's Account fails with SQLSTATE 23503 instead of
leaving an owner without a live Account. If an existing owner
has no Account at migration, the migration fails rather than
fabricating data. Ordinary `Person` membership without Account stays
valid and does not prevent account deletion.

The canonical SQLC create, promote, and demote queries set/clear
owner_person_id in the same SQL mutation; no second statement, service
layer, trigger, or new table. A partial child FK index helps account
deletion checks when organizations grow.

## Guarantees and exclusions

Protected:
- active owner requires Account of identical Person;
- Account DELETE is rejected while that owner remains active;
- invalid owner creation or raw role change without the owner FK fails;
- serial owner promotion/transfer keeps link consistent within commits.

Not protected:
- the count of owners against arbitrary raw SQL demotion
  (the app service serializes and checks, but the database cannot
  express a cross-row count with a CHECK);
- account suspension/credential removal or loss of verified channels;
- direct writes to privileged SQLC from newly introduced code without
  architecture ownership review;
- performance p95/p99 until measured.

Do not add a public DeleteAccount endpoint under this checkpoint.
An eventual account-deletion lifecycle must handle OAuth grants,
sessions, audit retention, team invitations and organization ownership
in one explicit operation; a bare SQL DELETE is not a business flow.

## Acceptance

Disposable PostgreSQL, SQLC parity, canonical migration v31:
- delete active owner's Account fails 23503 with owner-account FK;
- attempt to construct an owner for an orphan Person fails CHECK/FK;
- mismatched owner_person_id is rejected;
- ordinary orphan Person may still join as member;
- transfer and demotion release/acquire FK in same commit;
- delete-vs-promote race serializes and cannot orphan an owner;
- existing suite (including G1.2c.8b owner/team deadlock tests)
  passes unchanged; race, vet, route inventory, Docker and smoke green.

Costs: one partial child index and a nullable FK-backed column
on organization_members; owner creation and role changes incur FK
checks/locks. Measure contention before making throughput claims.
