# CODE-02 — PostgreSQL constraint identity at the operation boundary

## Observation

Previously `OrganizationService.AddMember` treated every PostgreSQL SQLSTATE
`23505` as an existing active membership, and every `23503` as a missing
Person. SQLSTATE describes a class of constraint violation, **not which
constraint**. This was a false business classification risk.

## Change

The existing service translates only these precise database identities:

| SQLSTATE | Constraint | Business failure |
| --- | --- | --- |
| 23505 | organization_members_one_open_uq | MembershipExists |
| 23503 | organization_members_person_id_fkey | PersonUnavailable |

All other errors (including other unique and foreign-key constraints) retain
their PostgreSQL cause and are not converted into a public business identity.

No new exception framework, SQL query, migration, or HTTP response registry is
created. `apperrors.Wrap` preserves the business sentinel **and** the original
database error for `errors.Is` / `errors.As`.

This is a scoped correction to an existing contract, not approval of the
global `internal/errors` package. Its usefulness and alternatives remain an
independent review topic. It also deliberately does not change how a missing
organization and an unauthorized actor are distinguished by `pgx.ErrNoRows`.
That requires a separate authorization/non-disclosure decision.

## Acceptance

- Unit tests must demonstrate correct and incorrect constraint identities.
- Disposable PostgreSQL tests must confirm emitted constraint names match
  names introduced by migration 000011 and the preceding DDL.
- Existing public 409/404 errors must remain unchanged for the correct
  constraints.
- Unknown database errors must not be disguised as a public business error.
- CI SQLC, migrations, race, vet, build and HTTP smoke gates must pass.
