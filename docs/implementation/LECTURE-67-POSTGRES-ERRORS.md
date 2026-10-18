# Lecture #67 — PostgreSQL/pgx Error Contracts

**Scope:** Engineering Core in Go, SQLC, PostgreSQL; no AWS, no network deployment and no schema migration. Implementation follows the real BDSPro errors, not a generic error-handling framework.

## Source of truth

- `pgconn.PgError.Code`: PostgreSQL SQLSTATE returned by the server.
- `pgconn.PgError.TableName` and `ConstraintName`: use for a *known* business conflict; SQLSTATE 23505 alone does not identify the relevant constraint.
- `errors.Is(err, pgx.ErrNoRows)`: a query returned zero matching rows. This is not a network failure; the service may translate it based on the query's purpose.
- `errors.Is(err, context.Canceled)` and `context.DeadlineExceeded`: context lifecycle, not automatically a PostgreSQL SQLSTATE.
- `db/sqlc.PostgresError(err)` only unwraps typed server errors with `errors.As`; it does not parse error strings or transform errors into domain errors.

## Observed cases and policy

| SQLSTATE / error | Meaning | Action |
|---|---|---|
| 23505 | unique_violation | Domain 409 **only** for recognized index/table, otherwise preserve error |
| 23503 | foreign_key_violation | Usually integrity failure; **not** automatically 409 |
| 23514 | check_violation | Preserve; e.g. Party C01 failed at COMMIT |
| 40001 | serialization_failure | Aborted transaction; retry only a verified idempotent **whole** unit |
| 40P01 | deadlock_detected | Aborted transaction; no automatic blind replay |
| 55P03 | lock_not_available | NOWAIT / lock policy; distinguish from 40P01 |
| 57014 | query_canceled (server-side) | Analyze statement timeout/explicit cancellation; do not guess client intent |
| pgx.ErrNoRows | no result for SELECT/RETURNING | Domain meaning depends on operation |
| context.Canceled / DeadlineExceeded | client/request/timeout termination | Not proof of rollback/commit outcome |
| connection/commit failure | transport failure / ambiguous COMMIT | Do not assert COMMIT or ROLLBACK without durable evidence |

## Changes

1. `organization.ApproveRegistration`: only unique violation on `organization_registry_identifiers` maps to `ErrConflict`. Other FK/unique errors pass through.
2. `organization.ApproveInitialClaim`: missing active organization or account may be a known conflict; every other query failure retains the original error. This fixes backend errors incorrectly returning HTTP 409.
3. `organization.AcceptInvitation`: map only known duplicate open membership/invitation constraints, rather than every 23505.
4. `auth.Register`: retain exact login email unique-index contract using `dbsqlc.PostgresError` for consistency.
5. `db/sqlc.PostgresError`: minimal typed extraction; unit and real PostgreSQL tests verify nested wrapping and SQLSTATE.

## Negative constraints

No retry framework, no new Repository/UnitOfWork abstraction, no raw SQL edits, no changed HTTP status for known business outcomes, no new SQLC query or migration. `ExecTx` behavior remains unchanged: every transactional callback uses its `q`, errors remain inspectable, and COMMIT failures cannot be blindly retried.

## Acceptance

Run `go test ./organization ./db/sqlc ./...`, race tests, SQLC generated diff, migration UP-DOWN-UP, server and Compose smoke through the existing CI on an isolated test database. CI PASS is evidence for these cases only, not a production reliability certification.
