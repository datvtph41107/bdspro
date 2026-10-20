# Lecture #38 — DB documentation from implemented PostgreSQL

**Purpose:** Document the schema actually applied by migrations, not the
larger business-design candidate.

The SimpleBank lesson generates DB documentation and SQL dump from DBML.
For BDSPro we intentionally **reverse the authority**:

```text
db/migration/*.up.sql (canonical executable history)
 -> migrated PostgreSQL 18 database
 -> cmd/dbdocs (read-only catalog projection)
 -> implemented.dbml and constraints.md (derived)
 -> pg_dump --schema-only (exact SQL schema snapshot)
```

The existing `docs/database/final/BDSPro-CANONICAL-DATABASE.dbml` is
an explicitly labeled **164-table design candidate**, NOT an applied schema.
Never use that design file to initialize the database or overwrite migrations.

## Local use

With a disposable PostgreSQL 18 database whose migrations are already up:

```bash
DB_DOCS_DATABASE_URL="$CORE_TEST_DATABASE_URL" \
  go run ./cmd/dbdocs -out /tmp/bdspro-dbdocs
pg_dump --schema-only --no-owner --no-privileges \
  --schema=public --dbname="$CORE_TEST_DATABASE_URL" \
  > /tmp/bdspro-dbdocs/schema.sql
```

Use **PostgreSQL 18 pg_dump**; an older client may reject PG18 server.
Explicitly choose the database. Do not run schema export on an
unreviewed production endpoint.

## CI and artifacts

GitHub Actions generates all three outputs from `bdspro_core_ci`
**after migrations**, using PostgreSQL 18 `pg_dump` from the
`postgres:18-alpine` image, and uploads one downloadable artifact:

- `implemented.dbml` — actual tables, columns, PK and composite/single FKs.
- `constraints.md` — actual constraint definitions, indexes (including
  expression/partial predicates), triggers and functions.
- `schema.sql` — PostgreSQL schema-only DDL snapshot.

The diagram is a **read-only approximation**. Partial indexes, check
constraints and trigger semantics cannot be reconstructed merely from
the diagram; review `constraints.md`, migrations and schema dump.
Generated snapshots do not enter the source-of-truth migration history.

This lecture does not introduce AWS, Kubernetes, gRPC, new DB migration,
new Go dependencies or a DBML-first schema workflow.
