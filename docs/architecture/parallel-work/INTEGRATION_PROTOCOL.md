# Integration Protocol — Three Parallel Producers, One Merger

## 0. Relationship between branches

- Stable merge target: `dev`.
- Existing unfinished HTTP branch: `refactor/gitea-web-ownership-20261110` (draft PR #218). **Chat A owns only this branch** until P01/P02 acceptance; nobody else pushes to it.
- Chat B branch: `work/gitea-b-data-20261110` based on current `dev`.
- Chat C branch: `work/gitea-c-runtime-20261110` based on current `dev`.
- Chat D (later): `work/gitea-d-api-auth-20261110`, created only when cross-lane gates are satisfied.
- Coordinator docs branch: `docs/gitea-parallel-harness-20261110` → docs-only PR targeting `dev`.

The **same repo does not mean the same working branch**. Never have concurrent writers on one branch or unreviewed writes to `dev`. Every chat must fetch current SHA before edit and avoid overwriting concurrent commits.

## 1. Dependency graph

`P01 HTTP routing ↔ P02 HTTP context` is tightly coupled → **one owner A**.

`P04 data transaction → P05 selected models/services` is sequential inside B. The interface exported by P04 is consumed by P03.

`P06 runtime modules` is mostly independent of A/B **until bootstrap wiring** (`routers/init.go`), which coordinator owns.

`P03 API/Auth/OAuth` can audit and write contract tests now but source refactor must await A's accepted Router/APIContext contract and B's accepted transaction/actor contract.

`P07` is continuous independent verification and a **final integrated gate**, not an independent source-writing agent.

## 2. Delivery waves

| Wave | A (HTTP) | B (data) | C (runtime) | D (API) | Integrator |
|---|---|---|---|---|---|
| W0: contract | inventory, freeze APIContext/route requirements | freeze current DB Tx/actor shape | enumerate init imports/startup | read-only route/Auth inventory | approve ownership & docs PR |
| W1: parallel | finish P01/P02 on PR #218 | implement scoped P04; P05 model-only | implement P06 module pieces without `routers/init.go` | design P03 contract/test plan (no prod source edits) | watch 3 PRs and ownership violations |
| W2: gates | provide accepted Router/APIContext contract | provide accepted DB/session interface | provide accepted constructors/cleanup | build P03 on fresh dev incorporating accepted contracts | coordinate ordered merge and integration smoke |
| W3: full | regression support only | regression support only | bootstrap integration with coordinator | complete Auth/OAuth routes and security tests | P07 whole-repo acceptance / merge |

A workstream may proceed to a self-contained subtask without waiting if it does not need another lane's unstable interface. **Do not fake independence** by adding temporary production adapters.

## 3. Proposed merge ordering

The exact order follows **real contract dependencies**, not phase numbers:
1. Merge docs/harness once reviewed.
2. Merge B's minimal isolated P04 DB contract *if it does not alter consumed signatures*; otherwise await integration review.
3. Merge C's isolated, actually-used modules if no bootstrap source collision or changed semantics.
4. Merge A's P01/P02 once its own CI, Auth/OAuth/route stability and source ownership pass.
5. Merge any B/C follow-on changes requiring A's new context; coordinator handles `routers/init.go`.
6. Unblock D and merge P03 only after A/B contracts are verified.
7. Run P07 whole-repo validation and publish acceptance evidence.

A change that touches public symbols must declare its merge prerequisites and tests. Never assume rebasing onto `dev` without revalidation is harmless.

## 4. PR rules

- **One worker → one isolated PR per coherent task** against `dev`. All PRs draft until their final HEAD is checked.
- PR title: `[LANE/Pxx] Verb: scoped outcome`.
- Required body: base/head SHA, paths, Gitea comparison, API/DB contracts affected, tests and actual CI run links, rollback notes, open blockers, [handoff template](./HANDOFF_TEMPLATE.md).
- A worker may commit to its own branch, but **only coordinator or an explicitly authorized integrator merges** accepted PRs. No auto-merge to `dev` under the current trust policy.
- All changes to coordinator-owned files: create a change request and handoff, do not write them on worker branches.
- Update each lane's handoff in its own file/PR; **do not** have all chats edit the same status document.

## 5. Integration gates

**Gate H (HTTP):** non-generic Router when target calls for it; typed action one-path; pre/post routing behavior; context one-request-one-APIContext; response and Gin bridge behavior; no API/Auth/OAuth regression; full HEAD green.

**Gate D (DB):** no stale session authorization; nested rollback-only semantics; SQLC-generated separation; transaction-bound executor; race and error-path tests; full HEAD green.

**Gate R (runtime):** init ordering, config/log/queue/cache failure contracts; graceful/worker behavior; no dependency violations; full HEAD green.

**Gate API:** Gate H + Gate D contract availability, Auth/AuthZ/OAuth tests and e2e, endpoint inventory, cookies/redirects/CSRF, secure public errors; full HEAD green.

**Gate FINAL (P07):** all integrated in current `dev`, clean tree, complete CI and runtime/benchmark/security review, updated baseline and migration report.

## 6. Deadlock, priority, and rollback

When two tasks both require a public file: pause that file, keep independent work going, submit an RFC to the integrator; avoid parallel hot-fixes. If CI fails, the responsible PR fixes it; don't hide the failure by changing another lane's code. If a merged PR regresses `dev`, coordinator makes an explicit revert decision and records why (DB migrations need special recovery; do not blindly reverse migrated schema).

## 7. Starting three sessions

Open 3 separate conversations and paste the **START NEW CHAT** prompt at the bottom of each of the respective task files. This is user-driven parallelism. ChatGPT conversations do not automatically share context, stream one another's progress or schedule background code work. GitHub is the handoff bus; each session independently fetches source/branch/CI and commits only to its assigned branch.

When worker tasks complete, paste the PR links to Chat 0 or invoke a new coordination session with the Chat 0 prompt. That session verifies repo state rather than trusting a message alone.
