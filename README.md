# BDSPro Modular Monolith Foundation

BDSPro contains the HTTP/runtime infrastructure and an initial session-authenticated Organization/Team API vertical slice. Historical Auth/OAuth login routes and unrelated legacy domain endpoints are not reinstalled.

## Source organization

- `cmd/` — CLI entry points for `web` and `migrate`.
- `modules/` — reusable technical subsystems (settings, logging, routing, graceful lifecycle, PostgreSQL/Redis connections, queue infrastructure).
- `models/db/` — context-bound SQLC execution, PostgreSQL engine, transactions and generated SQLC bindings.
- `db/migration/` and `db/query/` — retained SQL schema history and SQLC query definitions. This reset does not delete or rewrite database data.
- `services/context/` — HTTP request context, JWT session authentication, verified Organization assignment and responses.
- `models/organization/` and `services/org/` — Team domain values and context-first SQLC transaction orchestration.
- `routers/init.go` and `routers/api/` — HTTP composition and the minimal infrastructure API.
- `templates/swagger/` — Swagger UI and infrastructure-only OpenAPI contract.

## Run

Copy `custom/conf/app.example.ini` to `custom/conf/app.ini` and adjust PostgreSQL and Redis for your environment.

```sh
make server
```

The HTTP probe is `GET /api/v1/health` with `{"status":"ok"}`. The Team endpoints are `GET/POST /api/v1/orgs/{org}/teams` and `GET/PATCH /api/v1/orgs/{org}/teams/{team}`; they require a **session-bound access JWT** issued by a trusted authentication flow. Read calls require active Organization membership; writes require owner membership and are authorized again inside PostgreSQL.

Database migrations use the retained SQL files:

```sh
make migrateup
make sqlc-check
```

## Verification

All tracked Go `*_test.go` files were removed by explicit request.
The commands below now check compilation, vet and runtime smoke, not
functional or regression correctness.

```sh
go build ./...
go vet ./...
make swagger-check
```

**Scope:** This is an infrastructure source reset, not a database reset. Team business APIs and their JWT/session authorization are installed. Full login/token issuance and unrelated domain APIs are not reinstalled. The repo has no Go regression tests after the explicit removal request; CI exercises an isolated Team lifecycle on PostgreSQL plus infrastructure smoke checks. See [Team API vertical slice](docs/architecture/team-api-vertical-slice.md).
