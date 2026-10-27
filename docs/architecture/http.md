# HTTP — Gin routing, handlers, middleware and response

## Canonical source path

`core/router/server.go` creates the one Gin engine and global middleware,
then calls `routes.API(root, root.Group(apiPrefix), handlers, policies)`.
`routes/api.go` is the **only file that declares HTTP endpoints**, including
public, authenticated and conditionally enabled Google/Phone routes. Root
`/health` remains outside the API prefix.

`api/handler/` owns HTTP binding, admission using validated identities,
extracting the authenticated AccountID and presenting public responses.
Typed input contracts live in `api/handler/request/` and public output
resources in `api/handler/resource/`. `api/service/` owns business operations
and must not import Gin. `db/query/` and `db/sqlc/` own query execution.

## Middleware ownership and order

- Engine global: request ID, safe access logging, panic recovery, denial
  monitoring, body limits.
- API group: prefix and optional API-wide policy (none added just for style).
- Authenticated group: JWT verification once for its descendants.
- Per route: source limiter, login audit or other route-specific middleware.
- After binding in AuthHandler: identity-based limiter requiring input values.

Source-based and identity-based limits are **different policies**; moving a
limiter before parsing its identity or deleting it changes security behavior.
Keep `AbortWithStatusJSON` for middleware denials and propagate verified
AccountID through the established context key. Resource authorization belongs
to the operation and SQL predicates, not merely to JWT verification.

## Request, service and response

Handlers use Gin `ShouldBindJSON` / `ShouldBindQuery` once, return early
on invalid input and use `core/response.Send` for ordinary JSON responses.
Services return Go values and errors, not `gin.Context`, HTTP statuses or JSON.
`core/errors` maps known safe application failures; unknown database errors
remain internal. `core/response.Send` preserves the existing successful JSON
resources and error shape `{"error":"..."}`; `204` writes no body.
OAuth redirects/cookies and other protocol-specific responses are explicit
HTTP operations; do not force them into a JSON-only writer.

`AuthService` owns account registration, login, verification, session
renewal and revocation. Password, Google and Phone login share the
`service.IssueSession` implementation. `OrganizationService` owns
name validation and SQL error classification while HTTP handlers bind input.

## No second router or framework

Routes are Gin `GET`, `POST`, `PATCH` and `Group` calls. No custom
`Route::post` Facade, `core/route.Group` DSL, deferred mount, middleware
registry or per-request group lookup exists. The base prefix is composed
once at startup. Existing unversioned URLs must remain stable unless a
separate compatibility migration is approved.
