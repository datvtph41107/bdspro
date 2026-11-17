# Gitea file/function organization alignment — M01

Source versions:
- Gitea `1424a2b7c2207d75a37aff0211740c9e35e58f48`
- BDSPro before M01 `dd2a4a36ceff62de30653163669836f224ad6bf1`

## Exact source-level mapping

| Gitea file/functions | BDSPro file/functions | Decision |
| --- | --- | --- |
| `cmd/web.go`: `runWeb`, `serveInstalled` | `cmd/web.go`: same bootstrap flow | Keep |
| `routers/init.go`: `InitWebInstalled`, `NormalRoutes` | `routers/init.go`: same coordination | Keep error-returning startup |
| `modules/web/router.go`: `Router`, `Mount`, `Group`, `Methods`, `Combo`, `GetForm` | `modules/web/router.go`: same; added `Combo` | Align |
| `modules/web/router_combo.go`: `Combo.Get/Post/Delete/Put/Patch` | `modules/web/router_combo.go`: same | Align |
| `modules/web/router_path.go`: `RouterPathGroup` regex matcher | `modules/web/router_path.go`: `ginPattern`, `matchesMountScope` | Same file responsibility; advanced regex API not implemented |
| `modules/web/handler.go`: `toHandlerProvider`, `prepareHandleArgsIn` | `modules/web/handler.go`: matching adapter | Keep |
| `services/context/base.go`, `api.go` | `services/context/base.go`, `api.go`, `gin.go` | Gin is transport boundary |
| `models/db/context.go`: `WithTx`, `WithTx2`, `GetEngine` | `models/db/context.go`: `WithTx`, `WithTx2`, `GetQueries` | Keep native pgx/SQLC |
| `models/organization/team.go`: domain operations | `models/party/person.go`: domain Party/Person operation | Move out of core DB |
| `services/user/`, `services/org/` | `services/jobs/`, `services/worker/` | Do not invent unused services |
| Gitea domain query methods with XORM | `models/db/sqlc/*.gen.go` | Generated adapter, not hand-edited |
| `routers/api/v1/org/`, `user/`, `repo/` | `routers/api/v1/api.go` only | Domain routes deferred until real handlers exist |

## M01 domain persistence migration

`models/db/person_party.go` had no Go call sites at source inspection.
Move `CreatePersonWithParty(ctx, *db.Queries)` into
`models/party/person.go` and remove the old implementation.
No backward-compatible second source of truth is created.

The three SQLC writes preserve the same caller-owned transaction.
Test the statement order and error short-circuit; PostgreSQL rollback
is provided by `db.WithTx`, not by this domain primitive.

## Not equivalent, intentionally

Gitea uses Chi + XORM; BDSPro retains Gin + pgx/SQLC.
Do **not** add XORM `RegisterModel`, empty model `Init` methods,
fake `ctx.Org`, or extra database stores just to match names.

`Combo` is now available at route declaration, e.g.
`m.Combo("/teams/{id}", guard).Get(getTeam).Patch(updateTeam)`.
Gitea's advanced `RouterPathGroup`, `Any`, `NotFound`
and classic HTML context/template features are **not** claimed as
implemented by this milestone.
