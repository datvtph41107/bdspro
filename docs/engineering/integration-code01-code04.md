# CODE-01–04 integration acceptance

This branch integrates four independent draft review slices **without merging
any of them into dev**. It exists to test whether their combined source builds,
runs and preserves data/HTTP contracts.

| Scope | Reference |
| --- | --- |
| Request validation + public HTTP errors | CODE-01 (#85) |
| Exact PostgreSQL constraint error identity | CODE-02 (#86) |
| Service-owned Create Organization transaction | CODE-03 (#87) |
| Minimal Makefile with original swag/statik command | revised CODE-04 (#88) |

Three overlapping source files were reconciled rather than overwritten:

- `handler/organization_test.go`: keep the real SQLC transaction fake from
  CODE-03 **and** the new HTTP validation expectations from CODE-01.
- `service/organization.go`: keep the service-owned ExecTx workflow from
  CODE-03 **and** exact-code-and-constraint handling from CODE-02.
- `database/organization_authorization_test.go`: keep CODE-03's raw SQLC
  fixture and rollback test **and** CODE-02's live PostgreSQL constraint-name
  assertions.

There is no replacement router, new schema, URL rename or bulk refactor.
`CreateOrganizationTx` is removed, leaving the existing `ExecTx` as the
transaction mechanism. Original Makefile migration commands are preserved,
including their original default behavior; operators must use an isolated
database for destructive local migration steps.

## Swagger contract and explicit limitation

The currently served API reference is versioned OpenAPI 3.0
(`doc/swagger/openapi.json`). The UI is served using Go `embed` in
development. `make swagger` is restored exactly to the original
`swag init ...` and `statik ...` commands. These generate a **separate**
Swagger 2.0 snapshot and Statik bundle; the generated snapshot is not the
runtime source of truth and is not substituted for OpenAPI 3.0.

This review integration validates both generator invocation and runtime
OpenAPI-to-Gin route coverage. It **does not** assert that the old Swagger 2.0
snapshot has the complete endpoint/schema contract; the old generation source
currently lacks the required annotations. The two formats cannot be silently
considered identical. Promoting Swagger 2.0 or fully regenerating OpenAPI 3.0
is a separate contract decision, not a justification for changing the user's
Makefile again.

## Required evidence

Run `go test ./...`, `go test -race ./...`, `go vet ./...`,
`go build ./...`, the SQLC drift check, migrations, OpenAPI route coverage,
the original swag/statik generators, and HTTP runtime smoke on the **final
integration head**. Tests requiring a database must run against the isolated
`bdspro_test` fixture with `BDS_TEST_MODE=disposable`.

Passing CI on this integration branch demonstrates source-level compatibility
of the review slices. It does not grant permission to merge or claim full
production readiness.

## Integration failure found and corrected

The first combined CI run found a cross-package test race: the phone-signup
test counted **all** Parties before/after a failed registration, while the
service integration test independently created a Party on the same
`bdspro_test` database. Package test processes can run concurrently, so
a legitimate commit from another package caused a false rollback failure.

The HTTP/OTP production implementation was **not changed**. The phone test now
checks exactly the phone credential and AccountID whose uniqueness it is
testing. The database organization test likewise checks an operation-specific
unique name on rollback instead of global Party counts. CODE-03's real
PostgreSQL rollback test checks its own uniquely named Organization and owner
foreign-key failure. A global count assertion is not a valid proof when other
tests share the connection target concurrently.

These corrections prevent the observed test interference; they do **not**
establish comprehensive isolation of every integration test in the suite.
