# R01 — Router parameter syntax and scope

Gitea's `web.Router` declares named URL segments as `/{name}`. BDSPro
currently uses Gin for matching, where the same parameter is written
`/:name`. `modules/web.Router` now translates simple `{name}` segments
**only when registering with Gin**; route descriptors stay in Gitea form.

```go
m.Group("/organizations/{org}", func() {
    m.Get("/teams/{team}", GetTeam)
})
```

The existing `/:id` syntax remains accepted during migration. Unlike Chi,
Gin does not support Chi's parameter regexp syntax. For now, forms such as
`/{sha}.{diffType:diff|patch}`, `/{id:[0-9]+}`, and mixed literal segments
are rejected rather than silently registered as literal routes. Wildcards
must be evaluated separately before standardizing them.

The `Group` callback runs once at route registration. Group middleware
is scoped to routes registered inside the callback and restored afterward.
`Mount` installs compiled child route actions on the parent Gin matcher,
so the same request uses exactly one matcher. `BeforeRouting` is outside
the matcher, while `AfterRouting` and group guards run only on matched routes.

This is an adapter compatibility step, not a migration from Gin to Chi.
It introduces no new request or service dependency.

Source reference:
https://github.com/go-gitea/gitea/blob/main/modules/web/router.go
