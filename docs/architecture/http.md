# HTTP — Gin routing, handlers, middleware and response

## Canonical source path

`internal/core/router/server.go` creates the one Gin engine, application handlers, and global middleware.
`internal/core/router/routes.go` declares HTTP endpoints directly through Gin, including
public, authenticated and conditionally enabled Google/Phone routes. Root
`/health` remains outside the API prefix.

`internal/api/handler/` owns HTTP binding, admission using validated identities,
extracting the authenticated AccountID and presenting public responses.
Typed input contracts live in `internal/api/handler/request/` and public output
resources in `internal/api/handler/resource/`. `internal/api/service/` owns business operations
and must not import Gin. `db/query/` and `internal/db/sqlc/` own query execution.

## Middleware ownership and order

- Engine global: request ID, safe access logging, panic recovery, denial
  monitoring, body limits.
- API group: prefix and optional API-wide policy (none added just for style).
- Authenticated group: JWT verification once for its descendants.
- Per route: source limiter, login audit or other route-specific middleware.
- After binding in AuthHandler: identity-based limiter requiring input values.

Source-based and identity-based limits are **different rate limits**; moving a
limiter before parsing its identity or deleting it changes security behavior.
Keep `AbortWithStatusJSON` for middleware denials and propagate verified
AccountID through the established context key. Resource authorization belongs
to the operation and SQL predicates, not merely to JWT verification.

## Request, service and response

Handlers use Gin `ShouldBindJSON` / `ShouldBindQuery` once, return early
on invalid input and use Gin `c.JSON` / `c.Status` for success, and `response.Invalid` or `response.Error` for errors.
Services return Go values and errors, not `gin.Context`, HTTP statuses or JSON.
`internal/core/errors` maps known safe application failures; unknown database errors
remain internal. `response.Invalid` handles malformed JSON (400) and oversized bodies (413).
`response.Error` maps known public application errors to their HTTP statuses, logs
unknown errors without exposing internal details, and returns generic HTTP 500.
Successful JSON uses Gin directly; `204` writes no body.
OAuth redirects/cookies and other protocol-specific responses are explicit
HTTP operations; do not force them into a JSON-only writer.

`AuthService` owns password registration, login, verification, session
renewal and revocation. `GoogleAuthService` owns Google account linking and login;
`PhoneOTP` owns phone validation and database error classification. Password,
Google and Phone login share `service.IssueSession`. OAuth state, PKCE,
redirects and cookies remain HTTP operations. `OrganizationService` owns
name validation and SQL error classification while HTTP handlers bind input.

## No second router or framework

Routes are Gin `GET`, `POST`, `PATCH` and `Group` calls. No custom
`Route::post` Facade, `internal/core/route.Group` DSL, deferred mount, middleware
registry or per-request group lookup exists. The base prefix is composed
once at startup. Existing unversioned URLs must remain stable unless a
separate compatibility migration is approved.

## Registration and task delivery

Registration writes Party, Person, Account and a job_outbox row in one
PostgreSQL transaction. A worker dispatcher polls committed rows, selects them
with FOR UPDATE SKIP LOCKED, enqueues to Redis Asynq with a stable task ID, and
marks queued_at. Failed Redis enqueue leaves the row pending. Duplicate email
messages can still occur after a crash/retry; delivery is **at least once** and
not an exactly-once SMTP guarantee.
