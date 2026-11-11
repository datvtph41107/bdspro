# Workstream Handoff / PR Acceptance Template

Copy into each worker's draft PR. **Fill every field with evidence or explicitly mark NOT VERIFIED.** Coordinator alone marks ACCEPTED.

## Identity
- Lane + phase/subtask:
- Owning chat's task file:
- BDSPro repository and base branch:
- Base SHA:
- Head SHA:
- PR URL:
- Gitea source reference file(s) and pinned commit SHA:
- Files changed (literal list from GitHub PR diff):

## Hypothesis & counterexample
- Concrete inconsistency in BDSPro source BEFORE:
- Gitea source evidence and semantic reason:
- Smallest consistent change in BDSPro AFTER:
- Expected testable outcome:
- Adversarial test/failure input that could disprove the claimed outcome:

## Contract and ownership
- Approved paths:
- Paths outside normal ownership:
- RFC/file-lease reference if applicable (otherwise none):
- Exported signature/import changes and downstream consumers:
- Public API/DB/Auth/Session/runtime invariants preserved:
- Dependencies on other lanes and handoff blockers:

## Verification (final HEAD only)
- Formatting and static checks (exact command/result):
- Focused tests (exact command and evidence):
- Full `go test ./...` or CI job (link/conclusion; if running, mark pending):
- SQLC/schema/OpenAPI/route inventory where relevant:
- Runtime smoke/redis/shutdown or integration results where relevant:
- Performance/recovery metrics with baseline and units (or not measured):
- Last CI workflow URL and failed steps:
- Regression/rework log:

## Integration
- Readiness: DRAFT / READY FOR REVIEW / ACCEPTED / BLOCKED:
- Coordinator's independent review and counterexample result:
- Merge prerequisites:
- Post-merge verification:
- Rollback approach and special handling for data migrations:
- Remaining explicitly not implemented:
- Next task/owner:
