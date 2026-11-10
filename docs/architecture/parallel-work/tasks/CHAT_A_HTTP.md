# Chat A — HTTP Router + Request Context (P01 and P02)

**Role:** exclusive HTTP-layer implementer and source owner.

**Assigned branch:** `refactor/gitea-web-ownership-20261110`; draft PR [#218](https://github.com/datvtph41107/bdspro/pull/218). **Never merge this PR to dev without coordinator acceptance.** Do not create a duplicate A branch unless the integrator explicitly replaces PR #218.

**Why P01 and P02 together?** Gitea's router registers `*APIContext` handler arguments, and the response/continuation contract is implemented through `services/context`. Separating these into independent write sessions would create duplicate context providers and frequent incompatible intermediate commits. Own the boundary, not two disconnected work items.

## Write scope
- **Own:** `modules/web/**`, `services/context/**`, relevant tests under those paths.
- **Read:** `routers/api/v1/**`, `routers/init.go`, `models/db/**`, `services/auth/**`, Gitea's matching source.
- **Requires explicit temporary lease:** `routers/api/v1/**` and other route files needed to migrate call sites or tests. Record exact paths and approval in PR.
- **Never write:** DB models/SQLC, Auth/OAuth business flows, runtime process/modules owned by B/C.

## First inspection (before any new patch)
1. Read shared docs in [../README.md](../README.md), [../OWNERSHIP_AND_CONTRACTS.md](../OWNERSHIP_AND_CONTRACTS.md), [../TRUST_AND_VERIFICATION.md](../TRUST_AND_VERIFICATION.md), [../INTEGRATION_PROTOCOL.md](../INTEGRATION_PROTOCOL.md).
2. Inspect **live** PR #218 HEAD + CI. The last historical snapshot is *not* guaranteed to be green. Resolve red CI before refactor.
3. Read PR branch `docs/architecture/gitea-alignment/{README,IMPLEMENTATION_STATUS,CONTINUE_PROMPT,SKILL}.md`.
4. Compare Gitea `modules/web/{router,handler}.go`, `modules/web/types/premiddleware.go`, `services/context/{base,api}.go`, `routers/api/v1/api.go`.
5. Inspect BDSPro exact counterpart files, callers, tests, and actual Gin behavior.

## Execution milestones
- A1: stabilize P01.4–P01.6 existing implementation; demonstrate correct after-routing + pre-provider ordering **including** bootstrap context, no response double-write, and clean final HEAD CI.
- A2: establish one request-local APIContext / reqctx cleanup contract, no hidden global mapper; keep current Auth/OAuth/session/public response behavior.
- A3: non-generic `web.Router` and `NewRouter()`; one handler conversion and registered typed provider. Keep Gin matcher. Remove `Router[T]`, `r.toGinHandler`, duplicate code only after usages migrated.
- A4: true `BeforeRouting` executes ahead of Gin matching, not just a matched-route middleware; `AfterRouting` can use route params. Prove not-found, HEAD/multi-method, http.ResponseWriter wrappers, cancellation and middleware continuation.
- A5: phase acceptance P01/P02 on the same branch. Publish APIContext contract to B and D; only then authorize D to change API routes.

## Required tests
- Typed provider registration and unsupported signatures rejected at startup.
- Middleware order (pre-after/group/route, then normal after/group/route).
- One APIContext per request; `ctx.CurrentRequest()` has current auth/DB/scope context; no leaked request-local state between requests.
- Selected vs committed HTTP status, abort short-circuit, JSON/error privacy.
- Gin handler compatibility only where still justified; OAuth cookie/redirect/CSRF and session/verified actor preserved.
- Full Go 1.27.3 CI, route inventory, integration/runtime smoke and performance/allocations comparison if implementation adds material overhead.

## Done and handoff
Produce exact GitHub source comparison and complete changed-file list; an accepted final HEAD CI run; documented unchanged contracts; independent failure-mode review; [handoff](../HANDOFF_TEMPLATE.md) in PR body. If blocked by need to edit `routers/api/v1/**`, open RFC to Chat 0 rather than editing without lease.

## START NEW CHAT — copy the following message

> Bạn đảm nhiệm **Chat A — P01+P02: HTTP Router/Context**, chuyên trách BDSPro theo Gitea. Repository: https://github.com/datvtph41107/bdspro ; Gitea: https://github.com/go-gitea/gitea ; branch: `refactor/gitea-web-ownership-20261110` ; draft PR #218. Đầu tiên đọc đầy đủ thư mục `docs/architecture/parallel-work/` trên nhánh `docs/gitea-parallel-harness-20261110`, đặc biệt `README.md`, `OWNERSHIP_AND_CONTRACTS.md`, `TRUST_AND_VERIFICATION.md`, `INTEGRATION_PROTOCOL.md`, `SKILL.md`, và file task `tasks/CHAT_A_HTTP.md`. Sau đó đọc bộ `docs/architecture/gitea-alignment/` trên nhánh PR #218. Kiểm tra HEAD/CI mới nhất, khắc phục CI đỏ, rồi triển khai P01/P02 theo Gitea với **source code thật**, không thay Gin/SQLC. Bạn là owner độc quyền `modules/web/**` và `services/context/**`; không sửa files thuộc Chat B/C/D hoặc `routers/api/v1/**` nếu chưa có explicit file lease từ Chat 0. Mỗi commit cần Gitea before/after, call graph, tests, CI, blockers và cập nhật handoff trên PR. Không merge vào dev khi chưa đủ nghiệm thu.
