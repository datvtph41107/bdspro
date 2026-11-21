# BDSPro Architecture Decisions — register and template

**Status:** governance for future decisions. This directory does **not** establish that today's BDSPro architecture is accepted.

## Status vocabulary

- **OPEN:** problem and constraints not yet sufficiently understood.
- **PROPOSED:** competing alternatives assessed, awaiting explicit acceptance.
- **ACCEPTED:** decision owner accepted a documented design with evidence and trade-offs.
- **REJECTED:** investigated and declined, with reason.
- **SUPERSEDED:** newer accepted ADR explicitly replaces it.

Do not silently convert a historical note or passing test into an accepted ADR. Never edit previous decisions to erase their context; supersede.

## Accepted decisions after 2026-11-17

- [ADR-G02-02 — Canonical process settings and directory owners](ADR-G02-02-canonical-process-settings.md):
  ACCEPTED at user request 2026-11-21. Single-process Gitea-like package
  settings, pure preflight, one-time publication, resource-owner Close,
  and explicit primitive constructors. Implementation must still pass CI
  at the final refactor SHA; acceptance of the architecture does not
  imply production verification of every failure mode.

## Historical research register (all OPEN as of 2026-11-17)

| ID | Question | Required comparison |
| --- | --- | --- |
| G02-01 | Where is the Web/Worker application composition root and what is its contract? | Gitea `runWeb/InitWebInstalled/NormalRoutes` vs old CRM v4 `InitializeApp` vs current BDSPro |
| G02-02 | How are subsystem dependency and readiness represented without hidden temporal coupling? | Explicit calls vs registries/graph vs package state |
| G02-03 | Who owns shared resources and their lifecycle across process modes? | Package owner, injected instances, cleanup model |
| G02-04 | Which work is object construction vs subsystem init vs long-running activity? | Queue/worker/cron/process paths and actual consumers |
| G02-05 | Where do error semantics, observability and failure decisions belong? | Caller/callee policies, diagnostics and operational effects |
| G02-06 | What is the change-surface and maintenance cost when adding a CRM-sized domain? | File/import/caller graph for representative feature and subsystem |

See [G02 agenda](../g02-runtime-agenda.md) and [research protocol](../research-protocol.md). Existing [G02 implementation note](../g02-application-bootstrap.md) is historical implementation evidence, **not** acceptance of G02-01..06.

## ADR template

```markdown
# ADR-XXXX: <specific decision title>
Status: OPEN | PROPOSED | ACCEPTED | REJECTED | SUPERSEDED
Date / owner / reviewed commit(s):

## Context and constraints
## Fundamental engineering problem (not today's implementation)
## Invariants and scope of guarantee
## Source evidence: pinned links, caller/callee, observed behavior
## Reasoned inference vs unresolved facts
## Alternatives (including simpler baseline)
## Complexity/cost model and change-surface impact
## Counterexample / negative case / security or operational trade-off
## Decision and explicit rationale (only when ACCEPTED)
## Consequences and migration/compatibility
## Verification and measurement gates
## Revisit triggers / references
```

## Acceptance gate

No design verdict without evidence, alternative, ownership and cost. No implementation approval until it is clear what behavior changes, which interfaces/imports are affected, how to retain user-visible behavior/security, and what needs tests. Avoid universal numerical claims unsupported by measurements.

## Current research-first acceptance extension (2026-11-20)

Before writing an ADR proposal, complete the [system requirements and evidence gate](../research-governance/REQUIREMENTS_AND_EVIDENCE.md) and a [single design case](../research-governance/DESIGN_CASE_TEMPLATE.md) for the [one active phase](../research-governance/CURRENT_PHASE.md). Full producer/consumer/call graph, cross-subsystem import and operational impact, a minimal baseline, Gitea/Laravel DRX comparison when relevant, failure/security tests, counterexample and measurable costs are mandatory. A package tree, earlier assistant recommendation, draft code demo or passing test is never an accepted architecture. The user must explicitly accept a proposed ADR before implementation authorization.
