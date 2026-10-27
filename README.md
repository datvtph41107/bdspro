# BDSPro

BDSPro is a Go REST API using Gin, PostgreSQL/sqlc and Redis/Asynq. This source follows the direct startup approach of TechSchool SimpleBank, without starting gRPC or API Gateway.

## Read the program from the first line

Open **[main.go](main.go)**. The execution order is visible directly:

`main → run → LoadConfig → PostgreSQL → migrations → SQLC Store → Redis → worker → HTTP server`.

To trace a request, follow `routes/auth.go` or `routes/api.go` into the controller, then `app/services/` or `auth/` and `db/query/` (generated in `db/sqlc/`).

```sh
go run .
make test
docker compose up -d --build
```

The config loader reads `app.env` with process-environment overrides. PostgreSQL migrations live in `db/migration/`. Do not rewrite applied migrations or hand-edit SQLC-generated files.

The repository deliberately has no `bootstrap/`, `cmd/`, `scripts/`, or standalone `*_test.go` files for this code-reading phase. `make test` remains a Go compilation/package check until tests are introduced again.
