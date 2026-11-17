---
name: bdspro-architecture-research
description: Evidence-first Go monolith architecture investigation and decision workflow for BDSPro × Gitea; use before proposing core, runtime, package or subsystem changes.
---

# BDSPro × Gitea — Architecture Research Skill

Repository-local skill for coding assistants. **Not automatically installed or executed by ChatGPT**; agent tooling should load this document explicitly when the task concerns architecture.

## Triggers
Use when analyzing or changing CLI/bootstrap, process/worker/graceful, resource initialization/ownership, package boundaries, dependency wiring, router/context/core, model/service composition, monitoring/error policies, or Gitea-alignment decisions.

## Mandatory inputs
Read repository [AGENTS.md](../../../AGENTS.md), [engineering charter](../../../docs/architecture/engineering-charter.md), [investigation protocol](../../../docs/architecture/research-protocol.md), [source map](../../../docs/architecture/source-evidence-map.md), [G02 agenda](../../../docs/architecture/g02-runtime-agenda.md), [ADR register](../../../docs/architecture/decisions/README.md). Refresh actual GitHub refs; do not trust stale chat claims.

## AI execution loop
1. **Frame** one technical design question before touching code. State invariant, scope, non-goals, and where user/business requirements differ from speculative production conditions.
2. **Retrieve** the real Gitea source pinned by SHA, including caller, callee and consumer. Look at official Go/dependency docs when the mechanism requires semantics outside the repo.
3. **Trace** construction, init, activation, serving, shutdown; import graph vs runtime dependency; state, error boundaries, ownership and observable effect.
4. **Challenge** the Gitea mechanism: evidence for behavior vs inferred intent; limitations, hidden coupling, integration assumptions and alternative approaches.
5. **Compare** BDSPro v4 as scale/history and BDSPro dev as untrusted implementation under review. Record gaps rather than normalizing current source as correct.
6. **Design** at least two real code-shape alternatives where the choice is nontrivial, state their direct/import/runtime change surfaces, learning/test/operational costs and a counterexample.
7. **Decide conditionally**: OPEN, PROPOSED, ACCEPTED, etc. No “keep” purely because tests pass or a file already exists. Explicitly identify approval and missing evidence.
8. **Implement only after scope** and then assess actual diff, final-HEAD tests/CI and behavior. Never pretend local tests/incident data exist.
9. **Handoff** source permalinks, exact commits, findings, unresolved questions, next investigation and draft ADR status.

## Required report format
```text
Problem / engineering force:
Invariant:
Observed Gitea code (SHA/path/lines, caller → callee → consumer):
External authoritative source (when needed):
Observed BDSPro CRM v4 / current dev (separate from design validity):
Design interpretation and uncertainty:
Alternative A vs B (code shape + ownership + costs):
Counterexample / failure to satisfy:
Evidence measured vs not measured:
Decision status and remaining research:
```

## Anti-patterns for agent reasoning
- Inferring architecture legitimacy from passing tests, prior commits, history, or user-defined code structure.
- “Gitea does X so we must do X.”
- Designing disaster scenarios merely to justify an invented framework.
- Conflating business `handler → service` layering with the G02 runtime research question.
- Treating `mustInit` as a universal fail-fast doctrine or matching function names as a contract.
- Copying source snippets inaccurately, guessing SHAs/line numbers, claiming unrun tests or invented production savings.
- Editing live code while an essential architectural boundary remains OPEN without explicit implementation authorization.

## Completion gate
A research task is done when another engineer can reproduce the source trace, identify assumptions, critique alternatives and understand costs. An implementation task is done only with actual code and verification at final HEAD. These are **different** completion states.
