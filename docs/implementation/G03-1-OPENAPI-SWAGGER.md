# G03.1 — Gin HTTP OpenAPI 3.1 and local Swagger UI

**Source of truth (current source code):** `api/server.go` route registration,
`api/*.go` handler request/response types, `api/errors.go` for public errors
and `api/browser_security.go` for origin/CSRF. Specification:
`api/openapi.json`, embedded in `api/openapi.go`.

## Purpose

Swagger/OpenAPI is **not** a new API protocol, and does **not** require
Protobuf, Redis or a gRPC server. This stage documents **existing Gin HTTP/JSON
contracts** without converting their handlers or changing auth semantics.

Current contract: 29 operations over 28 paths:
15 Authentication, 8 Organizations, 4 Sources and 2 System endpoints.
`/auth/token/renew` requires the opaque Session cookie; the signed Bearer
JWT is session-bound and is **not** sufficient to renew. Organization/Source
authorization stays in live SQL membership/permissions, not JWT claims.

## Run locally (no cloud deployment)

```bash
git switch dev
git fetch origin
git merge --ff-only origin/dev
make server
```

Development URLs when server uses the default `127.0.0.1:8080`:

- Swagger UI: http://127.0.0.1:8080/docs
- Canonical JSON contract: http://127.0.0.1:8080/docs/openapi.json

Docker Compose local port mapping exposes the same paths under:
http://127.0.0.1:18080/docs

Both routes are **unregistered outside `ENVIRONMENT=development`** and
return normal 404. The OpenAPI JSON is embedded at compile time, so source
and Go binary can't accidentally resolve to different checked-out spec
files at runtime.

**Important dependency:** Swagger UI JavaScript and CSS load from the pinned
`swagger-ui-dist@5.17.14` CDN (`unpkg.com`) while the **API and
specification are always local**. The UI consequently needs Internet access
for the JavaScript/CSS download (unless these assets are already cached). This stage does
not claim self-contained offline vendor assets, and it does not fetch
credentials from the CDN. Do not treat this as a public documentation endpoint
or expose development with live credentials. An offline, vendored UI can
replace these static URLs without changing the API spec contract.

## How to exercise APIs in Swagger

1. Open `/docs`. The browser obtains the JSON schema from this same local
   server and renders endpoints grouped by Authentication/Organizations/
   Sources/System.
2. For anonymous email/password or Phone OTP start/verify, fill in the JSON
   and use **Try it out**. The UI automatically sets
   `X-BDSPro-Request: 1` on unsafe methods; Gin's real CSRF/origin
   middleware remains active.
3. For authenticated endpoints, use **Authorize → bearerAuth → JWT** from
   the login response, or rely on the existing browser Session cookie.
   Swagger cannot manually set an HttpOnly cookie. JWT also depends on the
   live DB Session; logging out invalidates the JWT.
4. `POST /auth/token/renew` works using a **Session cookie**, not Bearer.
   Remember that the cookie is `bdspro_session` in development versus
   `__Host-bdspro_session` outside development.
5. Source mutation requires `Idempotency-Key`. A fresh write may return
   201, replay returns 200. No secret, raw OTP, access token or email
   verification token should be pasted into a shared documentation instance.

**Do not confuse docs execution with permission grants.** All live Gin
middleware, PostgreSQL session validation, SQL ownership/RBAC, throttling and
source idempotency still execute normally.

## Compatibility and acceptance

- Existing routes and wire formats are unchanged (especially public fields
  `access_token`, `token_type`, cookie names, error envelope and 204 bodies).
- `api/openapi_test.go` verifies every Gin route/method has **exactly one
  corresponding OpenAPI operation and vice versa**, required path parameters,
  responses, Session cookie renewal policy, source idempotency header, and
  absence of docs routes in non-development mode.
- CI still verifies migrations UP-DOWN-UP, generated SQLC source, gofmt,
  go vet, full Go tests/race, server startup and local Compose smoke.
- The Source and Organization payloads preserve their **current** Go JSON
  representation. For example, `organization.Member` has no explicit
  `json` tags so its fields currently serialize as `MembershipID`,
  `PersonID`, `JoinedAt`; the spec describes reality rather than
  silently refactoring their public API.
- The schema describes known success/errors, not every possible client
  input nuance or all SQL constraints. G03.2 can enhance examples and
  schema-level validation without changing the domain ownership.

## Next stages

- **G03.2:** enhance contract validation and examples, consider offline
  Swagger UI assets if fully disconnected dev is a requirement.
- **G06:** `proto/` (typed service contracts), generated `pb/` and
  `gapi/` (gRPC adapters); **Gin HTTP handlers stay supported** and both
  adapters call the same business services.
- **G07:** optional gRPC-Gateway for chosen RPCs, test parity with Gin
  endpoints rather than magically treating the existing Gin routes as
  generated Protobuf contracts.
- No AWS, deployment, worker, Redis, db schema or migrations in this gate.
