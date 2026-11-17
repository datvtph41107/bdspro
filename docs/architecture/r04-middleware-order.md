# R04 — Authentication preconditions and middleware ordering

## Source references

- Gitea: https://github.com/go-gitea/gitea/blob/main/routers/api/v1/api.go
- Gitea: https://github.com/go-gitea/gitea/blob/main/modules/web/router.go
- BDSPro: `modules/web/router.go`, `modules/web/handler.go`
- BDSPro: `services/context/api.go`, `services/context/gin.go`

## Contract

```text
BeforeRouting (outside matcher)
  -> Gin route match
  -> parent AfterRouting (APIContexter first)
  -> child AfterRouting
  -> Group guards (authentication, authorization)
  -> route actions
  -> endpoint
```

Gitea's API routes establish `APIContexter` before auth middleware.
BDSPro preserves this precondition: no middleware may require typed
`*APIContext` until the request context provider has run.

For a denied request, a guard either writes an API response or aborts
the typed handler chain, and the endpoint must not run. Authentication
should use `AuthenticationRequired()` for the consistent 401 response
and `WWW-Authenticate` header. A selected HTTP status without commit
or abort does not itself stop the downstream chain.

`BeforeRouting` can reject a request even if no route matches; it
must not depend on Gin route parameters or on `APIContext` created by
`AfterRouting`.

Route declarations are the visible record of preconditions; a handler
does not implicitly load an authenticated actor, repository, or other
resource. Runtime authorization implementations are separate from
this infrastructure milestone.

## Test coverage

`TestR04BeforeRoutingAndScopedAuthenticationOrder` exercises
pre-matcher denial, missing credentials, and successful authorization
across parent/child router composition; it checks the exact trace.
`TestR04TypedGuardAbortWithoutResponse` verifies that an abort-only
typed guard stops endpoint invocation. The bearer header in the
test is a synthetic fixture and must not be copied as real auth.

## Current caveat

The matcher is Gin, not Chi. Mounting a subrouter's `BeforeRouting`
under a dynamic path prefix such as `/orgs/{org}` requires separate
verification: the mounted ingress path matching currently operates on
a raw string prefix. Do not treat this as a verified Gitea parity case.
