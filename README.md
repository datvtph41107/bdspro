# BDSPro

Go/Gin backend using PostgreSQL, sqlc, Redis/Asynq and JWT sessions. The active entrypoint is `cmd/bdspro/main.go`; `bootstrap/` loads `app.env`, composes dependencies and starts Gin HTTP plus the verification worker.

## Source navigation

```text
cmd/bdspro/main.go       Process entry and termination
bootstrap/               Config loading, PostgreSQL migration, Redis, worker and HTTP startup
routes/                  Canonical Gin route definitions
app/http/controllers/    HTTP handlers by business capability
app/http/requests/       Typed request contract
app/http/binding/        Shared JSON/query binding and validation
app/http/middleware/     Authenticated identity and request policy
app/http/resources/      Public responses
app/http/responses/      HTTP response writer
app/errors/              Safe application-to-HTTP error classification
app/services/            Business service operations
auth/                    Auth credentials and session business rules
db/query/                SQL source of truth
db/sqlc/                 SQLC-generated queries, Store and PostgreSQL transactions
db/migration/            Append-only schema and constraints
worker/ jobs/            Queue runtime and email verification job
token/ mail/ util/       Tokens, SMTP delivery and runtime utilities
```

Follow an API as `routes/` → `app/http/controllers/` → `app/http/requests/` → `app/services/` or `auth/` → `db/query/` and `db/sqlc/`.

Run with `docker compose up -d --build` or `go run ./cmd/bdspro` when PostgreSQL/Redis are available. Build: `go build ./cmd/bdspro`. Tests: `make test`. Keep applied migrations immutable and regenerate sqlc after SQL changes.

The retired protobuf/gRPC/Gateway and Swagger-static transport are **not** alternate runtime paths. `dev` is the canonical integration branch after CI acceptance.

See [Architecture](docs/architecture/README.md), [workflow](docs/architecture/workflow.md) and [implementation status](docs/architecture/status.md). Old `docs/engineering` chapters remain historical evidence, not competing code owners.
