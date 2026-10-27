# DATA-01 — SQL, Store, transaction, repository ownership

## One editable SQL source

**Rule:** application SQL statements live under `db/query/*.sql` and are compiled by `sqlc generate` into `db/sqlc/*.sql.go`. Services must never embed routine `SELECT`, `INSERT`, `UPDATE` or `DELETE` strings. Generated code is not edited manually. Schema evolves only through new migrations; applied migration history is immutable.

Both raw `pool.QueryRow(...,sql)` and SQLC execute SQL, but allowing both as equal default patterns produces two sources of truth. **Raw Go SQL requires an explicit, source-backed exception** (unsupported SQLC capability, dynamic SQL impossible to express reasonably with typed parameters, COPY/streaming-specific API). Even then, its executor stays inside the database owner, not inside controller or service.

**Migration in this branch:** Google Identity, Phone OTP and verification-reuse SQL statements have moved to `db/query/*.sql`; generated queries were produced with sqlc v1.30.0. The former standalone `phoneOTPStore` pool owner has been removed. The OTP consumption transaction stays at the database owner while its behavior and concurrency invariants are verified. New raw SQL still requires review.

## Store and repository

The `pgxpool.Pool` is constructed once in the application startup and is owned by the database execution layer. SQLC uses pgx internally. A separate `phoneOTPStore{pool *pgxpool.Pool}` is a contradictory execution owner and should be folded into the canonical store.

Do not build `OrganizationRepository -> BaseRepository -> SQLStore -> SQLC` if these layers only forward calls. Add a consumer-owned interface only when it has a real substitution/dependency-test value, not as a naming convention.

The current `Store interface { Querier; Tx... }` is broad; monitor its coupling as query count increases. Separate generated SQLC from handwritten transaction mechanics only when that split simplifies imports/tests and preserves cross-operation transactions.

## Transaction ownership

Service chooses what the workflow does and the atomic boundary. The database owner implements `Begin/Commit/Rollback` and binds SQLC queries to the same transaction. The long-term goal is to read the entire Create Organization workflow in one service method instead of hiding operations inside `CreateOrganizationTx`.

**Do not change semantics during file relocation.** OTP wrong-code handling commits an incremented failed-attempt counter before returning the business error; a generic `WithinTx` that rolls back whenever the callback returns an error would break brute-force controls. OTP bind, one-time consumption and concurrent locking require PostgreSQL transactions and constraints.

Resource authorization for Organization member/update is part of the SQL predicate (`WHERE EXISTS (active owner)`). Moving that authority into a preflight controller check can introduce time-of-check/time-of-use races.

PostgreSQL transaction cannot atomically commit Redis/Asynq enqueue. The current registration callback schedules verification before PG commit; retry delay does not guarantee consistency. Address delivery semantics in a separate business decision, not as a silent refactor.

## SQLC scale

Generated file count is not itself a design problem. Avoid a God Store *interface* for every consumer; prefer narrow interfaces when warranted. Multiple `sqlc.yaml` output packages can be considered later but may duplicate models and complicate transactions. No generalized reflection ORM or dynamic schema facade.
