# Live Coordination Board — BDSPro × Gitea

> Snapshot 2026-11-10, Chat 0 coordinator. **Not an automatically updating dashboard.** Every session must refresh current GitHub HEAD, PRs and Actions before editing. Update this board only through a coordinator-owned reviewed change.

## Workstream table

| Lane | Scope / phase | Working branch | PR | Current decision and next executable task |
|---|---|---|---|---|
| **A** | P01/P02 HTTP+request context | `refactor/gitea-web-ownership-20261110` | [#218](https://github.com/datvtph41107/bdspro/pull/218) (draft) | **CI PASS at `d2dcef51`**: [42775942108](https://github.com/datvtph41107/bdspro/actions/runs/42775942108). **P01/P02 not accepted**: `Router[T]`, `getContext`, `legacyActions`, separate Gin `AfterRouting` remain; true outer `BeforeRouting` and shared-file migration lease pending. Historic PR also edits `routers/**`, `services/user/**` beyond A ownership. |
| **B** | P04 DB / P05 scoped models | `work/gitea-b-data-20261110` | [#221](https://github.com/datvtph41107/bdspro/pull/221) (draft) | **CI PASS** at `868d10f8`: [42776794000](https://github.com/datvtph41107/bdspro/actions/runs/42776794000). **13 model-only files** changed (not 7), no cross-lane file overlap. Review explicit `WithTxResult` → `WithTx` reverse nesting, `ExecTxRaw` SQLC boundary and verified actor/session cases. P05 not complete. |
| **C** | P06 process/shared runtime | `work/gitea-c-runtime-20261110` | [#220](https://github.com/datvtph41107/bdspro/pull/220) (draft) | **CI PASS** at `553551e3`: [42776266066](https://github.com/datvtph41107/bdspro/actions/runs/42776266066). 26 paths within C ownership; removed unused ChannelTaskQueue after consumer search. **P06 not accepted**: preexisting `modules/cache` lacks a production consumer, external app.ini templates and cache-queue docs remain stale, init/graceful integration contracts still open. |
| **D** | P03 API/Auth/OAuth | not created | none yet | **READ-ONLY REVIEW**; build endpoint/contract/test inventory; wait for accepted A HTTP context and B DB transaction interfaces before source changes. |
| **0** | contracts/integration P07 | `docs/gitea-parallel-harness-20261110` | [#219](https://github.com/datvtph41107/bdspro/pull/219) (draft) | **ACTIVE**: accept docs/guardrail PR after current-HEAD CI; reconcile historical PR #218 cross-owner files; approve isolated B/C PRs after evidence; no dev merge without gates. |

## Actual CI evidence at this snapshot

- P01 code PR #218, prior HEAD `c97c480e`: [run 42758853017](https://github.com/datvtph41107/bdspro/actions/runs/42758853017) failed **Verify Go formatting** in `routers/api/v1/organization_context_test.go`; corrected by commit [`ad1f0138`](https://github.com/datvtph41107/bdspro/commit/ad1f01385ac9db3032070c04b2f12d85be43fe11).
- P01 follow-up HEAD `ad1f0138`: [run 42759268838](https://github.com/datvtph41107/bdspro/actions/runs/42759268838) **FAILED Test and compile**, specifically: `modules/web.TestRegisteredTypedRouterAcceptsPreMiddlewareInGroupAndMethods` panic `interface conversion: interface {} is nil, not *web.typedHTTPContext` at `modules/web/router_test.go:573`, invoked from `Router.Methods.func2` at `router.go:322`; and `routers/api/v1.TestAfterRoutingHTTPPreAndTypedAPIContextShareRequestLifecycle` reports HTTP/typed APIContext divergence at `organization_context_test.go:170`. **Do not fix by deleting tests, introducing another global context registry, or masking panic.**
- Docs PR #219 `2b312cd2`: [run 42762359493](https://github.com/datvtph41107/bdspro/actions/runs/42762359493) **success**. Later coordinator edits created new HEADs, so that run **does not** certify the latest PR.
- Ownership script correction `f631ad54`: [run 42763221965](https://github.com/datvtph41107/bdspro/actions/runs/42763221965) was **in progress** when last checked. This board commit will create an even newer HEAD needing validation.
- Current observed `dev` SHA `8dea6dc9` matches the HEADs of both B/C starter branches (verified by GitHub commit-ref fetch). No source PRs were opened by Chat 0; independence requires B/C to stay within their write sets. Refresh before each action.

## 2026-11-10 cross-workstream review — Chat 0

- **Validated GitHub current refs:** `dev@8dea6dc9`, A `d2dcef51` (112 commits ahead), B `868d10f8` (33 ahead), C `553551e3` (12 ahead); all 0 behind at check. Confirm fresh heads before further work.
- **GitHub diffs:** A 21 paths (includes historical `routers/**` and `services/user/**` cross-owner debt); B 13 `models/**` paths; C 26 `cmd/**`/approved runtime module paths. Three current implementation branches **do not edit any identical paths**. This is a path isolation observation, not integration correctness.
- **A:** final-head CI success; P01 target incomplete and historical cross-owner lease not approved. Hold merge; A must write a narrowly scoped P01.7 RFC (Router API, callsites, outer `BeforeRouting`), allow coordinator-owned wiring and ensure Auth/OAuth/session tests.
- **B:** current-head CI completed SUCCESS during this review; `WithTx`/`WithTx2` reuses SQLC executor and prevents nested raw commit, but explicit standalone transaction callback `func(*Queries)` does not expose `txCtx`; a context-first nested call using the original context can still open an independent transaction. This limitation is documented in `models/db/transaction.go`. Review RFC B-P04-01 before gate D; no silent global transaction adapter. Additional `models/auth` and `models/organization` changes must be reviewed as security/domain scope.
- **C:** final-head CI success, removal of in-memory channel queue has no non-test constructor consumer in the searched `dev` source. Runtime `modules/cache.NewStringCache` also has no production consumer in that search; do not claim complete P06. Config/examples and cache-queue docs require coordinator-owned update or approved lease; no placeholder cache consumer or empty graceful package.
- **Verdict:** A `CI VERIFIED / P01 BLOCKED`, B `CI VERIFIED / P04 BLOCKED`, C `CI VERIFIED / P06 BLOCKED`; **no source PR ACCEPTED / INTEGRATED**. P03 remains audit-only pending A+B contract gate. See [#218](https://github.com/datvtph41107/bdspro/pull/218), [#221](https://github.com/datvtph41107/bdspro/pull/221), [#220](https://github.com/datvtph41107/bdspro/pull/220).

## 2026-11-10 Chat 0 decision — P01.7 lease and coordinator wiring

- **A exact observed HEAD:** `33bb3c685dce8bd26b7860400c715e7ec0fa770f`; [CI 42788259804](https://github.com/datvtph41107/bdspro/actions/runs/42788259804) **SUCCESS**. This supersedes the earlier A `d2dcef51` snapshot.
- **Lease APPROVED:** `A-P01.7-API-20261110` for Chat A to modify **only `routers/api/v1/api.go`** during the one-time non-generic Router consumer cutover. See [LEASE_REGISTRY.md](./LEASE_REGISTRY.md). A must not edit `routers/init.go`, `routers/app.go`, `services/user/**` or other API fixtures without a separate lease.
- **G5 frozen contract:** `web.NewRouter(*gin.Engine) *web.Router`, `(*web.Router).WrapIngress() http.Handler` on same engine, `apiv1.Routes(*gin.Engine,Dependencies) *web.Router`. One Gin matcher and one APIContext/request; BeforeRouting executes before Gin matcher, AfterRouting after matched path. Details in [OWNERSHIP_AND_CONTRACTS.md](./OWNERSHIP_AND_CONTRACTS.md).
- **Coordinator implementation:** stacked draft [PR #222](https://github.com/datvtph41107/bdspro/pull/222), branch `integration/p01-router-wiring-20261110` from A `33bb3c68`. Chat 0 changed `routers/init.go`, `routers/app.go`, added `routers/app_ingress_test.go`; `Server.Handler()` now serves `CORS(apiRouter.WrapIngress())` rather than bypassing HTTP middleware. Last observed HEAD `99c89a01`; its workflow [42792201172](https://github.com/datvtph41107/bdspro/actions/runs/42792201172) is **in progress**, and the branch is intentionally **not buildable on the old generic Router API** until A's contract migration. Do not merge or claim verified.
- **Historical #218 disposition:** keep API context/validation tests as migration regressions (coordinate fixture signatures); rehome `routers/init.go` final wiring to Chat 0; reconcile `services/user/{user,update,settings}.go` as DB transaction consumer changes with Chat B. Do not assume historical cross-owner paths are permitted future writes.
- **Remaining gates:** A P01.7/P01.8 + P02 tests on final HEAD, coordinator refresh #222 from final A source, Go 1.27.3 CI, CORS/proxy/404/405/HEAD/Auth/OAuth/session/CSRF/cancellation/smoke, G1–G6 independent review. **APPROVED lease is not ACCEPTED architecture.** No dev merge.

## 2026-11-10 Chat 0 conditional acceptance: P04-B / P06-C

- **Chat B PR #221:** verified `d73e3ecada31898a59c625431dd4ce890c099348`, 20 `models/**` files, CI [42790012175](https://github.com/datvtph41107/bdspro/actions/runs/42790012175) SUCCESS. **CONDITIONAL ACCEPT scoped P04 models code; integrated G5 BLOCKED**, G6 independent scoped review performed. `WithTxResult` still has out-of-lane callers; raw OAuth refresh replay is commit-then-error. **No B lease** for `services/user/**` because historical A #218 owns overlapping diff. Chat 0 handles user/worker consumers. P05 remains CLOSED. See [B handoff](./CHAT0_TO_CHATB_HANDOFF.md), [lease disposition](./LEASE_REGISTRY.md), [PR221 comment](https://github.com/datvtph41107/bdspro/pull/221#issuecomment-6764137583).
- **Chat C PR #220:** verified `07287479864a33304c243913d5d99d29f1ba48c7`, 41 paths within C scope, CI [42779188456](https://github.com/datvtph41107/bdspro/actions/runs/42779188456) SUCCESS. **CONDITIONAL ACCEPT scoped P06 runtime source; freeze changes**, docs/templates/go.mod and combined P07 remain Chat 0. No unused cache/channel/graceful module resurrection. See [C handoff](./CHAT0_TO_CHATC_HANDOFF.md) and [PR220 comment](https://github.com/datvtph41107/bdspro/pull/220#issuecomment-6764138330).
- **Integration ordering:** A non-generic router + Chat0 stacked PR #222; cross-owner `services/user/**` reconciliation and worker outbox transaction migration; B/C integration in isolated branch with final-head A/B/C P07 tests; only then consider merges to `dev`. A/B/C CI green independently does NOT prove combined HEAD passes. Chat D remains read-only until A+B contracts accepted in combined source.

## Guard script independent verification

`tools/parallel/check-owned-paths.sh` at corrected `f631ad54` was manually syntax-checked with `bash -n` and behavior-tested against a **temporary local Git repository** using the matching script text:
1. B changing `models/db/context.go`: accepted (exit 0).
2. A changing a model: rejected (exit 1).
3. B deleting `modules/web/router.go`: rejected (exit 1).
4. B moving a coordinator-owned `docs/architecture/parallel-work/README.md` into an owned `models/db/copied.md`: rejected (exit 1).

This verifies the shell rule mechanics on a fixture, **not** the BDSPro repository contents or semantically correct source changes; changes uncommitted or untracked are not covered. Re-run in actual worker branches after fetching this script or after #219 merges. CI enforcement/branch protection not yet configured.

## Priority of the coordinator now

**W0 — Freeze and handoff**: do not allow more than one writer for the same file. A owns HTTP P01/P02 and must resolve its red CI without asking B/C to patch the route test. B starts models-only; C starts runtime-only. D can audit read-only.

**W1 — Independent verification**: for each new PR confirm exact base/head SHA, allowed paths, Gitea source links, tests and current CI. Workers must not report another lane's progress without GitHub evidence.

**W2 — Dependency acceptance**: A publishes stable Router/APIContext; B publishes stable transaction/acting-session; C publishes constructors/lifecycle; coordinator handles `routers/init.go` and shared build changes. Only after these are accepted does D get permission to implement P03.

**W3 — P07**: integration tests, API/OAuth/session regression, SQLC/schema, Go 1.27.3, runtime recovery, route inventory and performance before merge.

## Hand-off protocol

Each worker's next deliverable is a **draft PR**, not a chat promise. Use [HANDOFF_TEMPLATE.md](./HANDOFF_TEMPLATE.md). Update this board only after comparing PR source and Actions against actual GitHub. When an owner needs an unowned file, request an explicit lease in [LEASE_REGISTRY.md](./LEASE_REGISTRY.md).

Related: [README](./README.md) · [OWNERSHIP_AND_CONTRACTS](./OWNERSHIP_AND_CONTRACTS.md) · [INTEGRATION_PROTOCOL](./INTEGRATION_PROTOCOL.md) · [SKILL](./SKILL.md).
