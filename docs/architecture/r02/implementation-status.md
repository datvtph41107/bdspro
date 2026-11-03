> **Historical R02 evidence.** R03 has preserved Gin and removed the unused Chi runtime candidate. See [R03 Gin ownership](../r03-gin-ownership.md) for the current design; benchmark entries below describe experiments, not the runtime.

# R02 implementation status (2026-11-03)

## Base and reference

- BDSPro baseline: `e1c3085321a26ee244809be80977155b2a10df1d`, branch `refactor/gitea-ownership-r01` (PR #91; merged into dev 2026-11-03).
- Gitea reference: `12b20c7c9962e0be75912b290a0c5778799b0ad2`.
- PR #90 (R00), PR #91 (R01), and PR #92 (R02 foundation) **have been merged into dev**. See verified merge commits below.

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
- Do not remove existing `services/organization` or `modules/errors` until all callers migrate. The new model is now invoked through a **temporary** R01 Gin service adapter; it is not a parallel SQLC write implementation. Delete the adapter after HTTP cutover.
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

**Acceptance status: R02 FOUNDATION MERGED; FULL R02 NOT COMPLETE.** GitHub Actions for the foundation passed on Go 1.27.3 with real PostgreSQL, Redis, Docker and HTTP smoke. The full 34-route Chi cutover and final performance/security acceptance remain outstanding.

## GitHub integration and measured evidence (2026-11-03)

| Scope | PR | Merge SHA | Evidence |
|---|---|---|---|
| Gitea-style runtime base | #90 | `3fc81ad6bf4362fbf315ee7034e7324dcb033f11` | dev workflow `41624403803` passed |
| Source ownership and migrations | #91 | `23dc0eaf593fc6c5dc32e956709903119d2a93d7` | dev workflow `41624508543` passed |
| HTTP/SQLC foundation | #92 | `36af7f7eadb321ff5ae02bbbd2398511f96c6ce2` | PR workflow `41627324759` passed |

The foundation measured three runs of a simple parameterized HTTP route benchmark:

- Gin: 181.4, 182.0, 183.2 ns/op; 208 B/op; 4 allocs/op.
- Chi native: 577.5, 567.7, 576.4 ns/op; 912 B/op; 8 allocs/op.
- Gitea-style with fast path: 568.4, 576.7, 577.7 ns/op; 912 B/op; 8 allocs/op.
- Typed APIContext handler (request context + status): 579.8, 580.9, 586.1 ns/op; 1056 B/op; 13 allocs/op.

These are **microbenchmarks**, not measurements of Gitea itself nor database-backed endpoint throughput. They do not establish end-to-end performance equivalence. No unsupported conclusion of full R02 readiness should be drawn from them.
