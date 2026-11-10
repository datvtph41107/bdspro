# Live Coordination Board — BDSPro × Gitea

> Snapshot 2026-11-10, Chat 0 coordinator. **Not an automatically updating dashboard.** Every session must refresh current GitHub HEAD, PRs and Actions before editing. Update this board only through a coordinator-owned reviewed change.

## Workstream table

| Lane | Scope / phase | Working branch | PR | Current decision and next executable task |
|---|---|---|---|---|
| **A** | P01/P02 HTTP+request context | `refactor/gitea-web-ownership-20261110` | [#218](https://github.com/datvtph41107/bdspro/pull/218) (draft) | **BLOCKED — actual test failures** at current observed branch HEAD `ad1f0138`, CI run 42759268838: `TestRegisteredTypedRouterAcceptsPreMiddlewareInGroupAndMethods` panics (typed context not published before `getContext`), and `TestAfterRoutingHTTPPreAndTypedAPIContextShareRequestLifecycle` fails (HTTP/typed request context divergence). A should repair **context creation/order**, not remove or weaken these tests; final HEAD CI must pass before further refactor. |
| **B** | P04 DB / P05 scoped models | `work/gitea-b-data-20261110` | none yet | **READY TO START** from `dev`; wave-1 write scope only `models/**`. First task: Tx API / nested rollback / actor-session authority inventory, then isolated PR. `services/user/**` frozen pending #218 reconciliation. |
| **C** | P06 process/shared runtime | `work/gitea-c-runtime-20261110` | none yet | **READY TO START** from `dev`; write only `cmd/**` and approved `modules/setting/log/cache/queue/graceful/**`; no `routers/init.go`. First task: Gitea-vs-BDSPro imports and startup lifecycle inventory. |
| **D** | P03 API/Auth/OAuth | not created | none yet | **READ-ONLY REVIEW**; build endpoint/contract/test inventory; wait for accepted A HTTP context and B DB transaction interfaces before source changes. |
| **0** | contracts/integration P07 | `docs/gitea-parallel-harness-20261110` | [#219](https://github.com/datvtph41107/bdspro/pull/219) (draft) | **ACTIVE**: accept docs/guardrail PR after current-HEAD CI; reconcile historical PR #218 cross-owner files; approve isolated B/C PRs after evidence; no dev merge without gates. |

## Actual CI evidence at this snapshot

- P01 code PR #218, prior HEAD `c97c480e`: [run 42758853017](https://github.com/datvtph41107/bdspro/actions/runs/42758853017) failed **Verify Go formatting** in `routers/api/v1/organization_context_test.go`; corrected by commit [`ad1f0138`](https://github.com/datvtph41107/bdspro/commit/ad1f01385ac9db3032070c04b2f12d85be43fe11).
- P01 follow-up HEAD `ad1f0138`: [run 42759268838](https://github.com/datvtph41107/bdspro/actions/runs/42759268838) **FAILED Test and compile**, specifically: `modules/web.TestRegisteredTypedRouterAcceptsPreMiddlewareInGroupAndMethods` panic `interface conversion: interface {} is nil, not *web.typedHTTPContext` at `modules/web/router_test.go:573`, invoked from `Router.Methods.func2` at `router.go:322`; and `routers/api/v1.TestAfterRoutingHTTPPreAndTypedAPIContextShareRequestLifecycle` reports HTTP/typed APIContext divergence at `organization_context_test.go:170`. **Do not fix by deleting tests, introducing another global context registry, or masking panic.**
- Docs PR #219 `2b312cd2`: [run 42762359493](https://github.com/datvtph41107/bdspro/actions/runs/42762359493) **success**. Later coordinator edits created new HEADs, so that run **does not** certify the latest PR.
- Ownership script correction `f631ad54`: [run 42763221965](https://github.com/datvtph41107/bdspro/actions/runs/42763221965) was **in progress** when last checked. This board commit will create an even newer HEAD needing validation.
- Current observed `dev` SHA `8dea6dc9` matches the HEADs of both B/C starter branches (verified by GitHub commit-ref fetch). No source PRs were opened by Chat 0; independence requires B/C to stay within their write sets. Refresh before each action.

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
