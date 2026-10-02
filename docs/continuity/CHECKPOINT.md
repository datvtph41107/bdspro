# Current Checkpoint

Updated: 2026-10-02

## Source identity

Active repository:

`datvtph41107/bdspro`

Active development branch:

`dev`

Stable/default branch:

`main`

Preferred local workspace:

`~/projects/bdspro`

Operational Go module identity:

`github.com/datvtph41107/bdspro`

## Verified Git reality

At the branch transition on 2026-10-02:

```text
main:
  1a49d5bc469110c3b4b8d42a0d5f419bc45857bc
  docs: checkpoint staged recovery implementation

dev:
  created from
  5fe5cbda5d4be5e54e9b21323efc01d04f9e0e7f
  ft: sqlc sql base

relationship before this continuity update:
  dev is 2 commits ahead of main
  dev is 0 commits behind main
```

The former branch `recovery/local-implementation` was a temporary recovery
vehicle. `dev` is now the active development branch. Delete the old branch only
after local `dev`, `origin/dev` and the expected HEAD are verified.

Live Git/source outranks older prose. Do not reset, clean, rebase, overwrite or
blindly pull merely to make the workspace match this document.

## Current phase

```text
PostgreSQL / explicit SQL / sqlc / pgx persistence foundation
STATUS: ACTIVE
```

The canonical persistence direction is:

```text
business fact
 -> relational design
 -> durable database invariant
 -> versioned SQL migration
 -> explicit SQL query
 -> sqlc generated persistence code
 -> pgx/v5 / pgxpool
 -> PostgreSQL / future PostGIS
```

GORM and the old BDSPro persistence model are historical evidence only, not the
canonical implementation path.

## Verified live source on dev

The current live branch contains:

```text
Makefile
migrations/000001 ... 000005
sqlc.yaml
db/query/listing.sql
db/sqlc/db.go
db/sqlc/listing.sql.go
db/sqlc/models.go
db/sqlc/querier.go
go.mod
go.sum
```

Verified details:

- `go.mod` requires `github.com/jackc/pgx/v5 v5.11.0`;
- `sqlc.yaml` targets PostgreSQL + `pgx/v5`, reads schema from
  `migrations`, queries from `db/query`, generates to `db/sqlc`, and emits
  an interface;
- `db/query/listing.sql` defines `CreateListing :one`;
- generated sqlc code exists and exposes a `DBTX` contract using
  `pgconn.CommandTag`, `pgx.Rows`, `pgx.Row` and `pgx.Tx`;
- generated Listing persistence types map nullable `description` and `price`
  through `pgtype.Text` and `pgtype.Int8`;
- migration `000005` adds nullable `price BIGINT` with a non-negative CHECK;
- commit `5fe5cbd` removed `compose.yaml`, `listing/integration_test.go`
  and `test-integration.sh`.

Do not recreate deleted files automatically. Their absence is current source
reality; any replacement must be earned again from current pressure.

## Reconciled contradiction

Older continuity prose said sqlc implementation and pgx installation had not
been verified. That is now stale.

Live `dev` source proves:

```text
pgx dependency exists
sqlc.yaml exists
query source exists
generated sqlc code exists
```

Those files alone do NOT prove:

```text
current PostgreSQL runtime state
current migration version in the live local database
that sqlc generate is reproducible with zero diff now
that pgxpool is wired into the current application runtime path
that CreateListing has executed successfully through sqlc + pgxpool
```

Historical local work reports migration 000005 was applied successfully, but a
fresh recovery must inspect the actual database before treating that runtime
state as current truth.

## Persistence learning gate completed

The current learning work established the core model for:

```text
query generator vs runtime driver
database/sql vs PostgreSQL driver
lib/pq historical role
native pgx vs pgx/stdlib
pgx / pgxpool / pgconn / pgtype / stdlib
connection vs pool
Query / QueryRow / Exec
ErrNoRows vs PostgreSQL PgError
PostgreSQL type/nullability mapping
transaction connection pinning
extended vs simple query protocol
Parse / Bind / Execute / Sync
connection-scoped prepared statements
pgx statement cache
lazy pool construction vs connectivity proof
Acquire / Release / pool pressure
connection lifetime / idle retirement
pool metrics and shutdown lifecycle
```

Architectural consequence:

```text
PostgreSQL-specific knowledge is intentional inside Postgres infrastructure.

sqlc/pgx/pgtype/pgconn representations must not accidentally become
transport or domain representations.

Application use cases own business transaction boundaries.
```

## Current unresolved pressure

The source has crossed from conceptual selection into generated persistence code,
but the runtime proof chain has not caught up.

Current questions:

```text
Can the current sqlc configuration regenerate deterministically?
Does generated code match the current migration/query contract?
Where should the shared pgxpool be constructed and owned?
How does the first CreateListing call flow through the application without
leaking pgx/sqlc types across the intended persistence boundary?
What runtime evidence proves the connection/query path works?
```

## Next smallest justified actions

First finish the branch transition locally:

```text
move local work from recovery/local-implementation to dev
 -> set upstream to origin/dev
 -> verify working tree and HEAD
 -> remove the obsolete local recovery branch name if it still exists
 -> delete remote recovery/local-implementation only after dev is verified
```

Then continue persistence work from live source:

```text
inspect current query/generated files
 -> run sqlc generate and inspect whether it creates a diff
 -> inspect current application construction point
 -> establish the smallest shared pgxpool ownership point
 -> execute the first sqlc query
 -> observe real result/error
 -> only then add the next boundary
```

Do not jump ahead to repository/service/domain scaffolding, Docker Compose, CI,
Testcontainers or microservice extraction without new demonstrated pressure.

## Working mindset

Use:

```text
CURRENT REALITY
 -> INTENT
 -> REQUIREMENT
 -> PRESSURE
 -> INSUFFICIENCY
 -> SMALLEST RESPONSIBLE RESPONSE
 -> EARNED BOUNDARY
 -> MECHANISM
 -> EVIDENCE
 -> TRADE-OFF
 -> DECISION
```

The user types and implements the code/commands. For unfamiliar mechanisms,
trace primary documentation/source, predict behavior, run the real operation and
learn from observed state rather than receiving a completed implementation by
default.
