# Implementation Status, Acceptance Gates, and Execution Roadmap

> Snapshot created 2026-11-10. **Always refresh GitHub data at the start of a session.** A SHA listed here is historical evidence, not a promise that the branch still points there. Architecture is not accepted until all gates pass.

## Repository controls

| Field | Value |
|---|---|
| Repository | https://github.com/datvtph41107/bdspro |
| Protected baseline | `dev` (do not merge unfinished work) |
| Integration branch | `refactor/gitea-web-ownership-20261110` |
| Draft PR | https://github.com/datvtph41107/bdspro/pull/218 |
| Reference source | https://github.com/go-gitea/gitea (review `main`, pin relevant blob SHA for evidence) |
| Stack | Go 1.27.3, Gin 1.12, SQLC / pgx 5, PostgreSQL 18, Redis, Asynq |
| Current critical phase | **P01 in progress**, all-stage acceptance **NOT COMPLETE** |

### Verified reference snapshots

- Gitea `modules/web/router.go`: `2ddfbdaec67480fb6266e41f374c6ad73f033c3c`.
- Gitea `modules/web/handler.go`: `62fb0709b452be9b2ff46c4b8031faff28de3848`.
- Gitea `services/context/api.go`: `c14c3b3e467589904e4e3207cde1f4f65bca2140`.
- At the beginning of document creation BDSPro working code was at `9579336f1b74274bf24fb2c26d4cae5e6d11c9e5`. Later commits include this document; **refresh the branch HEAD**.
- GitHub CI run for that pre-document code: https://github.com/datvtph41107/bdspro/actions/runs/42753671358. It was observed `in_progress`; do not infer success.
- One earlier green baseline on this PR was `f434e5ed`, workflow https://github.com/datvtph41107/bdspro/actions/runs/42746654905. This **does not certify current HEAD**.

## Phased project state

| Phase | Scope | State | Completion test |
|---|---|---|---|
| **P01** | `modules/web/router.go`, `handler.go`: Gitea-style route/handler pattern on Gin | **IN PROGRESS** | single non-generic Router; one provider/handler compilation path; true pre/post match middleware; API+OAuth tests + CI green |
| **P02** | `services/context/{base,api,...}`: net/http Base, APIContexter, request and response ownership | **NOT CLOSED** | remove redundant Gin context adapter; one APIContext; correct request scope, cleanup and writer semantics |
| **P03** | `routers/api/v1`: typed resource actions, guards, Auth/OAuth, API behavior | **NOT CLOSED** | one route convention, same endpoints/body/status/auth/CSRF/cookies/redirects |
| **P04** | `models/db`: canonical SQLC executor, nested Tx, rollback/commit and session authority | **PARTIAL** | no duplicate Tx public APIs, tx-bound queries, race and rollback assertions |
| **P05** | `models` + `services`: Gitea-like domain ownership/function names/errors | **NOT CLOSED** | consistent import/call graph, no transport leakage or unnecessary store forwarding |
| **P06** | `cmd`, `routers/init.go`, `modules/{setting,log,cache,queue,graceful,...}` | **NOT CLOSED** | process init order, fail-fast, cleanup, worker and CLI lifecycle checks |
| **P07** | full-source acceptance & readiness | **NOT STARTED** | Go 1.27.3 full CI, OpenAPI, SQLC, schema, behavior, security, runtime, import graph; then merge only when eligible |

Do not equate progress through P04's partial user-service refactor with P01 completion.

## P01 — actual before/current/target change inventory

| Owner | Before | Current integration (verify HEAD) | Remaining |
|---|---|---|---|
| `web.Router` | `Router[T]` tied to `gin.RouterGroup` | same generic type, with Gitea-style `curGroupPrefix`/`curMiddlewares` | replace with `Router` and `NewRouter()`, keeping Gin implementation |
| `web.AfterRouting` | Gin-wide/context adapter | matched route composition plus HTTP providers | unify middleware kinds in common chain |
| `web.Methods` | single method typed actions | comma-separated methods/Head supported | Gitea `wrapMiddlewareAndHandler` pre/normal ordering, method/handler validation |
| `web.handler` | `r.handlers` and `gin.HandlerFunc` return contract | package `middlewareProvider`, `toHandlerProvider`, `RegisterResponseStatusProvider`, `preCheckHandler`, `prepareHandleArgsIn`, `hasResponseBeenWritten` | remove `r.toGinHandler` and redundant typed execution path |
| `web.BeforeRouting` | no matching public API | **not implemented** | truly execute pre-match using HTTP boundary and **same Gin matcher**, no parallel router |
| `context.APIContext` | Gin accessor only | registry in `api.go`, request access in `GetAPIContextFromRequest` | move request creation/publication to canonical HTTP lifecycle; remove Gin context compatibility |
| `context.Base` | `RequestSource`, `ginRequestSource` | old adapter remains | `NewBaseContext(http.ResponseWriter,*http.Request)` target; no duplicate response writers |
| `services/user` | explicit `WithTxResult(ctx,store,...)` | `WithTx2(ctx,...)` and `TxQueries(txCtx)` for Profile/Settings; actor capability validated | P04 review all remaining users and nested semantics |

### Commit evidence (not a claim of acceptance)

