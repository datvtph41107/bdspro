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

## New leases

Use the following template for each request (no entries approved at plan creation):

| Lease ID | Paths | Requester | Owner | Base/head SHA | Why and impacted imports | Approved by | Window/gate | Status |
|---|---|---|---|---|---|---|---|---|
| (add) | | | | | | | | REQUESTED |

Only Chat 0 marks a lease APPROVED after identifying contract and rollback behavior. Worker must report lease in PR body. Close lease after final CI and merge; another agent must not assume a prior lease still authorizes new paths.

## How to use the path guard

`bash tools/parallel/check-owned-paths.sh B origin/dev` rejects unauthorized paths. The checker is deliberately strict and does not read this ledger. A historical PR #218 will have path failures until coordinator reconciliation. **Do not weaken the script to accept everything under an unrelated lane just to green a CI run.** Approved exceptions require a separate explicit review, not an implicit bypass.
