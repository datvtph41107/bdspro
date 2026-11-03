# R11 — HTTP listener lifecycle belongs to cmd

Reference: Gitea `cmd/web.go`, `cmd/web_graceful.go`, `routers/init.go` at commit `12b20c7c9962e0be75912b290a0c5778799b0ad2`.

## Before

`cmd/web.go` called `routers.InitWebInstalled`, which returned `*api.Server`. `api.Server.StartWithReady` owned net.Listen, http.Server timeouts, readiness callback and shutdown. This made `routers/api` responsible for both API routing and process lifecycle.

## After

`cmd/web.go` calls `serveHTTP(ctx, server.Handler(), address, onListening)`, implemented in `cmd/http.go`. HTTP server configuration and lifecycle are unchanged. The Gin server's `Handler()` still owns CORS wrapping. The API router no longer imports `context` or `net` for lifecycle management.

## Acceptance

- Existing 34 HTTP route contracts, OpenAPI, Gin and response codes unchanged.
- Identical ReadHeaderTimeout=5s, ReadTimeout=15s, WriteTimeout=30s, IdleTimeout=60s, MaxHeaderBytes=16KiB, shutdown=10s.
- Startup log `http.listening` and shutdown log `http.stopped` unchanged.
- New test checks bound listener readiness, dispatch and graceful cancellation; second test covers failed net.Listen.
- Go 1.27.3 GitHub Actions verifies compile, race, DB/Redis integration, Docker build, HTTP smoke.
- Does **not** add Chi, GORM, dependency registry or middleware loop.

Next distinct PR: move Gin HTTP composition to `routers/init.go` and provider selection to application startup, without introducing a new `modules/web` framework.
