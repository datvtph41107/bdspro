# R02 implementation status (2026-11-03)

## Base and reference

- BDSPro baseline: `e1c3085321a26ee244809be80977155b2a10df1d`, branch `refactor/gitea-ownership-r01` (PR #91; not in dev).
- Gitea reference: `12b20c7c9962e0be75912b290a0c5778799b0ad2`.
- No GitHub push, merge or remote branch mutation performed.

## Added code in the candidate patch

| Scope | Source | Reference | Status |
|---|---|---|---|
| Request lifecycle | `modules/reqctx/datastore.go`, `middleware.go` | Gitea `modules/reqctx/datastore.go`, `routers/common/middleware.go` | Unit + race tested locally |
| Routing mechanism | `modules/web/router.go`, `handler.go`, `router_combo.go`, `types/response.go` | Gitea `modules/web/*` | Adapter tested locally; router smoke type-checked with an isolated fake Chi surface, NOT actual Chi |
| Typed API context | `services/context/base.go`, `api.go`, `response.go` | Gitea `services/context/*` | Unit + race tested locally |
| Public errors | `services/context/errors.go`, `auth_response.go`, `validation.go` | Gitea `APIErrorAuto`, BDSPro R01 response mappings | Unit + race tested locally; must run R01 parity integration |
| Organization persistence | `models/organization/org.go`, `org_user.go` | Gitea `models/organization/org.go`, `org_user.go` | Compile-smoked against signature stubs; PostgreSQL tests NOT executed |
| Public representation | `modules/structs/organization.go`, `services/convert/organization.go` | Gitea `modules/structs`, `services/convert` | Compile-smoked; JSON contract test pending |
| Candidate organization HTTP | `routers/api/v1/*` | Gitea `routers/api/v1/api.go`, `org/org.go` | Unmounted candidate: 7 routes only; never use as a production replacement yet |

## Deliberate nonchanges

- Do not replace `routers.InitWebInstalled` or R01 Gin routes until all **34** method/path pairs and their auth/validation semantics are migrated.
- Do not change `cmd/web.go`, `cmd/worker.go`, `docker-compose.yaml`, `Dockerfile`, `Makefile`, SQL migrations or generated SQLC code without specific failures and regression tests.
- Do not enable GORM or introduce a second DB transaction engine.
- Do not remove existing `services/organization` or `modules/errors` until all callers migrate. The new model candidate is for testing separately, not a parallel production write path.
- Do not alter URI prefix (`/api/v1` is not deployed in R01), JWT/session semantics, OTP handling or async outbox payloads.

## Local verification already performed in the source workspace

