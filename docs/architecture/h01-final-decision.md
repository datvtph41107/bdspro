# H01 final decision: Chi-only HTTP runtime

Date: 2026-11-17. Branch: `dev`.

## Final production ownership

`routers/init.go` → `modules/web.NewRouter()` (Chi) →
`modules/web.Router` route compiler and mount →
`services/context.APIContexter()` (native net/http) →
typed API handler → services → models.

`modules/web/router.go` owns the single Chi matcher, HTTP handler compiler,
route scopes, method registration and structural mount.
`modules/web/router_chi.go` resolves trusted proxy IPs without Gin.
`services/context/http.go` owns the HTTP request source and cleanup.

Deleted: `modules/web/gin.go`, `services/context/gin.go`,
`modules/web/router_path.go`, Gin dependency and **all 69** tracked
`*_test.go` files across this repository.

## Acceptance limitation

Removing the tests was requested for a source cleanup. This also deletes
auth, DB, transaction, permission, lifecycle and HTTP regression coverage
outside the H01 scope. Build, go vet, migration, SQLC, Docker and the HTTP
health smoke remain CI checks. They **cannot prove business or security
behavior has remained unchanged**. Reintroduce focused tests before
production release.

Gitea reference: https://github.com/go-gitea/gitea/blob/1424a2b7c2207d75a37aff0211740c9e35e58f48/modules/web/router.go
