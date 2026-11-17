# Chi migration: HTTP-native Swagger extraction

This incremental patch makes `routers/api/swagger.go` an ordinary
`http.Handler` without importing Gin. `SwaggerHandler("development")`
serves the existing embedded files and preserves the direct
`/swagger/index.html` URL; other modes return nil.

During the transition, `routers/init.go` installs a small Gin route
adapter that invokes this handler. The adapter must be removed when
`modules/web.Router` becomes Chi-backed.

This is **not** the complete migration. The following still import Gin:
`modules/web/router.go`, `modules/web/gin.go`,
`services/context/gin.go`, `routers/init.go` and existing
Gin-specific tests. Do not delete the dependency in `go.mod` until
the runtime and tests no longer import it.

Gitea source target:
https://github.com/go-gitea/gitea/blob/main/modules/web/router.go
