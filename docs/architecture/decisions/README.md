# BDSPro Architecture Decisions — register and template

**Status:** governance for future decisions. This directory does **not** establish that today's BDSPro architecture is accepted.

## Status vocabulary

- **OPEN:** problem and constraints not yet sufficiently understood.
- **PROPOSED:** competing alternatives assessed, awaiting explicit acceptance.
- **ACCEPTED:** decision owner accepted a documented design with evidence and trade-offs.
- **REJECTED:** investigated and declined, with reason.
- **SUPERSEDED:** newer accepted ADR explicitly replaces it.

Do not silently convert a historical note or passing test into an accepted ADR. Never edit previous decisions to erase their context; supersede.

## Research register (all OPEN as of 2026-11-17)

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
