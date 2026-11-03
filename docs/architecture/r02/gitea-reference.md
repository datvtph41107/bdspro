# Gitea correspondence matrix — pinned source, not directory imitation

Reference: https://github.com/go-gitea/gitea/tree/12b20c7c9962e0be75912b290a0c5778799b0ad2

Gitea's own guidelines:

- [Backend](https://github.com/go-gitea/gitea/blob/12b20c7c9962e0be75912b290a0c5778799b0ad2/docs/guidelines-backend.md)
- [Refactoring](https://github.com/go-gitea/gitea/blob/12b20c7c9962e0be75912b290a0c5778799b0ad2/docs/guidelines-refactoring.md)
- [Testing](https://github.com/go-gitea/gitea/blob/12b20c7c9962e0be75912b290a0c5778799b0ad2/docs/testing.md)

| Gitea source and observable function | BDSPro R01 corresponding location | Decision and evidence required |
|---|---|---|
| `main.go`, `cmd/web.go`: `newWebCommand`, `runWeb` | `main.go`, `cmd/web.go` | Keep CLI command semantics; test startup and process lifecycle, don't rewrite the CLI for symmetry. |
| `routers/init.go`: `InitWebInstalled`, `NormalRoutes` | `routers/init.go`: `InitWebInstalled`; `routers/api/server.go` | Extract HTTP composition from the broad `Server` object without leaking DB/Redis management into routing. |
| `modules/web/router.go`: `NewRouter`, `BeforeRouting`, `AfterRouting`, `Group`, `Methods`, `Mount` | Native Gin routes in `routers/api/routes.go` | Prototype Chi-based routing **with equivalent execution order**, not only method names. Compare with Gin baseline. |
| `modules/web/handler.go`: `toHandlerProvider`, `preCheckHandler`, `executeMiddlewaresHandler` | Gin typed handlers | Prove admission short-circuit, registration errors and hot-path cost before replacing Gin. |
| `modules/reqctx/datastore.go`: `NewRequestContext` | Gin request context | Only introduce shared cleanup/data store with concrete consumers and concurrency rules. |
| `services/context/api.go`: `APIContexter`, `GetAPIContext`, `APIErrorAuto` | Gin context, `routers/api/response` | Typed principal and response ownership; preserve `{error, code}`, headers and 500 redaction. |
| `services/context/response.go`: `WrapResponseWriter`, `WrittenStatus` | Gin response writer/status | Ensure optional interfaces and no duplicate response writes. |
| `routers/api/v1/api.go`: resource route groups and typed endpoints | `routers/api/routes.go`, `routers/api/handler` | Move endpoints by resource and keep existing external URL prefix `""`; `/api/v1` is a separate API decision. |
| `modules/structs`, `services/convert` | `routers/api/handler/requests.go`, `resources.go` | Public representations independent of generated SQLC structs, with privacy/visibility tests. |
| `models/db/engine_init.go`: `InitEngineWithMigration` | `models/db/engine_init.go`: `Connect` | SQLC/pgx connection and canonical migrations; never add ORM `Sync` or `AutoMigrate`. |
| `models/db/context.go`: `WithTx`, `GetEngine` | `models/db/store.go`: `ExecTx` | Preserve **explicit pgx transaction-bound Queries**, unlike Gitea's implicit XORM session reuse. One transaction owner. |
| `models/organization/org.go`: `CreateOrganization`, `GetOrgByName` | `services/organization`, generated `models/db/organization.sql.go` | Organization data operations belong in model package where they carry semantics; no forwarding-only wrappers. |
| `models/organization/org_user.go`: `IsOrganizationOwner`, `IsOrganizationMember` | SQLC `UpdateOrganization` and `AddOrganizationMember` SQL predicates | Preserve authorization in atomic writes. Prechecking membership must not weaken SQL protection or add redundant DB calls. |
| `models/organization/org_test.go`: `TestCreateOrganization` | `services/organization/organization_create_integration_test.go`, `models/db/organization_authorization_test.go` | Re-run success, duplicate, permission, rollback, FK and SQL-authorization cases. |
| Gitea model-specific `ErrOrgNotExist` and `util.ErrNotExist` | `modules/errors`, response mapping | Relocate errors only with all public codes and HTTP contracts unchanged. |
| `modules/setting/database.go` | `modules/setting/config.go`, `database.go` | One INI load path, typed validated settings, explicit environment override precedence. |
| `modules/templates` + `templates/` | `templates/pages.go`, auth HTML, Swagger assets | Separate renderer logic from assets only when there is a real reusable mechanism. |
| Gitea `tests/integration` and colocated `*_test.go` | BDSPro R01 integration and architecture tests | Isolated PostgreSQL fixture setup; fail acceptance if all DB tests skip. |
| Gitea command/build/deployment conventions | `Dockerfile`, `docker-compose.yaml`, `Makefile`, `docker/` | Preserve Postgres 18, Redis 7, Mailpit, API/worker startup, config and developer commands. |

## SQLC and GORM decision

- SQL migration files under `db/migration` own schema.
- SQL in `db/query` owns query text. `sqlc.yaml` generates `models/db` code using `pgx/v5`.
- `models/organization`, `models/account`, and identity packages own operations with real semantics. They can depend on `models/db` without replicating SQLC model structs.
- **No automatic SQLC package sharding** in R02; separate generated packages need a transaction/type-duplication experiment.
- **No GORM production dependency, `AutoMigrate`, or second transaction path** while SQLC covers the current use cases. An isolated experimental adapter may be evaluated in a distinct PR and is not part of readiness.

## Anti-patterns rejected

- Creating an empty `modules/*` or `models/*` simply because Gitea has one.
- `Router` methods that silently reimplement Chi middleware composition with another executor.
- Wrapping every SQLC query with `Repository` and `Service` forwarding methods.
- Moving authentication/account flows into `models/db` solely to share a large `Store` interface.
- Opening a GORM transaction inside a SQLC pgx transaction, assuming they share atomicity.
- Removing tests or rewriting HTTP statuses just to make a directory change compile.

