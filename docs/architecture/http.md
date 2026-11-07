# HTTP ingress and API ownership

BDSPro uses **Gin only**. `routers/app.go` constructs the HTTP server's
dependencies; `routers/init.go` installs ingress handlers via `NormalRoutes`;
`routers/api/v1/api.go` is the canonical declaration of public method/path
pairs. The source directory `api/v1` does not add a URL prefix.

## Owners

| Package | Current responsibility |
| --- | --- |
| `routers/common` | Request ID and CORS |
| `services/context` | APIContext/Wrap, first-party access JWT and verified-contact admission, 1 MiB request body cap, safe HTTP errors and validation |
| `services/security` | AuthThrottle, denied-request monitoring and login outcome logging |
| `modules/ipactive` | Per-action IPActive admission policies and limiter engine |
| `routers/api/v1/user`, `routers/api/v1/org`, `routers/api/v1/oauth` | HTTP endpoints; parsing, calling operations and returning resources |
| `models/organization`, `models/auth`, `models/db` | Persisted invariants, SQL authorization, transaction authority |

`routers/api/response` and `routers/api/middleware` were compatibility
packages removed by R39.4. No second response writer, middleware executor,
handler registry or route engine is introduced.

## Request path

```text
cmd/web.go -> routers.NewServer -> routers.NormalRoutes
  -> common.RequestID -> access logging / recovery
  -> security.RequestMonitor -> context.RequestIngress
  -> routers/api/v1.Routes
     -> context.Authenticate (first-party access JWT only, where required)
     -> context.VerifiedContact/VerifiedEmail (where required)
     -> IPActive / security.AuthThrottle (route and operation specific)
     -> endpoint / services / models
     -> context HTTP error/validation contract
```

Ingress execution order is defined by Gin route registration, not a custom
middleware chain. A verified JWT is not proof of a live PostgreSQL session;
authorization predicates remain in models/SQL at the mutation boundary.

## Dependency and failure policy

- Authentication throttling depends on Redis and fails closed with 503 when
  Redis is unavailable. Redis key namespaces `bdspro:auth_limit:v1:` and
  `bdspro:security:denied:v1:` are persisted operational contracts.
- Denied-request monitoring emits safe route-template logs and degrades to a
  warning when Redis counting fails; it does **not** block requests.
- The HTTP body cap is 1 MiB, enforced with Gin request ingress and
  `http.MaxBytesReader`; oversize requests retain the existing 413 response.
- HTTP error codes, `WWW-Authenticate`, `Retry-After`, and the error-code
  request context key are owned by `services/context`.
- `services/security` is an intentional HTTP-specific Gin adapter. Other
  business service packages must not import Gin.
- URL, payload and HTTP response changes require a separate public contract
  decision and tests.

## Source contract

- `routers/api/v1/api.go`: routes, groups and current method/path inventory.
- `templates/swagger/openapi.json`: reviewed OpenAPI 3.0.3 contract.
- `routers/swagger_contract_test.go`: API contract checks.
- `services/context/*_test.go`: safe error, validation and auth contracts.
- `services/security/*_test.go`: throttle, Redis key and monitoring contracts.
- `tests/architecture/ownership_test.go`: enforced import boundaries.
