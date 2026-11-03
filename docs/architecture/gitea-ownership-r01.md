# R01 — Source responsibility review against Gitea

This review records concrete ownership decisions, not a pledge to reproduce
Gitea's folder tree. Baseline: `refactor/gitea-runtime-r00`.

## Sources examined

- https://github.com/go-gitea/gitea/blob/main/routers/init.go
- https://github.com/go-gitea/gitea/blob/main/routers/common/middleware.go
- https://github.com/go-gitea/gitea/blob/main/routers/api/v1/api.go
- https://github.com/go-gitea/gitea/blob/main/routers/web/web.go
- https://github.com/go-gitea/gitea/blob/main/models/db/engine_init.go
- https://github.com/go-gitea/gitea/blob/main/services/auth/auth.go
- https://github.com/go-gitea/gitea/blob/main/modules/setting/path.go
- https://github.com/go-gitea/gitea/blob/main/modules/setting/config_provider.go
- https://github.com/go-gitea/gitea/blob/main/Makefile
- https://github.com/go-gitea/gitea/blob/main/.air.toml
- https://github.com/go-gitea/gitea/blob/main/docker/root/etc/s6/gitea/setup
- https://github.com/go-gitea/gitea/blob/main/custom/conf/app.example.ini

## Decision table

| Gitea area | Actual responsibility | BDSPro decision |
| --- | --- | --- |
| cmd | CLI modes; startup/exit | web + worker + migrate |
| routers/init.go | installed-app initialization and dependency startup | Gin/config/HTTP delivery composition, not domain policy |
| routers/common | protocol middleware shared by HTTP surfaces | RequestID and CORS; API-specific error recovery remains under api |
| routers/api | API routes, handlers and middleware | Gin API, no URL version changes in R01 |
| routers/web | separate rendered user-facing web surface | **Do not create** until genuine web product/routes |
| routers/private | Gitea internal HTTP endpoint for git/SSH subprograms | **Do not create**; worker and HTTP share code, not internal HTTP |
| models/db | connection, persistence and DB engine | pgxpool, SQLC, SQL migrations; no GORM engine yet |
| models/repo, issues, secret, packages, shared | persistent domain records and queries | not BDSPro domains yet; avoid empty analogues |
| services/auth | business operations, account/session behavior | create services/auth from service root |
| services/organization | membership and organization workflows | create services/organization |
| services/actions, repository, packages | domain-specific workflows | no consumer today |
| modules/setting | config path, INI, common settings | ResolvePaths, config provider, typed validation |
| modules/log | sinks and logging facade | existing manager; no copied async writer manager |
| modules/redis | Redis connection for rate-limit/queue | explicitly named Redis, not a generic cache without cache semantics |
| modules/private | Gitea HTTP client for internal API | none; do not duplicate HTTP channels |
| modules/web, util, secret | technical HTTP, utilities, cryptographic primitives | Gin stays native; token/password helpers exist, no secret subsystem until proven |
| templates | rendered HTML assets | embed account verification/reset pages, preserve HTML escaping |
| options | Git license/gitignore/locale/static options | no comparable static catalog |
| web_src/public | frontend compilation and public assets | not introduced for REST-only backend |
| modelmigration | Gitea schema version evolution | keep canonical golang-migrate SQL files |
| tests | cross-component integration fixtures and harness | package-local tests and existing CI, no empty tests/ |
| tools | maintainer automation scripts | Air/Makefile/Docker scripts; new script only for repeatable nontrivial work |
| .air.toml | development watcher/build/restart | build to ignored bin/bdspro and restart web |
| build/bin | release packaging/build artifact | local bin/bdspro, no source-controlled executable |
| docker | image/first-boot config and process | template/runtime app.ini and isolated API/worker |

## Request path and ownership

```
main.go -> cmd.RunMainApp()
  -> cmd.runWeb() [runtime owner]
    -> setting.LoadConfig() [config read + validation]
    -> models/db.Connect() [pool health]
    -> modules/redis.Connect() [security Redis health]
    -> routers.InitWebInstalled() [HTTP dependencies]
    -> routers/api.NewServer() [Gin route setup]
      -> routers/common.RequestID, CORS
      -> routers/api/middleware [auth admission]
      -> routers/api/handler [parse and transport contract]
      -> services/auth, services/organization [business rules]
      -> models/db SQLC and transactions [persistence]
```

Worker path: same config/db/redis access, but starts Asynq processor and
outbox dispatcher, not Gin listener. SQL migration is a separate command
whose only required settings are DB_SOURCE and MIGRATION_URL.

## Discriminating acceptance

1. `go run . --help` succeeds without valid DB/Redis or config.
2. `make config-local` never overwrites a customized INI.
3. `make migrateup` and `make server` use identical INI DB target.
4. `bdspro migrate` works with only database configuration.
5. `bdspro migrate --down` refuses any mode other than development.
6. Route list, response codes, auth admission, SQLC query generation and database constraints unchanged.
7. `make build` outputs ignored `bin/bdspro`; Air watches only relevant source and runs web.
8. Docker image does not contain local `custom/conf/app.ini` or generated bin outputs.
9. Worker still dispatches outbox records and handles Asynq tasks; HTTP does not start processor.

## Deliberate differences and unresolved review

Gitea defaults work path to the installed binary directory. For BDSPro's
`go run .` workflow, default is current source working directory to avoid
the ephemeral executable path. Explicit config/work-path flags override it.

Gitea's Docker setup persists GITEA__ environment changes into INI;
BDSPro applies BDS__ overrides in memory. This intentionally avoids writing
secrets into persistent INI but requires documented precedence and diagnostics.

Gitea supports XORM and synced models; BDSPro retains SQLC and versioned
migrations. Only add GORM after an operational query/transaction experiment.

This PR does not prove multi-replica outbox concurrency or live Mailpit
end-to-end delivery. Those need explicit separate integration evidence.
