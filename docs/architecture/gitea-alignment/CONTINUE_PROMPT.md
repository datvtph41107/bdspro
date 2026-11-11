# Copy/Paste Prompt — Resume BDSPro × Gitea Refactor

> This is a **self-contained execution prompt for a new ChatGPT conversation**, not an instruction to assume historical chat assertions are always current. Read the listed GitHub files first. Do not mark any phase complete without live source and CI evidence.

## Prompt for new session

~~~text
Bạn tiếp quản dự án BDSPro từ checkpoint kiến trúc Gitea-alignment, ưu tiên triển khai source thực tế và báo cáo bằng chứng, không chỉ phân tích.

REPOSITORIES / BRANCHES
- BDSPro: https://github.com/datvtph41107/bdspro
- Primary baseline: dev
- Integration branch: refactor/gitea-web-ownership-20261110
- Draft PR: https://github.com/datvtph41107/bdspro/pull/218
- Reference architecture/source: https://github.com/go-gitea/gitea, branch main
- DO NOT assume the branch HEAD is the historical SHA in this prompt. Verify latest branch SHA, PR diff and GitHub Actions before editing. Other contributors may have committed changes.
- Source instructions in:
  docs/architecture/gitea-alignment/README.md
  docs/architecture/gitea-alignment/IMPLEMENTATION_STATUS.md
  docs/architecture/gitea-alignment/CONTINUE_PROMPT.md

HARD CONSTRAINTS
- BDSPro must KEEP Go 1.27.3; Gin v1.12; SQLC-generated queries/pgx; PostgreSQL 18; Redis; Asynq; existing Authentication, Authorization, OAuth2/OIDC semantics and REST API. DO NOT migrate to Chi or XORM and DO NOT introduce gRPC.
- Match Gitea in SOURCE ORGANIZATION, code/function/receiver patterns, import and call graph, single responsibility per package/file, dependency init/lifecycle, HTTP handling, context, domain/model/service and transaction error propagation.
- Never claim equivalence merely from function names or directory trees. Compare Gitea actual source and callers: package ownership -> imports -> signatures -> call order -> request/transaction lifecycle -> tests.
- Gin is allowed at the private routing boundary of modules/web, not allowed to dictate domain handler or HTTP contract. SQLC/pgx are implementation under models/db; Gitea XORM usage is used for conceptual transaction convention only.
- No duplicate handler/response, errors, transaction, or APIContext sources. No new adapters unless justified by runtime semantics and benchmark.
- Preserve verified actor/session checks; lock acting session with write transaction where required. Preserve OAuth redirects, cookies, CSRF, short-circuit and error privacy.
- Keep application-level routes and their method/path/status/request/response behavior unchanged without explicit approval.
- Merge into dev only after final P01-P07 acceptance and CI; until then work on integration branch and update draft PR. Avoid modifying unrelated PRs.
- If tools cannot execute Go 1.27.3 locally, use actual GitHub Actions job logs as evidence; never invent passing tests.

CURRENT CHECKPOINT
P01 ("modules/web" — handler/router execution) IS NOT COMPLETE. Current interim code includes:
- modules/web/handler.go: package-level middlewareProvider = func(http.Handler)http.Handler, toHandlerProvider(handler any), responseWriter, RegisterResponseStatusProvider[T], preCheckHandler, prepareHandleArgsIn, hasResponseBeenWritten. Gitea handler.go at verified blob 62fb0709.
- modules/web/router.go: still generic Router[T], NewRouter(group,getContext), scoped Group, AfterRouting, Methods (comma-separated verbs) and Head, Gin bridge/ginDelegatingWriter, compileActions and legacy toGinHandler; target must be a non-generic Router/NewRouter(), one action compiler and true BeforeRouting. Gitea router.go at verified blob 2ddfbdae.
- services/context/api.go: registers *APIContext provider in init() and GetAPIContextFromRequest; services/context/gin.go still contains ginRequestSource/APIContexter compatibility adapter. Gitea services/context/api.go at verified blob c14c3b3e.
- routers/api/v1/api.go: still uses web.Router[*APIContext] and legacy GinActions on some Auth/OAuth endpoints; must migrate consistently in P01/P02/P03.
- models/db/context.go: WithStore, StoreFromContext, TxQueries, WithTx, WithTx2. services/user update/settings changed to WithTx2 and TxQueries but the full transaction API consolidation is P04.
- No complete canonical wrapMiddlewareAndHandler, no true BeforeRouting, no fully net/http Base yet.
- A CI failure for direct org router tests without APIContexter prompted a fixture change plus a compatibility getContext call. THIS IS TRANSITIONAL. Do not cement fallback into final source without design proof.
- Previous CI regressions: missing-store error precedence; gofmt; missing APIContext from HTTP request. Tests must remain and be fixed, not deleted.
- The status document was drafted while historical code HEAD was 9579336f, but subsequent docs commits and possibly independent commits can change it. Re-fetch HEAD and CI.

