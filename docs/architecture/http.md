# HTTP: route, admission, request, controller, response

## Canonical reading path

`routes/auth.go` / `routes/api.go` → `app/http/middleware/` and `app/http/binding/` → `app/http/controllers/{auth,api,admin}/` → operation/Store → `app/http/resources/` → `app/http/responses/`.

`routes/` contains **native Gin calls** and must not know how PostgreSQL is connected, how a token is issued or how a job is executed. `routes.Mount` receives already-bound Gin handlers once at construction. Feature route files may use a private Go function to group declarations. There is no global mutable router or dynamic `route.To` resolver. New controllers should be added to the route composition at bootstrap, not looked up per request.

## Request vocabulary and source owner

| Input/source | Owner | Responsibility |
|---|---|---|
| Body | `app/http/requests/<area>/*Body` | JSON/form fields, tags, shape |
| Query | `app/http/requests/<area>/*Query` | Search, flags, paging input |
| Path | `app/http/requests/<area>/*Path` | Resource ID/slug |
| Header/actor | Middleware/admission | Token, source, identity; no caller-controlled actor |
| Binding | `app/http/binding` | Exactly one decode/validate pass per source, abort before controller |
| Controller | `app/http/controllers` | HTTP-specific action and projection |
| Resource | `app/http/resources` | Public fields, never expose a credential hash |
| Response | `app/http/responses` | JSON serialization, public error safety, HTTP codes |

**Input validation is not authorization.** A valid `organization_id` does not grant update permission. Authentication establishes account identity, while owner membership checks are performed in the data operation that mutates Organization.

## One bind/validation pass

Prefer the route middleware `binding.JSON[T](key)`, `binding.Query[T](key)`, `binding.Path[T](key)`; use `binding.Input[T](ctx,key)` in controller. This reuses Gin's decoders and validators, not a second parser. For login the invariants are deliberate: `SufLogin` wraps the request, source limiter is before JSON binding, identity limiter uses parsed input, then action runs. No mandatory universal chain fits Google OAuth callback, query verification link and JSON registration simultaneously.

Input contracts must have explicit names (`LoginBody`, `VerifyEmailQuery`, `OrganizationPath`). Business state checks stay in services/SQL.

## Response and error

Use the shared writer for JSON results. It emits the existing `{"error":"..."}` shape today; do not silently migrate to DRX's `{"success":true,"data":...}` envelope. It distinguishes known, safe business failures from unexpected backend errors. HTTP 204, redirects, file/download streaming and other non-JSON outcomes remain explicit actions.

In Go, `error` is returned, not thrown like a PHP exception. A named application error can be mapped once at the HTTP writer, but this conversion **is necessary**: the same business failure needs a different representation for HTTP and gRPC. There is no giant `switch` mapper and no automatic exposure of raw `err.Error()`. The catalog currently uses HTTP statuses as a practical API-facing definition; evaluate protocol-neutrality when RPC error handling is refactored.

## Reuse without magic

Pagination/filter are first **request contracts**, typed and validated, not strings appended to SQL. When multiple live consumers need the same default/max-limit or cursor behavior, extract the shared primitive. Do not manufacture `BaseController`, a universal builder, a `Route` facade or application-wide service locator before concrete use cases justify it.

**Current transitional seam:** several old actions still reside under `api/`, and their request/response helpers delegate to new owner packages. Move them by verified behavior, not by mass renaming.
