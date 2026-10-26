# BDSPro Architecture — Canonical Source Organization

**Working ref:** `refactor/canonical-architecture-20261026` from `dev@0cb3d050` (2026-10-26).
**Status:** evolving implementation; distinguish *implemented* from *approved design* and *open work*. Do not label this branch production-accepted until its CI, PostgreSQL, worker and HTTP contracts pass.

## Why the Laravel DRX working model matters

The useful property of DRX is **developer navigation**: begin with `routes/api.php` → a named controller/action → request contract → service/use case → persistence and infrastructure. Developers do not need to rediscover where the input, response, error and query should live for each endpoint.

BDSPro inherits this **mental model and source responsibility**, not Laravel's runtime or its implicit facades. PHP executes route files; Go cannot run package-level route calls. Gin already supports routing, grouping, admission and middleware. We therefore use a small **explicit one-time route composition** rather than a registry, reflection-based method resolution, `init()` hooks or simulated DI container.

Do not copy DRX's weaker mechanisms: `AppBaseController::sendError($exception->getMessage())` can leak internal faults, and defaulting every caught exception to HTTP 404 is not a valid backend error policy. Similarly, controller-side Eloquent operations are not a reason to implement an ORM over SQLC.

## What is canonical

1. **Developer entrypoint:** `routes/` (method/path/admission chain), then a controller action.
2. **HTTP contracts:** `app/http/requests/`, `binding/`, `responses/`, `resources/`; each has a different owner.
3. **Application operations:** `app/services/` when an operation coordinates meaningful steps; `auth/` is a legitimate existing reusable capability.
4. **Persistence:** `db/query/*.sql` source → SQLC-generated methods → one PostgreSQL Store/pool; business workflow decides transaction scope.
5. **Failure contracts:** `app/errors/` defines known public failures; unknown infrastructure errors are logged internally and sanitized at the HTTP boundary.
6. **Async execution:** `worker/` owns Asynq/Redis transport and retry decisions; `jobs/` owns job action; services state intent.
7. **Process start:** `main.go` delegates to `bootstrap/` for lifecycle and explicit dependency construction.
8. **Engineering rules:** review SQL, permissions, transaction semantics, request context, failure/retry and public response compatibility; do not replace code with style-only wrappers.

## Read next

- [HTTP / request / response ownership](http.md)
- [Database and transaction contract](data.md)
- [Runtime, worker, cache and error ownership](runtime.md)
- [Source map, implementation status and remaining gaps](status.md)
- [Developer recipe and change acceptance](workflow.md)

Prior decisions under `docs/engineering/` remain historical and must not be rewritten to imply that incomplete target architecture is already implemented.
