# R06 — Resource-scoped middleware and dynamic mounts

Gitea assigns resource context in route-scoped middleware. For instance,
`services/context/api_org.go` defines `APIOrganization` containing
`*organization.Organization` and `*organization.Team`, while route actions
read `ctx.Org.Organization` only after the appropriate assignment middleware.

BDSPro does **not** have an equivalent production organization model or
API context contract in the reset API, so this milestone does not introduce
a fake `ctx.Org`, a generic service locator, or an authorization policy.

## Runtime infrastructure fix

`web.Router.Mount("/orgs/{org}", child)` registers Gin's parameterized
pattern, but the mounted child's `BeforeRouting` previously matched
the literal text `/orgs/{org}`. The child ingress guard was therefore
skipped for a real path like `/orgs/acme/teams/42`. This is dangerous when
a developer uses child `BeforeRouting` for protocol/security checks.

`matchesMountScope` now matches path segments, recognizing simple
`{name}` and legacy `:name` parameter segments before Gin matching.
Static segments match exactly and the scope includes descendants.

This matcher checks only whether ingress middleware is within the mount's
path scope; it does **not** resolve resource identity, perform permission
checks, or replace Gin's matcher.

## Execution contract (synthetic test fixture only)

```text
Request /orgs/acme/teams/42
  -> child BeforeRouting (scoped pre-matcher)
  -> parent APIContexter
  -> Group resource assignment (load)
  -> Group permission check (403 or continue)
  -> typed endpoint reads assigned request data
```

A missing resource returns 404 and prevents the endpoint; insufficient
permissions return 403 and prevent the endpoint. Assignment runs once
per request, not once per registered route, and never uses process globals.

Production R06 completion requires a real resource/model and a concrete
`APIContext` field, plus the corresponding loader and permission checks.
The tests intentionally demonstrate the boundary without creating business
models during the infrastructure reset.

Sources:
- https://github.com/go-gitea/gitea/blob/main/services/context/api_org.go
- https://github.com/go-gitea/gitea/blob/main/services/context/api.go
- https://github.com/go-gitea/gitea/blob/main/modules/web/router.go
