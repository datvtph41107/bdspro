# R05 — API JSON form binding

Gitea API source: https://github.com/go-gitea/gitea/blob/main/routers/api/v1/api.go

Gitea API routes implement a local `bind[T](tmpl T) any` helper that calls
`modules/web/middleware.BindFormAny`, writes errors through `APIContext`,
and publishes the successfully bound form using `web.SetForm`. This is distinct
from the separate `web.Bind[T]` used by Gitea's HTML form routes.

BDSPro's API currently consumes JSON through its Gin-backed `Base.BindJSON`.
The `routers/api/v1.bind[T]` helper keeps the same Gitea-style arrangement
but uses `middleware.BindJSON[T]` and the existing `ctx.Validation` HTTP
error contract. The form is stored only after successful decoding/validation.

Example route declaration (illustrative; not installed in production):

```go
m.Post("/teams", bind(CreateTeamOption{}), CreateTeam)
```

Handler:

```go
form := web.GetForm[*CreateTeamOption](ctx)
```

Execution: route matching → APIContexter → bind typed action → validation
or SetForm → endpoint → JSON response. Validation writes the established
BDSPro error envelope and aborts, so the endpoint never executes after
an invalid body. No second request decoder or JSON encoder is introduced.

The current binder accepts JSON only. Multipart/form/query binding and
Gitea's custom form validation are separate targets; do not imply they
are supported here. Authentication and resource guards must be declared
ahead of bind when present, so unauthenticated callers do not unnecessarily
parse the request body.
