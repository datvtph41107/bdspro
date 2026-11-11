# BDSPro configuration and runtime lifecycle

BDSPro adopts Gitea's division of responsibilities, not its Git-specific features. This keeps one configuration provider and predictable initialization paths.

## Startup

1. `cmd/main.go` declares commands and global flags. The CLI `Before` hook in `cmd/config.go` resolves paths, reads `app.ini`, applies `BDS__SECTION__KEY` overrides and configures logging once.
2. `modules/setting/setting.go` loads settings from `CfgProvider`. Each domain owns its loader: `database.go`, `server.go`, `security.go`, `oauth2.go`, `redis.go`, `queue.go`, `audit.go`, `mailer.go`, `phone.go`, `ipactive.go`. No domain reloads INI.
3. `routers.InitWebInstalled(ctx)` loads Web settings, initializes PostgreSQL and Redis, applies development migrations and builds HTTP services. `routers.NormalRoutes()` registers routes; `cmd/http.go` owns listener readiness and graceful shutdown.
4. `Server.Close()` releases resources initialized for that Web instance after HTTP requests have drained.

## Command isolation

`migrate` and `doctor` need database configuration only. `cleanup` additionally loads audit retention. `worker` loads database, Redis, queue, signing secret, mailer and audit retention; it does not initialize an HTTP server or require OIDC configuration.

## Ownership and behavior

Gitea is the reference for CLI hooks, per-domain setting loaders and the initialization graph. BDSPro retains its PostgreSQL/Redis/Gin dependencies and existing authentication, OAuth2, trusted-proxy, audit and queue validation rules. `setting.Config` remains a runtime input for existing HTTP services; `WebConfig()` projects the loaded domain settings without another parse or persistent duplicate configuration.

Reference files: [Gitea cmd/main.go](https://github.com/go-gitea/gitea/blob/main/cmd/main.go), [Gitea modules/setting/setting.go](https://github.com/go-gitea/gitea/blob/main/modules/setting/setting.go), [Gitea routers/init.go](https://github.com/go-gitea/gitea/blob/main/routers/init.go).
