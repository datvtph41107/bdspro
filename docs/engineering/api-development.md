# Developing BDSPro APIs — Current source convention

1. `main.go` owns config, database, Redis, worker and HTTP lifecycle.
2. `internal/api/server.go` composes dependencies and a single Gin engine.
3. `internal/api/routes.go` registers native Gin paths/middleware in order.
4. `internal/api/handler` owns binding, HTTP responses and representations.
5. `internal/service` implements business behavior without Gin.
6. `db/query` owns editable SQL; `internal/database` owns SQLC output and scoped transaction operations.
7. `internal/api/response` owns public error mapping.
8. `doc/swagger/openapi.json` owns the reviewed public HTTP specification.
   `make swagger` checks operation coverage against Gin's actual routes.

Do not recreate Gin middleware composition or a second route registry. Do not
force a service/repository for a one-query endpoint without business value.
Keep success responses native and stable; map known errors through one writer.
Transactions cover all database mutations protecting one invariant; external
SMTP/Redis side effects are not part of the database commit.

When adding an API, update its Gin route, input/output contract, service and
SQL only when each is needed. Add HTTP and database tests for failure paths,
security scope, idempotence, and concurrency. Update OpenAPI **in the same
change** and run its coverage test. Avoid modifying generated SQLC by hand.

Development checks: `make sqlc`, `make swagger`, `go test ./...`,
`go test -race ./...`, `go vet ./...`, `go build ./...`.
Database tests use only a disposable `bdspro_test` database and explicit
`BDS_TEST_MODE=disposable`. Never run them against main data.

For the current identity contract see `docs/architecture/authentication.md`.
For security and release readiness, rely on CI from the exact commit being
reviewed, not historic passing runs.
