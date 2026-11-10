# Cross-Lane File-Lease Registry

**Purpose:** write down unavoidable exceptions to sole file ownership. Every entry must have path(s), owning lane, requestor, reason, approved reviewer, affected commit range, allowed window, tests and close-out status. No silent exemptions.

## Existing migration overlap — PR #218

PR: https://github.com/datvtph41107/bdspro/pull/218

BDSPro's unfinished P01 feature branch **already** changes paths beyond prospective Chat A ownership. These files predate the new parallel-work plan and are a **reconciliation debt**, not automatically approved scope for future A changes:

| Path | Planned owner | Current reason for overlap | Disposition |
|---|---|---|---|
| `routers/api/v1/api.go` | D after gates / coordinator integration | P01 route signature call-site changes | Freeze D code changes until A's contract accepted; coordinator review |
| `routers/api/v1/org/org_validation_test.go` | D / integration | typed-context test fixture | source review + exact fixture justification |
| `routers/api/v1/organization_context_test.go` | D / integration | request context and middleware integration tests | source review + preserve on final integration |
| `routers/init.go` | coordinator (0) | HTTP ingress/bootstrap setup | coordinator to review or rehome final wiring |
| `services/user/{user,update,settings}.go` | B after data gate | partial transaction API migration | **B must not edit services/user in wave 1**; coordinator reconciles PR #218 before B obtains this lease |

The coordinator must decide whether to keep these as approved one-time migration exceptions, split them into subsequent PRs, or rebase/reconcile against fresh `dev`. An existing change is not automatically a reason to let A own that directory permanently.

## APPROVED — A-P01.7-API-20261110 (Chat 0 architecture decision)

