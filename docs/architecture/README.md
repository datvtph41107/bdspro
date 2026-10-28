# BDSPro — Current Application Architecture

**Current source of truth (Go packages under `internal/`):** `main.go`, `internal/api/server.go`,
`internal/api/routes.go`, `internal/api/handler/`, `internal/service/`, `internal/core/`,
`db/query/` and `internal/database/` on the `dev` branch.

This document describes the implementation, not the former
`bootstrap/`, `app/http/`, `internal/routes/auth.go` or business `modules/` layouts.

## Developer navigation

1. Open `internal/api/routes.go` to find the HTTP method, relative URL, scope,
   applicable middleware and handler method.
2. Open `internal/api/handler/<area>.go` to inspect body/query binding, verified actor,
   input admission, public resource and HTTP response.
3. Follow the called `internal/service/` operation to see business rules, session
   operations, transaction boundaries and database error classification.
4. Review `db/query/*.sql`, `internal/database/` and migrations for durable behavior.
5. Read `internal/api/server.go` for application-wide ingress, base-prefix
   policy and explicit dependency composition.

## Routing conventions

`Server.registerRoutes` calls native Gin GET/POST/PATCH and Group methods.
There is one Gin engine and no Facade, implicit service container, route
registry or custom request dispatcher. Gin resolves middleware order and
matched paths. Avoid a second routing convention.

The HTTP prefix is intentionally `""` to preserve current clients, OAuth
callback and verification URLs. Introducing `/api/v1` is a **public API
migration**, not a cosmetic refactor.

## Contracts and operational boundaries

- HTTP: [http.md](http.md) and [HTTP architecture](../http-architecture.md).
- Database: [data.md](data.md).
- Runtime, logging, queue and shutdown: [runtime.md](runtime.md).
- Current implementation and acceptance caveats: [status.md](status.md).
- Adding an endpoint: [workflow.md](workflow.md).

Historical planning and incident records under `docs/engineering/` preserve
their original decisions and commit context; they are **not** instructions
to resurrect removed source packages.
