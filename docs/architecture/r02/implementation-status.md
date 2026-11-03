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

```
go test -race ./modules/reqctx ./services/context ./modules/structs
go test -race modules/web/handler.go modules/web/handler_test.go
go vet ./modules/reqctx ./services/context
```

A separate smoke harness with **synthetic stubs**, not real Chi/pgx, passed `go test -race` and `go vet` for the added packages. This verifies syntax and the assembled interfaces but is not a full real-dependency build and is not valid runtime/performance evidence.

## Mandatory acceptance before merge or cutover

1. Apply to exact R01 SHA; use Go 1.27.3 and real `github.com/go-chi/chi/v5@v5.5.0`, pgx and SQLC dependencies.
2. `go mod tidy`, `go test ./...`, `go vet ./...`, `go test -race ./...`, `go build ./...`.
3. `sqlc generate && git diff --exit-code -- models/db`.
4. Configure a **disposable** `bdspro_test` and explicitly run the PostgreSQL integration suite, checking it did not skip.
5. Compare each method/path, headers, status, JSON/error codes from the 34-route R01 inventory.
6. Compare middleware order, authentication, verified contact, 404/405, CORS, request ID, cancellation, Redis outage and response short-circuiting under real Chi.
7. Run DB/HTTP performance baseline and assess allocations and SQL calls; no performance claims from fake-dependency smoke tests.
8. Migrate the remaining 28 endpoints to `routers/api/v1/<resource>` with exact contracts and tests **before** removing Gin. Remove old mappings/services only after all uses are gone.
9. Verify `cmd`/worker, queue/outbox retry, templates/OpenAPI, Docker/CI and graceful shutdown without deleting or weakening existing tests.

**Acceptance status: NOT READY TO MERGE.** R02 foundations and one vertical slice candidate are implemented locally; the full R02 plan is not finished or verified.
