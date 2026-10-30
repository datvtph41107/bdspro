# Go naming and code organization — BDSPro

Reference: [laravel-drx](https://github.com/datvtph41107/laravel-drx), especially `routes/api.php`, `app/Http/Controllers`, `app/Http/Requests`, `app/Http/Resources`, `app/Http/Services` and the login middleware. This is a **language/convention reference**, not a requirement to emulate Laravel's container, facades or middleware pipeline in Go.

## One readable path for a request

```text
internal/api/routes.go
  → handler/organization.go     HTTP action: Index / Show / Store / Update
  → service/organization.go     Business operation: List / Get / Create / Update
  → database/                  SQLC and transactions
  → handler/resources.go        Client response DTO
```

`handler/requests.go` owns HTTP request DTOs and validation tags. `handler/resources.go` owns HTTP JSON response types and conversions. These two files are kept within the **existing Go package** to avoid unnecessary dependencies and package churn; separate packages will require a concrete use case and review.

## Name by observable work, not an invented mechanism

| Prefer | Avoid when no concrete meaning | Rationale |
|---|---|---|
| `errors.go` | `catalog.go`, `failures.go` | Identify the contents rather than create taxonomy |
| `AuthThrottle.Allow` | `IdentityLimiter.Allow` | Authentication throttling can cover email, phone, tokens; it is not an identity model |
| `Index`, `Show`, `Store`, `Update` | Arbitrary synonyms for the same HTTP resource actions | Match route/controller discoverability in Laravel DRX |
| `requests.go`, `resources.go` | `contracts.go`, `representations.go` | Put incoming/outgoing HTTP data where a new developer looks first |
| `GetAccountByID` | An opaque `FetchData` or `Execute` | State the requested data and action |

Do **not** rename true domain/protocol terms merely to ban a word. `GoogleIdentity` is an OAuth identity relation in SQL, while `identity_store_unavailable` and `auth.google_identity_unlinked` are already public client codes. Keep those stable. Existing Redis key bytes such as `bdspro:auth_limit:v1:` and the nested `identity:` segment must be preserved without a planned migration. `policy` and `adapter` should not be invented as package labels, but existing *actual* authorization rules or library adapters require judgment rather than a global ban.

## How to write a new API operation

1. Add the Gin route directly in `routes.go`. Reuse native Gin middleware. No custom router registry or middleware-building loop.
2. Use a descriptive exported controller method. Prefer conventional resource methods for resource actions. Use an explicit verb such as `AddMember` for a non-CRUD action.
3. Put input struct and `binding` tags in `requests.go`; do not pass a Gin context into a service.
4. Invoke the service with `c.Request.Context()`. Services handle state changes, permissions and DB operations and return normal Go errors.
5. Map success through `resources.go`. Use `response.Error` for application errors and `response.Reject` for local HTTP errors. Keep Bearer and throttling helpers where they encode protocol semantics.
6. Add tests for malformed inputs, authentication, permission denials, error JSON, and success. Verify status/header/code compatibility.
7. Keep methods in a predictable order within a resource controller: `Index`, `Show`, `Store`, `Update`, then explicitly named secondary actions. Put local helpers after exported entrypoints.

Example currently in source:

```go
organizations.POST("", server.organizationHandler.Store)
organizations.GET("", server.organizationHandler.Index)
organizations.GET("/:id", server.organizationHandler.Show)
organizations.PATCH("/:id", server.organizationHandler.Update)
```

Use multi-line keyed literals when JSON mapping would otherwise require the reader to scan a long line. Apply `gofmt` and `go test`; do not create a special formatter or builder.

## Explicitly not copied from the DRX source

Laravel DRX has features appropriate to its PHP stack (facades, request lifecycle, exception rendering and IoC). This Go code uses Gin, normal Go errors, explicit constructors and SQLC rather than reconstructing those mechanisms. Do **not** copy full-request logging found in some Laravel middleware: requests can carry credentials, OTP, JWTs or OAuth tokens.

## Current boundaries / next review

This change standardizes terminology and one resource-controller example; it does not rename the `handler` and `service` package directories or add new `controllers` and `services` folders. Moving entire packages would require import-path changes across the repository and a separate demonstrated payoff. Likewise `SetTokenMaker`, `SetPhoneBinding` and variadic constructors still need a separate construction review, not an unverified renaming campaign.
