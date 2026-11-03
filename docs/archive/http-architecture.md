# HTTP application contract

## Ownership and entrypoints

- `internal/core/router/server.go` constructs the one Gin engine, global middleware, HTTP
  server and route group. `apiPrefix` is deliberately empty to preserve existing
  URLs; versioning requires a separate public API migration.
- `internal/core/router/routes.go` is the one authoritative HTTP route inventory. Its function
  receives the already-configured Gin group, concrete handlers and middleware.
  Paths are relative to the supplied group; `/health` remains a root system route.
- `internal/api/handler` owns Gin request and response interaction. Per-endpoint input
  structures are under `internal/api/handler/request`; public wire resources are under
  `internal/api/handler/resource`.
- `internal/api/service` owns application and business operations and cannot import Gin.
  AuthService owns registration, email verification, token renewal and revocation;
  OrganizationService owns resource validation and PostgreSQL error classification.
- `internal/core/middleware` implements HTTP request policies. `internal/core/response`
  provides one shared JSON/error writer. `internal/core/errors` owns the public error
  catalog and internal causes. SQL and transactions remain in `db/sqlc`.

## Authentication: Laravel guard vs Go

Laravel DRX resolves `auth:api` through the `auth` middleware alias and the
`internal/config/auth.php` guard's driver and provider. Its `web`, `api`, and
`admin` guards have different identity/session implementations.

BDSPro currently has one principal type for HTTP authentication: Account,
authenticated by signed access JWTs, with refresh sessions in PostgreSQL.
`internal/config/auth.go` exposes the actual JWT configuration used by the one
token maker. `internal/core/middleware.Authenticate` validates access tokens and sets
the verified AccountID on the Gin request context. Resource ownership is
enforced by business operations and SQL predicates. No fictional `web` or
`admin` guard, generic alias lookup, runtime driver registry, or duplicate
session backend is added. If admin authentication requires another principal or
credential type, it must be designed and reviewed independently before its
routes are exposed.

## Middleware scope

1. Engine global: request ID, safe route-pattern access logs, recovery,
   denial monitoring, and request size limits.
2. API group: supplies the base prefix; it adds no extra middleware solely
   to alter formatting or URL presentation.
3. Authenticated group: verifies JWT exactly once before protected handlers.
4. Endpoint policies: admission rate limits and audit middleware where
   explicitly declared in `internal/core/router/routes.go`.
5. Identity-based throttling remains after request binding in auth handlers,
   because the limiter requires validated identifiers. It is not equivalent
   to source/IP admission limiting.

Gin combines group prefixes and handlers at **startup**. Route registration
does not create another dispatcher, group lookup, middleware loop, or per-request
prefix transformation. Lexical Go `{}` blocks are only source-level scopes.

## Compatibility and security invariants

The restructuring intentionally preserves all preexisting HTTP method/path
pairs, status/JSON wire contracts and conditional Google/phone exposure:

- Public auth and token endpoints remain at their current root URLs.
- `/health` is never pushed beneath the versioned API prefix.
- Google linking requires a JWT; Google callback remains public and must
  continue matching configured Google Redirect URI and the OAuth state cookie.
- Phone OTP endpoints exist only in the development environment, and phone
  binding requires JWT authentication.
- Redis admission failure remains fail-closed, while request monitoring has
  its distinct observation/failure policy.
- Logging uses full route patterns rather than user-controlled raw URLs.
- CORS continues to wrap the Gin engine in the HTTP Server handler.

Do not introduce `/api/v1` by changing `apiPrefix` without a deliberate
client migration, OAuth callback reconfiguration, verification-link review,
and explicit compatibility tests.

## Adding an API

Declare a relative method/path/handler in `internal/core/router/routes.go`; place public
request data types and response resources with the relevant handler; put
business rules in `internal/api/service`; use the existing SQLC-generated queries and
transaction helpers. Reuse `internal/core/response.Send` and classified core errors.
Do not create new middleware dispatchers, route registries or redundant
business repositories.

## Acceptance gates

- `go build ./...`, `go vet ./...`, and `go test ./...`.
- Confirm all method/path pairs and their global, group, and endpoint
  middleware match the pre-refactor inventory.
- Smoke test public/authenticated endpoints and denial cases, especially
  `401`, `429`, oversized-body `413`, and Redis-unavailable `503`.
- Validate conditional Google/phone routes and external callback paths.
- No route registration in other files, no old `app/http` or root `auth`
  packages, and no redundant Response Core implementation.

## Service boundary verification (2026-10-27)

- Auth handlers bind validated body/query inputs, perform identity-based request
  admission, call AuthService and send one response via Core Response.
- AuthService owns database/session workflows and canonical authentication
  errors. Password, Google and phone login reuse the same IssueSession action.
- Organization handlers contain no SQL error classification; operations
  validate names, enforce actor-related SQL predicates and map expected
  constraint failures in OrganizationService.
- `internal/config/auth.go` is typed JWT settings for the **existing** Account principal,
  not an implementation of Laravel's container-resolved guard registry.
  No inactive web/admin guard or provider is created until a real requirement.
- All endpoints retain their preexisting URLs, HTTP statuses, callback
  contracts and security middleware; `/api/v1` is not silently introduced.