REFERENCE FILES TO FETCH AND COMPARE DIRECTLY
Gitea:
- modules/web/router.go
- modules/web/handler.go
- modules/web/types/types.go or relevant ResponseStatusProvider declaration
- services/context/base.go and services/context/api.go
- routers/api/v1/api.go; routes with typed *APIContext actions
- routers/init.go, services/org/team.go, models/db/context.go as needed.
BDSPro:
- modules/web/router.go
- modules/web/handler.go, modules/web/router_test.go
- modules/web/types/response.go
- services/context/{base,api,gin}.go; services/context/response_status_test.go, api_context_test.go
- routers/{init.go,api/v1/api.go}
- routers/api/v1/org/org_validation_test.go, team*_test.go, Auth/OAuth routes
- models/db/context.go, services/user/{user,update,settings}.go
- .github/workflows/test.yml, go.mod

MANDATORY P01 IMPLEMENTATION ORDER
0. Verify GitHub live HEAD of dev and integration branch, draft PR #218 state, changed files, final HEAD GitHub Actions; resolve any red CI before more refactor.
1. Deep comparison of Gitea toHandlerProvider, wrapMiddlewareAndHandler, pre/normal middleware order, PreMiddlewareProvider, RegisterResponseStatusProvider, preCheckHandler, responseWriter and request context. Call out exact differences and behavioral implications.
2. Canonicalize modules/web/handler.go with ONE provider/argument/response pipeline: no long per-type special-case switch and no extra Gin return type. Match actual Gitea code decomposition and reject unsafe signatures at route registration.
3. Implement actual Gitea-style middleware composition, including pre and normal ordering and no double Next, preserving Gin route matching. Code must use new helpers, not merely define dead helpers.
4. Migrate Router[T] to non-generic Router, NewRouter(), BeforeRouting, AfterRouting, Group, Methods, ServeHTTP. Keep Gin only as private backend and avoid second router/matching engine.
5. Migrate all real API route declarations/guards/binders and tests to new contract in one coherent branch. Remove toGinHandler and unused compatibility public functions once no consumers.
6. Verify selected vs committed response status, wrapped ResponseWriter, path params, reqctx cleanup, route-miss, GET/HEAD/multi-method behavior, error/privacy, Auth/OAuth cookie/redirect/CSRF and session checks.
7. Run complete Go 1.27.3 CI for final HEAD. Show source diff, test and benchmark evidence. Close P01 only after all its acceptance criteria pass; P02 thereafter.

P02-P07 ROADMAP
P02: Pure net/http Base/APIContext/APIContexter, eliminate Gin-specific services/context adapter without duplicate context lifecycle.
P03: Canonical API v1 routes, AuthN/AuthZ/OAuth2 flows, context guards, error/response shape, no separate Controller/Handler layer.
P04: canonical context-scoped SQLC/pgx transactions, nested rollback-only semantics, session revocation concurrency and permission invariants.
P05: model/service package ownership, direct model reads when appropriate, service workflows, typed domain errors and util/context HTTP mapping.
P06: Gitea-style cmd, routers/init, modules setting/log/cache/queue/graceful/reqctx lifecycle with real consumers.
P07: full clean-source acceptance, CI, architecture diff, runtime/performance security checks, ready PR and merge only when fully validated.

EVIDENCE AND REPORTING IN EVERY RESPONSE
- "Current checkpoint: Pxx" + pass/fail criteria.
- Table columns Gitea (actual path/function/import), BDSPro BEFORE, BDSPro AFTER, runtime behavior, evidence.
- Quote short ACTUAL function excerpts and link Gitea source + BDSPro commit/PR; distinguish illustrative snippets from actual code.
- List files/commits changed and precise CI run ID/URL, job result, failing test names. Do not state CI green if running or failed.
- Next uncompleted stage and precise blockers. Avoid "tiếp tục" filler or vague promises.
- The output should also maintain/update docs/architecture/gitea-alignment/IMPLEMENTATION_STATUS.md at meaningful checkpoints.

IMPORTANT TECHNICAL QUESTION
Why did old BDSPro have (r *Router[T]).toHandlerProvider(any) gin.HandlerFunc while Gitea has package-level toHandlerProvider(any) middlewareProvider? The earlier implementation chose Gin as abstraction boundary, not net/http. Correct by following Gitea's FUNCTION RESPONSIBILITY and actual execution graph while retaining Gin matcher. Do not "fix" merely by changing the return type; verify all callers, wrappers, writer and continuation.

START NOW:
Fetch latest source and CI, determine precise P01 subcheckpoint and implement the next cohesive source changes. Then update docs and draft PR, report the before/after with evidence. No generic future promises; do the work in this turn and be transparent about anything not completed.
~~~

## Source and checkpoint links

- [Canonical target](./README.md)
- [Status and acceptance plan](./IMPLEMENTATION_STATUS.md)
- [Current PR #218](https://github.com/datvtph41107/bdspro/pull/218)
- [Integration code](https://github.com/datvtph41107/bdspro/tree/refactor/gitea-web-ownership-20261110)
- [Gitea router source](https://github.com/go-gitea/gitea/blob/main/modules/web/router.go)
- [Gitea handler source](https://github.com/go-gitea/gitea/blob/main/modules/web/handler.go)

## Why this file exists

A new chat must **reconstruct state from the GitHub repository**, not hallucinate completed patches from a summarized conversation. GitHub code, CI logs and explicit diff are authoritative. This prompt preserves **user intent, constraints, phase order, references, unresolved bugs, and the next work item**; it does not replace live verification.
