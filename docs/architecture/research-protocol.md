# Architecture Investigation Protocol — problem → evidence → decision

**Status:** workflow specification. It governs analysis, not runtime behavior.

## Investigation sequence

**0. State the scope.** Which technical problem? Which process and execution path? Which of construction, initialization, serving, background work, shutdown, deploy, or domain changes? Avoid expanding unrelated layers.

**1. Define the real problem independently of an implementation.** What makes this difficult in a long-lived backend? What breaks if no contract exists? Is it a necessary system invariant or a speculative convenience? Record assumptions and unknowns.

**2. Study actual source.** Fetch exact commit SHA, path and line ranges; trace entrypoint → caller → callee → consumers. Examine import direction, APIs/signatures, state, error branches, cleanup, shared-resource ownership and concurrency. Code snippets labeled REAL must reproduce actual code; label ellipses and illustrations.

**3. Seek intent only with evidence.** Source may prove behavior but not design rationale. Upstream comments, linked issues, PRs, docs or commit history may support intent. Otherwise mark motive as inference.

**4. Consider design alternatives.** Minimal direct calls, package-level ownership, explicit dependency injection, compile-time wiring, generic orchestration, or dedicated runtime controller only when genuinely relevant. Never list alternatives without comparing effects and costs.

**5. Use an adversarial case.** What if dependencies are initialized in a different order? What if a consumer outlives a resource? What if a new business capability is added? What does this design make harder? Distinguish a possible failure from a currently demonstrated defect.

**6. Evaluate change and operational cost.** Measure or statically count real imports/locations/call paths for representative changes. Identify what future measurement would validate improvement. Do not fabricate traffic, incident frequency, CI results or benchmark numbers.

**7. Compare BDSPro last.** Inspect live `dev` and CRM v4 after establishing the independent design requirements. Mark existing BDSPro behavior as **observed**, **suspect**, **incomplete**, or **not inspected**, never automatically correct. Preserve business invariants while questioning architectural mechanisms.

**8. Record decision state.** `OPEN`, `PROPOSED`, `ACCEPTED`, `REJECTED`, `SUPERSEDED`. A recommendation is not acceptance. If evidence is missing, record an explicit research task rather than an unsupported decision.

**9. Implement only after scoped decision.** Diff contracts, caller/consumer impact, migration and rollback, behavioral compatibility, validations; test against the final commit and disclose unverified conditions.

## Required investigation record

```text
Question / context:
Operating or development constraint:
Invariant and boundary:
Source A (commit/path/lines/caller/callee):
Source B and contrasting mechanism:
What source proves vs what is inferred:
Design A / effects / complexity costs:
Design B / effects / complexity costs:
Counterexample / failure of preferred design:
BDSPro code audit (observed, not authoritative):
Decision state + why:
Next evidence needed / measurement / implementation gate:
```

## Strict separation of kinds of proof

| Kind | Can establish | Cannot establish alone |
| --- | --- | --- |
| Source tracing | Exact control flow and interfaces in inspected files | Production incident rates or author's intentions |
| Design reasoning | Logical invariants and consequences under assumptions | That a particular organization actually suffered a failure |
| Controlled test | Observed behavior under tested inputs/environment | Architectural necessity or production SLO achievement |
| Operations data | Observed runtime cost and failure frequency | Universal correctness in untested conditions |

## Reference and contradiction handling

Pin Gitea and BDSPro commit SHAs; compare dated docs against current source. If documentation disagrees with source, cite both and mark the inconsistency. A previous chat answer or AI skill is not authority. No invented source lines, commit output or measurements.

For Go-specific correctness use official Go and package docs as a baseline; for deployment/configuration refer to the relevant platform docs as warranted. The user has not authorized wholesale framework migration simply because another project chooses different dependencies.

## Deliverable and handoff

Write the research question, evidence table, options/costs, provisional conclusion or explicit OPEN status, acceptance threshold, next owner/action. When implementation eventually occurs, link exact before/after commits and final-HEAD CI. Avoid claiming work happened asynchronously.

## Current context and evidence extension (2026-11-20)

The single-phase, cross-chat implementation of this protocol is [research-governance/README.md](research-governance/README.md), with [CURRENT_PHASE](research-governance/CURRENT_PHASE.md), [REQUIREMENTS_AND_EVIDENCE](research-governance/REQUIREMENTS_AND_EVIDENCE.md), [EVIDENCE_LEDGER](research-governance/EVIDENCE_LEDGER.md) and [SESSION_HANDOFF](research-governance/SESSION_HANDOFF.md).

Additional mandatory checks for new architecture target discussions:

1. No folder/tree, package, interface, control or code pattern is a target decision without actual consumer/import graph and impact analysis across Web, CLI, workers, DB, Redis, Auth/security, deployment and observability; mark unavailable facts NOT INSPECTED rather than generalizing from one route.
2. Where appropriate, compare **Gitea and Laravel DRX** at pinned revisions and explain Go vs Laravel framework semantics; establish governing constraints first from Go/net/http and relevant official standards.
3. Analyze the mechanism: exact code, construction vs execution time, caller → callee → consumer, trust boundary, state/concurrency, errors/response and cleanup; separate verified facts, inferred rationale, synthetic model and actual tests.
4. Keep exactly one user-selected phase active. Historical docs/PRs/previous chat narratives never authorize work in another phase. Architecture recommendation remains PROPOSED until explicit acceptance; no production code edits during the current RG-00 governance-only phase.
