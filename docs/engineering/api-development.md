# Developing a BDSPro HTTP API

This is a code-first handbook for the current **Gin REST monolith**. Use actual
functions and packages; do not build a second MVC framework.

## First read

1. `main.go` creates database, Redis, queue processor and the HTTP server.
2. `internal/core/router/server.go` creates one Gin engine and installs global
   request-ID, safe logging, recovery, request monitoring and body-size limits.
3. `internal/core/router/routes.go` registers **all** current HTTP endpoints
   and explicit authentication/rate middleware.
4. `internal/api/handler/` reads HTTP input and returns HTTP output.
5. `internal/api/service/` implements shared operations or business rules.
6. `db/query/*.sql` contains canonical SQL; `internal/db/sqlc/` has generated
   methods and the few explicit transaction operations.
7. `internal/core/response/writer.go` maps input or application errors to
   the public JSON error contract. Successful responses use native Gin.

## Writing a new endpoint

Do not create packages or wrappers as prerequisites. Check each layer for a
real responsibility.

**Route**: add a native Gin route to `routes.go`. Put it under
`authenticated` only if it must require a verified access JWT; distinguish
authentication from resource ownership checks. The route is the source of
truth for the HTTP method, path and middleware order.

**Handler**: use `ShouldBindJSON`, `ShouldBindQuery` or `ShouldBindUri` as
appropriate. For normal JSON input errors call `response.Invalid(c, err)`.
Take the authenticated `middleware.AccountID(c)` only on protected routes.
Invoke the operation. Check `err != nil` before sending success. Call
`response.Error(c, err)` for errors and Gin `c.JSON`/`c.Status` for success.
OAuth redirects and cookies remain explicit HTTP operations.

**Service**: create one when an operation has meaningful business checks,
coordinates multiple calls or must be reused. Do not pass `gin.Context` to
services. Pass `context.Context`, verified account identity and validated
inputs. A one-query read does not automatically need a service.

**Database**: write and review queries in `db/query/`; regenerate with
`make sqlc`. Do not hand-edit generated `*.sql.go` or `models.go`.
Use SQL conditions for resource-specific authorization where the write must
be guarded atomically. Do not create a repository that only forwards SQLC.

**Transaction**: use the existing `ExecTx` when a logical operation changes
multiple records atomically; do not start transactions from handlers.
External Redis/SMTP side effects cannot participate in PostgreSQL commit.
Registration records a job_outbox row in the same transaction and
an asynchronous worker enqueues it only after commit; do not queue from
within RegisterTx.

**Errors**: apply a public application error only at the point its meaning is
known. `response.Error` maps known public errors; unknown errors become
generic HTTP 500 and are logged privately. Do not return raw PostgreSQL
errors to clients. Keep validation and auth error status contracts stable.
Do not use magic status zero or a combined success/error function.

**Response**: use native `c.JSON(status, data)` or `c.Status(204)`.
Introduce a public resource type when the database/generated model could
expose private fields or change the public API unexpectedly. Avoid a resource
wrapper that adds no protection.

## Concrete example: Create Organization

```go
organizations.POST("", organizationHandler.Create)

// In OrganizationHandler.Create:
var req apirequest.CreateOrganizationBody
if err := c.ShouldBindJSON(&req); err != nil {
    response.Invalid(c, err)
    return
}
organization, err := h.organizations.Create(
    c.Request.Context(), middleware.AccountID(c), req.Name,
)
if err != nil {
    response.Error(c, err)
    return
}
c.JSON(http.StatusCreated, organization)
```

The service normalizes the name, resolves AccountID to PersonID, then calls
`CreateOrganizationTx`; the transaction creates Party + Organization +
owner membership, or rolls all of them back. SQLC queries come from
`db/query/organization.sql`.

## Safety invariants to test

- Access JWT is accepted; refresh JWT is not an access credential.
- An unauthenticated request cannot reach a protected handler.
- Input errors use 400 or 413 for oversized bodies, without internal details.
- Every business error uses its intended public status and message.
- Unknown errors are logged and return generic 500; no SQL details are exposed.
- Duplicate owner membership is prevented by PostgreSQL, including concurrent requests.
- Registration passwords respect bcrypt's 72-**byte** ceiling in addition to
  Gin's rune-based length tags.
- SQL transactions leave no partial Party/Organization on failure.
- Public response JSON does not leak authentication secrets.
- Optional Google routes exist only when configured; Phone OTP routes only
  in development. Rate limits and middleware ordering must survive refactors.

## Local and CI acceptance

```sh
gofmt -w <changed-go-files>
go vet ./...
go test ./...
go test -race ./...
go build ./...
```

Integration tests need a **disposable** PostgreSQL database named
`bdspro_test` with migrations applied and the environment variables
`BDS_TEST_MODE=disposable` and `BDS_TEST_DB_URL`. Never run the integration
tests against the main `bdspro` database.

The `.github/workflows/test.yml` pipeline tests with Postgres 18. After
merging or pushing, confirm its actual GitHub Actions conclusion; a successful
Git push or Go compile alone is not acceptance.

## Rules not adopted

- No automatic mandatory Controller → Service → Repository chain.
- No custom `Route::post` facade, DI container, generic handler adapter,
  Command Bus or generic `response.Success` on top of Gin without proof.
- No duplicate route packages, old and new response mechanisms or alternative
  raw SQL implementations.
- No service method that merely changes the name of one SQLC call.
- No unrequested change to the public URL, HTTP status, security or queue behavior.

## References and research limits

See [architecture-evidence.md](architecture-evidence.md) for the 50 verified
repository inventory and explicitly distinguished source-level reviews.

## Shared naming and ownership

Use a technical name that describes what the component **does**, not the first
business feature that happens to use it:

- A **job** is one business operation; use the `jobs` package when it has
  substantial execution behavior.
- A **task** is its Asynq wire type and payload. Adding a task introduces a
  new task type, payload and native `asynq.Handler`, registered at startup.
- A **worker** owns Asynq runtime, logging, retries and handler registration.
  Never add one new method per job to its transport interface.
- An **enqueuer** performs queue transport. A **job outbox** records one
  durable enqueue intent for a committed database transaction; a generic
  **dispatcher** moves intents to Asynq.
- Account identity is `accounts.id`; a **credential** proves an authentication
  method, and need not exist for every account. **External identity** maps a
  verified provider subject to an account. **Session** belongs to the account,
  regardless of login method.
- Use the established names and Go identifiers in a package. Do not introduce
  facades, registries, import aliases or convenience wrappers that only rename
  existing Gin, SQLC or Asynq calls.
- Comments explain invariant, security reason, lifetime or non-obvious
  behavior. Tutorial references belong in archival documentation, not active
  implementation commentary.

**Compatibility:** existing Redis key prefixes, Asynq task types, OAuth state
cookie and OTP HMAC domain string are stored or security-sensitive identifiers.
Do not remove `bdspro:` from Redis keys merely for aesthetics; these prefixes
also prevent collisions in a shared Redis instance. Do not edit applied SQL
migrations (including old tutorial comments): add a new migration instead.

## Google authentication

Option C uses the current Google callback for both sign-up and sign-in. New
Google subjects require a verified email; existing email is never enough to
auto-link another account. Password and Google accounts share AccountID,
sessions and authorization. The existing account-linking flow still requires
password reauthentication. See docs/operations/google-login.md.