- `3d4a4677`: multi-method `Methods`, `Head`, registration tests.
- `9579336f`: simplified `toHandlerProvider` by separating `preCheckHandler`, `prepareHandleArgsIn`, `hasResponseBeenWritten` following Gitea's function organization.
- `b15ba283`: compatibility adjustment of `r.getContext(c)` before registered API action; this is **transitional**, not the canonical P01 target. Investigate/test alongside context lifecycle, do not perpetuate without a reason.
- `fb45b680`: updates a direct org-route test fixture to install `APIContexter`; source and later compatibility commit may have been modified by another writer. Verify both.
- Earlier commits introduced HTTP middleware provider, Gin bridge, wrapped response writer tests, and request-published `APIContext`. These remain subject to final CI.

## Failure history — must not lose

1. A user-service refactor changed the ordering of missing Store vs validation errors. `TestUserServiceRequiresVerifiedActorAndRequestStore` failed. Restored explicit missing-Store validation; never remove regression tests to hide behavior.
2. A handler refactor caused `gofmt` failures in `modules/web/handler.go`; formatting was corrected.
3. Routing `*APIContext` through HTTP providers made direct Gin test fixtures without `APIContexter` panic with `APIContext missing from HTTP request`. The org fixture was updated; a separate compatibility commit still invokes `getContext` before provider. **Do not confuse legacy fixture support with final architecture.**
4. A native `net/http` middleware can pass a wrapped `ResponseWriter` into `next`. Gin actions must not bypass that writer. A Gin bridge/writer was introduced and tested; revalidate `Hijack`, `Flush`, `CloseNotify`, status-only and duplicate-commit paths.
5. A full CI success at a historical commit cannot be used to certify newer commits.

## Exact P01 to-do, in dependency order

### P01-A: audit + decide semantics
- Fetch Gitea's `handler.go`, `router.go` **and real call sites** from `routers/api/v1`.
- Compare `middlewareProvider`, `toHandlerProvider`, `preCheckHandler`, `prepareHandleArgsIn`, `hasResponseBeenWritten`, `wrapMiddlewareAndHandler`, `BeforeRouting`, `AfterRouting`, `Methods`, `Group`.
- Reject function-name parity if Gitea and BDSPro diverge on middleware continuation, route params, error/response or status provider.
- Before implementing new wrappers, find whether existing helpers are redundant.

### P01-B: finalize one handler compiler
- One accepted action signature mechanism, registered typed `*APIContext`; no duplicate `toGinHandler` for application actions.
- Canonical `wrapMiddlewareAndHandler` that handles **pre** middleware, standard middleware, endpoint, nil/unsupported signatures and short-circuit.
- Reflection/signature validation once at registration, not per-request routing setup.
- `responseWriter` tracks committed output, not selected-only HTTP status. Preserve status, body and header semantics.

### P01-C: Router API and Gin boundary
- `type Router struct`, `NewRouter()`, `BeforeRouting`, `AfterRouting`, `Group`, `Methods`, `Get/Post/Put/Patch/Delete/Head`, `ServeHTTP` as needed.
- Gin stays sole engine; no Chi import or extra independent matcher.
- All registered handlers/guards migrated to same canonical contract; no compatibility API left when P01 accepted.
- Test method/path mutation truly before Gin match, route params after match; matched/unmatched path and method behavior.

### P01-D: prove behavior and performance
- Tests: `TestTypedAPIContextResponseStatusControlsActionChain`, new web router tests, org/teams Auth/OAuth flows, request-local cleanup, concurrent session revocation and mutation.
- CI: gofmt, SQLC generation, OpenAPI, migration, `go test ./...` (as CI config), route inventory, HTTP smoke, recovery, runtime image, benchmark; Go **1.27.3**.
- Compare Gin production route benchmark to a **matched baseline**; report any overhead of reflection/provider and additional allocations. Do not claim a target or improvement without measurements.
- Show code diff side by side and name which Gitea source snippet supports each decision.

### P01-E: finalize P01
- No generic Router, no legacy typed/Gin action compiler, no misleading duplicate context source, true pre/post route separation, CI final HEAD green.
- Only then mark P01 **ACCEPTED** and proceed to P02.

## Suggested acceptance commands in a compatible Go 1.27.3 environment

~~~sh
git fetch origin dev refactor/gitea-web-ownership-20261110
git switch refactor/gitea-web-ownership-20261110
git status --short
git diff origin/dev...HEAD -- modules/web services/context routers
go version                  # must match required toolchain
gofmt -l modules/web services/context routers
go test ./modules/web ./services/context ./routers/...
go test ./...               # or exact CI make test
~~~

Also run `make sqlc`, schema/OpenAPI checks, worker and full runtime checks from the repository's current `.github/workflows/test.yml`, not guesses. Do not launch network/DB tests unless dependencies are actually available. On ChatGPT tools that cannot run Go 1.27.3 locally, rely on **actual GitHub Actions job/step logs** and state the limitation.

## Reporting format after each coherent patch

~~~text
Checkpoint: P01.X — [behavior]
Gitea: file:line + signature/import/call reason
BDSPro before: file:line + why it differs
BDSPro after: file:line + complete relevant function or source link
Invariant: what remains unchanged (Gin, SQLC, API, Auth/OAuth, session)
Evidence: commit SHA, PR diff URL, current HEAD CI run URL/status, failing test if any
Outstanding: next smallest coherent blockers
Merge: NO until P01-P07 acceptance
~~~

Next session should read [CONTINUE_PROMPT.md](./CONTINUE_PROMPT.md) and [README.md](./README.md), refresh branch HEAD, fix any CI failure first, then continue P01.
