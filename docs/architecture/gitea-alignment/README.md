# BDSPro × Gitea — Canonical Architecture & Source Patterns

> Status: **TARGET DESIGN / NOT ACCEPTED** · 2026-11-10 · Integrating PR [#218](https://github.com/datvtph41107/bdspro/pull/218). This document is the architecture contract, not proof that the implementation is finished. Always consult [IMPLEMENTATION_STATUS.md](./IMPLEMENTATION_STATUS.md) and the latest GitHub HEAD/CI.

## 1. Objective and non-negotiables

BDSPro seeks Gitea's **package responsibility, import direction, function ownership, handler orchestration, runtime initialization discipline, and maintainable code patterns**. This is NOT a business-domain copy and NOT a framework migration.

- **Keep** Go 1.27.3, Gin 1.12, SQLC generated Go, pgx 5, PostgreSQL 18, Redis, Asynq, current JWT/session, OAuth2/OIDC contracts, Docker/CI and the established API.
- **Gitea reference stack**: Chi, XORM; they are references for the design boundaries, **not replacements** for Gin/SQLC.
- Keep one canonical source of truth per responsibility; no extra controllers/handlers, duplicate error mappers, duplicate transaction engines, or adapter layers just to mimic function names.
- Compare actual source **directory → file → import graph → types → function signature → call graph → request/transaction lifetime**. A matching name without matching semantics does not pass.
- Preserve security invariants: verified acting session, transaction/authorization ordering, fail-closed security paths, no raw DB/provider secrets in public API errors, OAuth redirect/cookie/CSRF semantics.
- Do not merge partially migrated code into `dev`. Use an integration branch and a draft PR until final full-source CI/runtime acceptance.

## 2. Source of truth and links

| Concern | Gitea source reference | BDSPro integration source |
|---|---|---|
| Router | [Gitea modules/web/router.go](https://github.com/go-gitea/gitea/blob/main/modules/web/router.go) | [BDSPro modules/web/router.go](https://github.com/datvtph41107/bdspro/blob/refactor/gitea-web-ownership-20261110/modules/web/router.go) |
| Handler & continuation | [Gitea modules/web/handler.go](https://github.com/go-gitea/gitea/blob/main/modules/web/handler.go) | [BDSPro modules/web/handler.go](https://github.com/datvtph41107/bdspro/blob/refactor/gitea-web-ownership-20261110/modules/web/handler.go) |
| Typed API context | [Gitea services/context/api.go](https://github.com/go-gitea/gitea/blob/main/services/context/api.go) | [BDSPro services/context/api.go](https://github.com/datvtph41107/bdspro/blob/refactor/gitea-web-ownership-20261110/services/context/api.go) |
| Base context | [Gitea services/context/base.go](https://github.com/go-gitea/gitea/blob/main/services/context/base.go) | [BDSPro services/context/base.go](https://github.com/datvtph41107/bdspro/blob/refactor/gitea-web-ownership-20261110/services/context/base.go) |
| API registration | [Gitea routers/api/v1/api.go](https://github.com/go-gitea/gitea/blob/main/routers/api/v1/api.go) | [BDSPro routers/api/v1/api.go](https://github.com/datvtph41107/bdspro/blob/refactor/gitea-web-ownership-20261110/routers/api/v1/api.go) |
| Init | [Gitea routers/init.go](https://github.com/go-gitea/gitea/blob/main/routers/init.go) | [BDSPro routers/init.go](https://github.com/datvtph41107/bdspro/blob/refactor/gitea-web-ownership-20261110/routers/init.go) |
| Transactions | [Gitea models/db](https://github.com/go-gitea/gitea/tree/main/models/db) | [BDSPro models/db/context.go](https://github.com/datvtph41107/bdspro/blob/refactor/gitea-web-ownership-20261110/models/db/context.go) |

Reference blobs verified when this document was drafted: Gitea `router.go@2ddfbdae`, `handler.go@62fb0709`, `services/context/api.go@c14c3b3e`. Reference `main` can subsequently change; fetch and compare new blobs before edits.

## 3. Canonical ownership and import graph

~~~text
cmd/                      process lifecycle: config → initialize → serve → shutdown
  └─ routers/init.go      sequence dependencies; build final router (Gin stays)
       ├─ modules/web     framework-neutral HTTP action contract;
       │   ├─ handler.go  middlewareProvider, toHandlerProvider, validation,
       │   │              typed provider registry, response status/continuation
       │   └─ router.go   route declarations, scope, method registration,
       │                  ONE Gin execution/matching boundary
       └─ routers/api/v1  routes, admission/authorization, bind, call, emit
            └─ services/context
                 ├─ Base / APIContext   request scope, HTTP response/error
                 └─ APIContexter       one context per HTTP request
                    ├─ services/{domain}   orchestration, multi-model operations
                    │    ├─ models/{domain} persistence/domain errors/invariants
                    │    └─ models/db       transaction/executor
                    └─ models/{domain}     direct reads/simple operations when appropriate
modules/{util,setting,log,queue,cache,reqctx,...}  shared capabilities as actually consumed
models/db/sqlc generated output only; SQLC/pgx are retained.
~~~

This is a logical dependency/call map, not a strict prohibition on `services` importing `models/db`: Gitea services use DB functions directly. Conversely, routes need not always traverse services to reach a simple model read. **No HTTP transport import inside domain models/services**.

### Package ownership rules

| Package | Owns | Must not own |
|---|---|---|
| `modules/web/router.go` | routes, groups, path/method matching configuration, pre/post-routing scope, Gin-specific boundary | domain DB reads, API JSON envelopes, duplicate typed action reflection |
| `modules/web/handler.go` | middlewareProvider, toHandlerProvider, action argument resolution, precheck, HTTP continuation | Gin/Chi-specific request context, domain decisions |
| `services/context` | Base, APIContext, one request-local state, response/error | application DB initialization, router matching logic |
| `routers/api/v1` | request-specific guards/binding/action invocation and HTTP result | SQL driver error parsing, long domain transactions |
| `services/domain` | multi-model workflows/Tx, domain policy, error context | Gin response writes |
| `models/domain` | queries/invariants/domain errors, call SQLC via canonical DB access | HTTP response status, Gin |
| `models/db` | Store/Queries, transaction lifecycle, SQLC executor selection | auth/account business policy |
| `modules/util` | reusable error categories and helpers | blanket repository of all domain errors |

## 4. P01: exact Gitea code pattern and semantic correspondence

### Gitea actual functions

From `modules/web/handler.go`:
~~~go
type middlewareProvider = func(next http.Handler) http.Handler
func toHandlerProvider(handler any) middlewareProvider
func preCheckHandler(fn reflect.Value, argsIn []reflect.Value)
func prepareHandleArgsIn(resp http.ResponseWriter, req *http.Request, fn reflect.Value, fnInfo *routing.FuncInfo) []reflect.Value
func hasResponseBeenWritten(argsIn []reflect.Value) bool
func executeMiddlewaresHandler(w http.ResponseWriter, r *http.Request, middlewares []middlewareProvider, endpoint http.HandlerFunc)
~~~

From `modules/web/router.go`: `Router` is not generic; it stores `chiRouter`, `afterRouting []any`, `curGroupPrefix`, `curMiddlewares []any`. `wrapMiddlewareAndHandler` compiles *pre* middleware, ordinary middleware, then the endpoint; `Methods` registers multiple verbs. `BeforeRouting` executes ahead of matching, `AfterRouting` can access params.

Gitea's `toHandlerProvider` is **short by deliberate decomposition**: identify wrapper middleware; prepare/validate arguments via helper; invoke once; use `hasResponseBeenWritten`; continue only if unwritten. The routing framework does not appear in its imports. It does not hand-roll a `switch` for every domain action.

### BDSPro before, current and target

| Dimension | Before PR #218 | Working branch state (verify HEAD) | P01 accepted target |
|---|---|---|---|
| Router API | `Router[T]`, `NewRouter(group,getContext)` | Still generic | `Router`, `NewRouter()`, one registration convention |
| Handler conversion | `(r *Router[T]).handlers` to Gin | package-level HTTP provider path exists; `toGinHandler` remains | one canonical `toHandlerProvider` and registered typed context |
| Provider type | Gin handler | package-level `func(http.Handler)http.Handler` | same as Gitea, actually used for all application HTTP actions |
| Context source | `GetAPIContext(*gin.Context)` | `APIContext` published on `*http.Request`, old accessor remains | `GetAPIContext(*http.Request)`; no parallel lifecycle |
| Router grouping | Gin `RouterGroup` guards | scoped group prefix/middlewares | canonical Gitea-style scope |
| Pre/post route | Gin middleware | after-route implemented for matched routes | true `BeforeRouting` pre-match + `AfterRouting` post-match |
| Middleware chain | Gin `Next/Abort` | net/http provider adapted at one Gin boundary, legacy path still exists | single semantics: precheck, composition, no-write continuation, early termination |
| HTTP response | Gin writer | `ResponseStatusProvider`, tracked wrapper + Gin bridge | one committed-response contract, no duplicate writes |

**Important:** There is a **legitimate framework difference** between Gin and Chi at the final matching/execution point. Do not claim parity by forcing Gin to behave like Chi through an additional independent router. Prove HTTP behavior, route params, request cancellation, writer wrappers, middleware order, `Next/Abort`, and body/header semantics.

### Target request walkthrough

~~~text
HTTP ingress
  -> BeforeRouting (protocol/method/path transformations before matching)
  -> Gin route match (Gin retained as only matcher)
  -> AfterRouting / APIContexter (post-match; route params visible)
  -> verified principal + resource assignment
  -> bind/validate form
  -> typed router action (*APIContext)
  -> model (simple read) OR service (workflow)
     -> models/db.WithTx, TxQueries (SQLC/pgx)
     -> model operation and typed domain errors
  -> APIContext.JSON / APIErrorAuto
  -> ResponseStatusProvider detects committed response
  -> no duplicate handler continuation; request cleanup once
~~~

Proof of correctness requires both a source/ownership review **and** regression tests of the actual call chain.

## 5. Transactions and errors: P04/P05 contract

- Match Gitea's **context-scoped** transaction usage without replacing SQLC/pgx. BDSPro's `db.WithTx(ctx, fn)`, `db.WithTx2(ctx, fn)`, `db.TxQueries(ctx)` are working migration APIs; choose one final convention after inspecting all callers and nested transaction policies.
- A transaction-bound SQLC executor must remain the same for session lock → mutation → projection read; do not open an independent pool query in the middle.
- **Never remove** `models/auth.RequireActingSession`/commit-time checks just because a Gitea example lacks session revocation rules.
- Model owns domain errors and constraint-safe translation; service handles workflow and can wrap/translate when needed; `modules/util` owns shared error categories; `services/context.APIErrorAuto` owns safe public HTTP mapping. Raw pgx/provider error text must not leak.
- A router can call a model directly for appropriate reads; a service can call `models/db` directly for workflows. Neither requires a mechanical always-present layer.

## 6. Operational guardrails

1. At the beginning of each new task, fetch **current** `dev`, integration branch HEAD, changed files, PR status and latest CI; do not trust an old SHA.
2. Read the matching Gitea file **and its consumers**; compare package path, receiver, imports, arguments, return, state ownership, behavior, error and cleanup.
3. Write the smallest cohesive change that removes an actual architectural discrepancy. Do not introduce code whose only purpose is to carry a Gitea-shaped name.
4. Add/retain focused tests for actual behavior. Do not delete failing tests to make a PR green.
5. Report before/after snippets, exact file paths, commit SHA, workflow URL and which targets **remain unsatisfied**.
6. Do not merge into `dev` while architecture or full CI is incomplete. No ungrounded claims of successful tests, local execution or merging.

See [IMPLEMENTATION_STATUS.md](./IMPLEMENTATION_STATUS.md) for detailed phased delivery and [CONTINUE_PROMPT.md](./CONTINUE_PROMPT.md) for a self-contained prompt to resume in a new chat.
