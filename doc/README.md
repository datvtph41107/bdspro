# BDSPro local documentation (the `doc/` workflow)

This directory follows the **purpose** of TechSchool SimpleBank's `doc/`
without copying bank-specific schemas or making a second SQL source of truth.

| SimpleBank | BDSPro | Owner |
| --- | --- | --- |
| `doc/db.dbml` | `make db_docs` emits `doc/db.dbml` | Read-only introspection from migrated PostgreSQL |
| `doc/schema.sql` | `make db_schema` emits `doc/schema.sql` | `pg_dump --schema-only` from the same actual DB |
| `doc/swagger` | `api/openapi.json`, `/docs` on Gin | OpenAPI **3.1** for running Gin HTTP routes |
| `doc/statik` | `api/openapi.go` with Go `//go:embed` | Compile-time embedded HTTP spec, no statik generator needed |
| gRPC-Gateway OpenAPI v2 | `docs/grpc-swagger/bdspro.swagger.json` (when proto PR merges) | Generated RPC-only Swagger **2.0**; does not replace Gin OpenAPI |

BDSPro's initial conceptual DBML remains in `docs/database/final/`,
but migrations are the executable schema truth. **Do not** run
`dbml2sql` on the conceptual model and apply the result to the existing DB:
it cannot faithfully preserve triggers, partial indexes, functions or all
transaction invariants. Use the actual PostgreSQL schema dump instead.

## Local commands

First migrate your chosen disposable/non-production database, then:

```bash
export DB_DOCS_DATABASE_URL='postgresql://root:secret@localhost:5432/bdspro_test?sslmode=disable'
make db_docs
make db_schema
```

Generated `doc/db.dbml`, `doc/schema.sql` and `doc/constraints.md`
describe **the selected migrated database**; they are not author-edited
migrations. The existing CI already publishes the implemented schema as
build artifacts. The `doc/` local outputs should be reviewed before
committing since an un-migrated or different DB produces different results.

**Statik vs embed:** SimpleBank generated a statik package holding Swagger
assets. BDSPro already uses first-party Go embedding for its spec, so a
second `doc/statik` tree would duplicate compiled assets. The Swagger UI
JS/CSS currently use a version-pinned CDN, not offline embedded assets.
Vendor those assets explicitly before claiming an offline Swagger UI.
