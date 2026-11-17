# H01 — Framework ownership at HTTP composition root

Gitea target: `modules/web.Router` wraps a Chi matcher while `routers/`
declares actions against the application's router contract. BDSPro still
uses a Gin matcher temporarily, but `routers/init.go` is now matcher-agnostic.

## Runtime call flow

`routers.NormalHandler()`
→ `web.ConfigureDefaultMode()`
→ `routers.NormalRoutes()`
→ `web.Router.Get("/swagger/*filepath", swagger.ServeHTTP)`
→ `web.Router.SetTrustedProxies(settings)`
→ `web.Router.WrapIngress()`.

The transport-specific `gin.SetMode` and `gin.Engine.SetTrustedProxies`
calls belong to `modules/web/router.go`, not to application bootstrap.

The architecture test now rejects direct Gin/Chi imports in production
`routers/` packages, and in business `services/` / `models/`.

This is an incremental boundary correction, **not** a claim of completed
Gin removal. Gin still backs route matching and `services/context/gin.go`
is still required. Replacing it requires an atomic migration of the
request/response writer and typed middleware tests.
