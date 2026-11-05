# Canonical database migrations

BDSPro source of truth is `db/migration/000001_core.up.sql`: one fresh-install
baseline defining the final table and index names, including the **OAuth2
authorization server** and the separate external OIDC credential mapping.
No transitional mirroring, rename or backfill SQL, and no custom triggers.

`make migrateup` uses the configured database; `make sqlc` regenerates
SQLC from `db/migration/` and `db/query/`. The `migrate --down`
command only runs in development and destroys all application data.

**Breaking migration version transition:** historical databases that used
migration versions 1–21 are NOT compatible with this rewritten ledger.
The CLI explicitly rejects version >1, and rejects version 1 if the canonical
schema marker `oauth_clients` is absent. Do not force a version number, delete
`schema_migrations`, drop old tables, or deploy the new binary against that
database. Export and verify a data migration into a separately provisioned
database, including accounts, credentials, sessions and OAuth binding ownership,
before switching traffic. Backups and restoration are required before cutover.

CI provisions `bdspro_test` from empty, validates table counts, exact current
names, absence of triggers, and verifies that a simulated v21 database is
rejected. Tests that modified legacy schema migration versions were removed;
this baseline change does NOT prove upgrade compatibility for a live database.

Run:
```sh
make config-local
make migrateup
make sqlc-check
make test
```
