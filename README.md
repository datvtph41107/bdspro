# BDSPro Infrastructure Foundation

BDSPro currently contains only the runtime infrastructure. Legacy domain models, services, Auth/OAuth and Organization HTTP endpoints have been deliberately removed.

## Source organization

- `cmd/` — CLI entry points for `web` and `migrate`.
- `modules/` — reusable technical subsystems (settings, logging, routing, graceful lifecycle, PostgreSQL/Redis connections, queue infrastructure).
- `models/db/` — context-bound SQLC execution, PostgreSQL engine, transactions and generated SQLC bindings.
- `db/migration/` and `db/query/` — retained SQL schema history and SQLC query definitions. This reset does not delete or rewrite database data.
- `services/context/` — HTTP request context, typed action resolution and responses.
- `routers/init.go` and `routers/api/` — HTTP composition and the minimal infrastructure API.
- `templates/swagger/` — Swagger UI and infrastructure-only OpenAPI contract.

## Run

Copy `custom/conf/app.example.ini` to `custom/conf/app.ini` and adjust PostgreSQL and Redis for your environment.

```sh
make server
```

The HTTP infrastructure probe is `GET /api/v1/health`. Its response is `{"status":"ok"}`.

Database migrations use the retained SQL files:

```sh
make migrateup
make sqlc-check
```

## Verification

```sh
go test -race ./...
go vet ./...
go build ./...
make swagger-check
```

**Scope:** This is an infrastructure source reset, not a database reset. Business APIs, authentication and workers are not installed. They must be reintroduced explicitly against the canonical infrastructure contracts rather than through compatibility adapters.
