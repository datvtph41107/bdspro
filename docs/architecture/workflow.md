# Developer contract — adding an HTTP API

The source is organized by **technical responsibility**, not forced
business-module packages:

`routes/api.go` → `api/handler` → `api/service` → `db/sqlc`.

1. Read the current endpoint inventory and preserve existing methods,
   paths, policies and JSON unless the public contract change is approved.
2. Declare the new endpoint **once** in `routes/api.go` under the
   appropriate public/authenticated group. The application prefix comes
   from `core/router/server.go`. Do not start another Gin engine.
3. Place body/query/path input types in `api/handler/request/`.
   Use native Gin binding once; malformed or oversized input must not
   reach the business operation.
4. Implement a discoverable `api/handler` method. Read verified actor
   identity from middleware; choose HTTP success status and output resource.
   Use `core/response.Send` for ordinary JSON responses.
5. Put business validation, orchestration, expected SQL errors and
   transaction intent in `api/service`. Do not import Gin or pass
   `*gin.Context` into the service.
6. Put ordinary SQL in `db/query/*.sql` and use generated SQLC.
   Treat database constraints and atomic authorization predicates as
   durable rules; do not invent per-table repositories.
7. Keep source-based and identity-based throttles distinct. Preserve
   middleware order and fail-closed behavior when appropriate.
8. Check `gofmt`, `go vet ./...`, `go build ./...`, `go test ./...`
   and HTTP denial/behavioral outcomes. Green compilation without tests
   does **not** prove response or authorization correctness.
9. Record changed contract, migrations, feature flags and rollback
   implications. Do not push outside a user-authorized task.

Use `CamelCase` method names that reveal the action. Follow Go package
conventions rather than reproducing Laravel Facades, hidden providers,
reflection-based dispatch or a duplicate response framework.
