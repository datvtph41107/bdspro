# R13 — Gin HTTP application composition belongs to routers

Gitea reference: `routers/init.go::NormalRoutes` and `routers/api/v1/api.go::Routes` at `12b20c7`.

Prior to this PR, `routers/api/server.go` owned dependency creation, Gin middleware setup and the resulting router, even though the API package should own only API-specific routes and resources.

Now `routers/app.go` constructs the existing authentication and organization handlers; `routers/init.go::NormalRoutes` assembles Gin shared middleware, the API v1 route tree, CORS (through Server.Handler) and local Swagger UI. `routers/api` only retains API-specific routes, middleware, response and Swagger registration. `cmd/http.go` owns listener and shutdown.

No change to Gin, OAuth, OTP, rate-limit scripts, Redis keys, SQLC, existing URL paths or HTTP response contracts. **No parallel HTTP pipeline** is retained.

Existing route, OAuth, PostgreSQL and logging tests are moved with their owner from `routers/api` into `routers`; their assertions remain unchanged, except the Swagger registration test now uses `api.RegisterSwagger` and route setup uses `NormalRoutes`.

The Go 1.27.3 integration gate must pass route inventory, OpenAPI, unit/race, vet/build, PostgreSQL, Redis, Docker and HTTP smoke. Retain `Server.Handler` as the existing CORS-wrapped net/http entrypoint.
