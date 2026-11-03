# CODE-03 — Transaction ownership reference

## Scope

This change demonstrates the coding convention for a real multi-step operation.
It changes only **Create Organization**; domain naming remains the vocabulary
of the current database. URL paths, SQL, migrations, authentication, public
response/error contract and OTP transaction rules are unchanged.

## Source evidence (before / after)

Before:

```go
// OrganizationService.Create validated name, resolved actor PersonID, then:
return s.store.CreateOrganizationTx(ctx, db.CreateOrganizationTxParams{
    Name: name, OwnerPersonID: personID,
})
// internal/database/tx_create_organization.go owned all three SQLC statements.
```

After:

```go
// OrganizationService.Create owns the ordered business steps.
err = s.store.ExecTx(ctx, func(q *db.Queries) error {
    partyID, err := q.CreateParty(ctx)
    // stop on error
    organization, err = q.CreateOrganization(ctx, db.CreateOrganizationParams{
        PartyID: partyID, Name: name,
    })
    // stop on error
    _, err = q.CreateOrganizationMember(ctx, db.CreateOrganizationMemberParams{
        OrganizationID: partyID, PersonID: personID, Role: "owner",
    })
    return err
})
```

The after snippet is abbreviated for explanation; the executable code is in
`internal/service/organization.go`.

## Contract by ownership

| Owner | Responsibility |
| --- | --- |
| HTTP handler | Bind request, resolve authenticated account, write HTTP response |
| Service | Validate a domain name defensively, resolve account to Person, order the three writes, select the atomic boundary |
| Database `ExecTx` | Begin/rollback/commit; supply SQLC `Queries` bound to one transaction |
| `db/query/*.sql` / SQLC | Execute the authored SQL, subject to PostgreSQL FK/unique constraints |
| PostgreSQL | Enforce referential integrity and atomic visibility |

No new `TransactionManager`, `UnitOfWork`, repository abstraction or interface
was added. The old `CreateOrganizationTx` was removed, preventing two
independent production implementations of the same workflow.

## Alternatives and tradeoffs

- **Keep the old database-owned business workflow:** fewer lines in Service,
  but a developer has to open a second package to discover the atomic business
  sequence and make feature changes. Rejected for this scoped operation.
- **Generic transaction framework:** provides no additional guarantee over the
  existing `ExecTx` callback and adds maintenance. Rejected.
- **Move every transaction into Service:** rejected. OTP wrong-code retries
  must commit the failed-attempt counter even when the operation returns a
  business failure; a default rollback-on-callback-error contract is not
  universally applicable.
- **Move actor lookup into the transaction:** not done to preserve request
  behavior and query timing in this PR. The owner FK is enforced at write time;
  future changes involving stricter account lifecycle coupling need a separate
  transaction/race review.

## Reliability properties to verify

- A valid account creates Party, Organization, and exactly one `owner` member.
- A missing Account or blank name opens no creation transaction.
- Error from a step prevents later steps and propagates to the caller.
- An invalid owner foreign key aborts the transaction; no Organization is
  committed. The database integration test also verifies no extra Party
  remains after a failed third insert.
- Existing access-control SQL still distinguishes active owners and members.
- Cancellation and errors continue through `context.Context` and wrapping.
- No additional HTTP/Redis network hop is introduced by this refactor; the
  count of SQLC writes remains three (plus the existing account lookup).

## Verification / review

```sh
go test ./internal/service ./internal/database ./internal/api/handler
go test -race ./...
go vet ./...
go build ./...
make swagger
```

Actual acceptance requires the GitHub Actions result for the **final PR SHA**
including the disposable PostgreSQL integration database, SQLC drift,
migrations, formatting and HTTP runtime smoke test.

This PR is deliberately separate from CODE-01 field validation and CODE-02
constraint-name classification. Neither URL vocabulary nor database table names
are changed by this technical ownership correction.
