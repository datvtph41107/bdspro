# BDSPro

Go REST monolith: Gin, PostgreSQL/SQLC, Redis/Asynq. There is one executable (`main.go`), no gRPC/Gateway, and no custom routing framework.

## Read the program from main.go

`main → run → config → PostgreSQL/migrations → SQLC Store → Redis/Asynq → HTTP server`.

Application Go packages live under `internal/`:

```text
main.go
internal/
  config/                  # process and JWT configuration
  core/router/             # Gin engine, startup wiring and HTTP lifecycle
  core/middleware/         # authentication and request admission
  core/errors/             # public application errors
  core/response/           # HTTP response mapping
  routes/api.go            # one authoritative endpoint inventory
  api/handler/             # HTTP request binding and responses
  api/handler/request/     # typed inputs
  api/handler/resource/    # public output projections
  api/service/             # auth/session and organization operations
  db/sqlc/                # SQLC generated queries and handwritten Store/Tx
  token/                  # JWT signing and verification
  worker/ jobs/ mail/     # asynchronous email verification
  util/                   # passwords, logging, randomness
db/
  migration/               # append-only schema history
  query/                   # canonical editable SQL
```

`internal/` is Go's import boundary; it does **not** grant or deny resource access. There is no `pkg/` because this app currently exposes no reusable public library.

## Work on an API

Follow `internal/routes/api.go → internal/api/handler → internal/api/service → db/query/*.sql → internal/db/sqlc`. JWT middleware establishes the verified AccountID; resource authorization is checked in the resource query/transaction. Do not change routes, statuses, rate limit semantics or database invariants as part of a package relocation. See [current architecture](docs/architecture/README.md) and [developer workflow](docs/architecture/workflow.md).

```sh
go run .
make sqlc
make test
go vet ./...
go build ./...
docker compose config --quiet
```

`make test` runs focused Go unit tests. PostgreSQL authorization integration requires `BDS_TEST_MODE=disposable` and `BDS_TEST_DB_URL` pointing to an isolated `bdspro_test` database with migrations applied; CI provisions that database. Never run write tests on production or existing local application databases.

The current architecture documents are authoritative; older `docs/engineering/` notes and the SimpleBank source map are historical records, and may describe removed packages. A green CI run is not proof of Google/SMS provider E2E or production readiness.