- **Decision:** APPROVED by **Chat 0 — Architecture Integrator**, 2026-11-10. This is a narrowly scoped **one-time cutover lease**, not permanent A ownership of `routers/api/v1/**`.
- **File:** `routers/api/v1/api.go` **only**. Owner of the file remains **D / Chat 0 integration**; temporary writer is **Chat A**. No other `routers/**` file, fixture, or `services/user/**` is included.
- **Verified starting point:** PR [#218](https://github.com/datvtph41107/bdspro/pull/218), A branch `refactor/gitea-web-ownership-20261110`, **HEAD `33bb3c685dce8bd26b7860400c715e7ec0fa770f`**, workflow [42788259804](https://github.com/datvtph41107/bdspro/actions/runs/42788259804) **completed/success**. Approval is for successor commits on this same branch from this exact ancestor; record final SHA for closeout. Do not treat old CI as certifying successor HEADs.
- **Allowed edits:** convert `web.Router[*appctx.APIContext]` consumers to non-generic `*web.Router`; change `Routes(root *gin.Engine, d Dependencies)` to `Routes(root *gin.Engine, d Dependencies) *web.Router`; call `web.NewRouter(root)` exactly once; declare `m.BeforeRouting(...)` **before any `Methods`/route registration** if ingress HTTP providers are required; preserve `m.AfterRouting(appctx.APIContexter())` sequencing and all existing endpoint/group/middleware order. Update private `register...Routes` parameter types only. Do not introduce other Router instances or HTTP matchers.
- **Frozen shared constructor / ingress contract:** `web.NewRouter(engine *gin.Engine) *web.Router`; `(*web.Router).WrapIngress() http.Handler` **must wrap its very same captured Gin engine**, exactly once; `(*web.Router).BeforeRouting(...any)`, `AfterRouting(...any)`, `Group(string,func(),...any)`, `Methods(string,string,...any)` and verb helpers. `apiv1.Routes(root *gin.Engine,d apiv1.Dependencies) *web.Router` returns the single registered Router; no additional Engine allocation. `Server.Handler()` will serve `common.CORS(router.WrapIngress(),allowedOrigins)`, owned by Chat 0 in `routers/{init,app}.go`. Preserve exact existing routes/HTTP status/handlers, APIContext identity, JWT live-session authority, OAuth cookies/redirect/CSRF, IPActive, CORS, trusted proxies, 404/405, HEAD and cancellation. Gin matching is the only matcher. This is a **target contract**, not a claim that source currently compiles to this API.
- **Forbidden:** Auth/OAuth or organization business changes; route additions/removals/renames; change to path/method inventory, public error envelopes, security guards, CORS/trusted proxy policies; edits under `routers/init.go`, `routers/app.go`, `services/user/**`, `models/**`, `cmd/**`; creating temporary router/adapters or weakening tests.
- **Integration ownership:** Chat 0 writes `routers/init.go` and `routers/app.go` exclusively on an integration branch *stacked on A*, and Chat A does not touch those files after this grant. Chat 0 will reconcile their historical diff and run the integrated final-HEAD test suite, independently review G1–G6 and decide ACCEPTED/REJECTED. **No merge to dev before gates.**
- **Exit/expiry:** lease ends at P01.7/P01.8 + P02 regression handoff, final HEAD CI success, route inventory and cross-owner diff reviewed, or explicit revocation by Chat 0; only one cutover window. A must include lease ID and starting/final SHA in PR #218 handoff; further edits to this file require renewed permission. A's production branch must incorporate the coordinator wiring before final acceptance, without a competing writer.
- **Gate G2/G5:** **APPROVED for cutover scope**; **not** architectural acceptance. Consumer and wiring remain blocked until code+test+CI demonstrate the frozen contract.

| Lease ID | Exact path | Requester / writer | File owner | Start SHA | Approver | Status |
|---|---|---|---|---|---|---|
| `A-P01.7-API-20261110` | `routers/api/v1/api.go` | A | D / 0 | `33bb3c68` | Chat 0 | **APPROVED / OPEN — P01.7/P01.8 cutover only** |

### Historical PR #218 disposition, independent of the new lease

| Historical changed path(s) | Decision | Gate |
|---|---|---|
| `routers/api/v1/api.go` | **KEEP as reviewed migration exception**, and apply lease above for **future cutover** | HTTP route inventory, Auth/OAuth, registration order |
| `routers/api/v1/org/org_validation_test.go`, `routers/api/v1/organization_context_test.go` | **KEEP as integration/regression fixtures**, preserve tests; not a standing write lease | Migrate fixture signatures via approved narrowly scoped separate lease or Chat 0, not silently in A |
| `routers/init.go` | **MOVE final production ownership to Chat 0**, preserve removal of duplicate APIContexter from global Gin chain if route-owned | Chat 0 integration wiring/test |
| `services/user/{user,update,settings}.go` | **REHOME to Chat 0/B transaction reconciliation**, NOT approved as new A writes. Existing change moves to context-first `WithTx2(txCtx)` while keeping `RequireActingSession`; review with B-P04 transaction contract before merge | No blind acceptance of historical A diff |
| `docs/architecture/gitea-alignment/**` | **KEEP as A evidence**, not a substitute for coordinator canonical records | Verify source links and accurate final HEAD |
| `modules/web/**`, `services/context/**` | **KEEP A ownership**, contingent on removing generic/legacy code and passing final tests | G1–G6 |

## P04-B-SERVICES-20261110 — REQUEST REVIEWED / NOT APPROVED

- **Requester:** Chat B, PR [#221](https://github.com/datvtph41107/bdspro/pull/221), starting source HEAD `d73e3ecada31898a59c625431dd4ce890c099348`.
- **Requested paths:** `services/user/update.go`, `services/user/settings.go` (with affected `services/user/user.go`), and retirement of `models/db/transaction.go::WithTxResult` once all consumers migrate.
- **Decision by Chat 0:** **NO NEW LEASE to Chat B while A #218 still carries its historical modifications to the exact services/user files.** Granting it now would recreate the G2 conflict under the guise of solving G5. This is an explicit *deferred* lease decision, not silence or a refusal of the transaction correctness objective.
- **Owner next:** Chat 0 reconciles those three files from #218 against the final B transaction API on an integration branch; Chat B reviews the diff and may request a **new exact-file lease only if there remains uncovered model/consumer work**. Until then B may only change its owned `models/**`.
- **Worker outbox:** `services/worker/outbox_dispatcher.go` remains **Chat 0 integration** responsibility for a separately reviewed worker transaction conversion, not a B/C implicit lease. Must preserve at-least-once outbox claims, error retry and rollback.
- **Closeout:** current decision expires upon accepted combined A/B transaction integration, at which point all caller references to `WithTxResult` are inventoried before Chat B removes legacy API. See [CHAT0_TO_CHATB_HANDOFF.md](./CHAT0_TO_CHATB_HANDOFF.md).
- **Status:** **REQUESTED → DEFERRED / NO WRITE PERMISSION**. This is a G2 safety hold, not an authorization to merge PR #221.

## APPROVED / CLOSED — A-P01.7-FIXTURES-20261110

- **Decision:** Chat 0 retrospectively reviewed and **APPROVED, CLOSED** the exact-fixture migration needed by the frozen non-generic Router, **without granting an ongoing API ownership lease**. The five files are `routers/api/v1/organization_context_test.go`, `routers/api/v1/org/org_validation_test.go`, `routers/api/v1/org/team_test.go`, `routers/api/v1/org/team_invite_test.go`, `routers/api/v1/user/me_test.go`. Owner remains D / Chat 0; Chat A is the one-time writer of already committed fixture changes.
- **Starting ref:** `33bb3c685dce8bd26b7860400c715e7ec0fa770f`; **accepted fixture cutover A HEAD:** `46dfc9e4d5b0c0dacd3ca47359ed90d88d32842f`. Commit `e013c1430289dd6d373cd800db6c4dcc016f1a88` replaces obsolete generic `web.NewRouter[*APIContext](group,getContext)` calls with `web.NewRouter(engine)` in each file, retains test assertions and routes; `user/me_test.go` moves existing auth/session Gin middleware from group to engine so the same matching/guard test still executes. The `organization_context_test.go` APIContext registration is moved onto the Gin engine before constructor; no new temporary Router or production security policy is created.
- **Evidence:** exact A-head [CI 42797385297](https://github.com/datvtph41107/bdspro/actions/runs/42797385297) **COMPLETED SUCCESS**; Chat 0 directly compared the five files on start/head, including constructor and middleware changes, not just the A report.
- **Scope guard:** only these five test fixtures; **no new production ownership for A**. This acceptance does not cover `routers/init.go`, `routers/app.go` (coordinator), service transaction consumers, or future fixture changes. All other historical #218 files keep their existing reconciliation dispositions. New changes require another approved lease.
- **Integration:** Chat 0 integrated A `46dfc9e4` with coordinator wiring on stacked [PR #222](https://github.com/datvtph41107/bdspro/pull/222) at merge commit `b1cb51c47f4f0dc1c1b19a808f560fa1bf6d0f32`, same matcher; integrated G4/G6 remain pending exact-head workflow and smoke security review.

| Lease ID | Exact paths | Writer | Owner | Starting SHA | End SHA | Decision |
|---|---|---|---|---|---|---|
| `A-P01.7-FIXTURES-20261110` | five API fixture tests listed above | Chat A | Chat 0 / D | `33bb3c68` | `46dfc9e4` | **APPROVED / CLOSED (one-time audited exception)** |

## New leases

Use the following template for each request (no entries approved at plan creation):

| Lease ID | Paths | Requester | Owner | Base/head SHA | Why and impacted imports | Approved by | Window/gate | Status |
|---|---|---|---|---|---|---|---|---|
| (add) | | | | | | | | REQUESTED |

Only Chat 0 marks a lease APPROVED after identifying contract and rollback behavior. Worker must report lease in PR body. Close lease after final CI and merge; another agent must not assume a prior lease still authorizes new paths.

## How to use the path guard

`bash tools/parallel/check-owned-paths.sh B origin/dev` rejects unauthorized paths. The checker is deliberately strict and does not read this ledger. A historical PR #218 will have path failures until coordinator reconciliation. **Do not weaken the script to accept everything under an unrelated lane just to green a CI run.** Approved exceptions require a separate explicit review, not an implicit bypass.
