# Canonical migrations

Each table has its own ordered `db/migration/NNNNNN_table.up.sql` and
`NNNNNN_table.down.sql`, with associated indexes in the same file. There
are 26 ordered migrations, ending at `000026_oauth_refresh_tokens`.
The names and constraints are final at creation: no custom triggers,
backfill, temporary mirror, rename, or archived credential table.

`make migrateup` and `make migratedown1` use the database selected by
`app.ini`. A rollback drops the corresponding table and its data; command
rollback is restricted to development. It must only be used on a disposable
database.

**Breaking compatibility boundary:** the old ledger also used versions
1–21, but a version number from that ledger does not mean the same schema.
Before each migration, `models/db/migration.go` checks the canonical
version-to-table count and marker tables. Old databases are rejected, never
silently stamped as migrated. No existing database records are deleted
by this work: to change an actual deployment, provision a **new** database,
export and verify data/identity links, test the conversion and switch traffic
through a separate reviewed rollout. Do not remove `schema_migrations` or
force a migration version on an existing DB.

SQLC reads only `db/migration/` and `db/query/`. Regenerate `models/db/sqlc/`
and its facade aliases using `make sqlc`; do not hand edit generated
code. CI tests fresh PostgreSQL 18 creation, all 26 tables and indexes,
the absence of obsolete OAuth tables and SQL triggers, and refusal of an
older migration ledger.

```sh
make migrateup
make sqlc-check
make test
```
