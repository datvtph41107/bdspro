# H01: Application boundaries before HTTP router migration

## Goal

Keep business services and domain models independent from HTTP routing
frameworks. A router replacement should not trigger changes to persistence
or business orchestration.

## Source-backed baseline

- Gitea uses `modules/web.Router` backed by Chi; typed handler compilation
  remains in `modules/web/handler.go`.
- Gitea's `services/org.NewTeam(ctx context.Context, ...)` accepts a Go
  context rather than a Chi context.
- BDSPro currently uses Gin in `modules/web/router.go`, `modules/web/gin.go`,
  and `services/context/gin.go`.

## Enforced contracts

`tests/architecture/infra_reset_test.go` checks that required runtime
resources and entrypoints exist, including `services/jobs`,
`services/worker`, and `routers/common`. It no longer rejects legitimate
future domain packages such as `models/organization` or `services/org`.

The import-boundary test checks production Go files under `models/` and
`services/`, excluding `services/context` (an HTTP application adapter).
It prohibits direct Gin, Chi, application router and APIContext imports.

This permits either Gin or Chi at the HTTP boundary without making domain
code depend on that choice. It does **not** mean a router swap is complete.

## Follow-up

1. Preserve one request-scoped `APIContext` and stable cleanup semantics.
2. Replace the Gin-backed matching/response/request-source implementation
   atomically with HTTP-native routing and tests.
3. Remove Gin only after caller, tests, proxy handling and binding are ported.
4. Run Go tidy, unit/integration tests, race tests and CI before declaring H01
   complete.

Reference:
https://github.com/go-gitea/gitea/blob/1424a2b7c2207d75a37aff0211740c9e35e58f48/modules/web/router.go
