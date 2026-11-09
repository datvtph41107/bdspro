# R01.8 — Modules own infrastructure; context and routes consume it

## Reason for this checkpoint

Gitea does **not** prohibit framework imports. Its `modules/web/router.go` builds on Chi, while `services/context/base_path.go` calls `chi.URLParam`. The architectural invariant is ownership: router composition, request data and form binding live at different source boundaries.

Verified reference source (Gitea):
- `modules/web/router.go`: `Router`, `BeforeRouting`, `AfterRouting`, `Group`, `Methods`, `Mount`; wraps Chi rather than replacing Chi
- `modules/web/middleware/binding.go`: form binding
- `modules/reqctx/datastore.go`: request-specific `ContextData`, cleanup/closers and scope creation
- `services/context/base.go`: Base references the datastore supplied by HTTP middleware rather than owning a second registry
- `services/context/api.go`: API-specific context (Doer, Org, Repo) and middleware adaptation

https://github.com/go-gitea/gitea/blob/main/modules/web/router.go
https://github.com/go-gitea/gitea/blob/main/modules/web/middleware/binding.go
https://github.com/go-gitea/gitea/blob/main/modules/reqctx/datastore.go
https://github.com/go-gitea/gitea/blob/main/services/context/base.go

## Actual BDSPro changes

- `modules/reqctx/datastore.go` now owns request data and cleanup; `APIContexter` creates one scope and invokes cleanup exactly once at request completion.
- `Base.Data` refers to that same request-local map. Unit tests that bypass middleware get a fallback isolated map, not a process/global registry.
- `modules/web/middleware/binding.go` owns typed JSON decoding/validation through the transport binder.
- `modules/web/form.go` stores typed form data through `reqctx.ContextDataProvider`. There is one request store for form and API context.
- `routers/api/v1/organization_context.go` calls the binder, invokes existing `APIContext.Validation` for public error mapping, then calls `web.SetForm` only on success.
- Gin remains the only dispatcher and JSON binder. SQLC remains the persistence engine. `services/context` owns public response/error semantics, not form-storage lifecycle.

## Contract under test

- Isolation between two requests, cleanup idempotency and late cleanup registration
- Parent cancellation/value preservation after scope installation
- Shared data between Base, module web form and middleware during one request
- Existing auth session, HTTP validation shape, errors, SQLC owner predicates and route inventories unchanged

## Not copied

Do not add Gitea's process tracing, locale, Chi before-routing hook or `responseStatusProvider` registry unless a BDSPro consumer needs it. BeforeRouting in Gitea can modify routing inputs **before** Chi matches a route; a Gin handler group runs after Gin has resolved the route, so using the same name would claim incorrect semantics.
