---
name: bdspro-architecture-research
description: Requirement-led evidence and incremental Go engineering for BDSPro; use for nontrivial design or runtime changes.
---
# BDSPro requirement-led engineering skill
Read [AGENTS](../../../AGENTS.md), [charter](../../../docs/architecture/engineering-charter.md), [current phase](../../../docs/architecture/research-governance/CURRENT_PHASE.md) and [protocol](../../../docs/architecture/research-protocol.md). This is a repository-local guide, not an automatically installed tool.

1. Clarify real business goal, owner, invariant, acceptance example, constraints and unknowns.
2. Refresh branch/SHA. Trace relevant source and test paths; find historical evidence without inventing rationale. Separate source facts from owner reports and hypotheses.
3. Teach the underlying Go/stack mechanism using official contracts, runnable example where useful, and actual output only when executed. Use Gitea/Anthdm or another project only if relevant to the same problem.
4. Compare no-change, simplest adequate change and other viable design; analyze learning, development, testing, operations, security, migration, rollback and cost. No design pattern is a requirement.
5. Record OPEN/PROPOSED/ACCEPTED; only implement approved scope. One phase at a time. No auto-resume, broad refactor, or automatic merge.
6. Show actual diff and before/after code, targeted verification at final SHA, limitations and handoff. Distinguish lab, integration, CI, production observation. Owner must be able to understand and operate the result.

Report: requirement → baseline → problem/risk → mechanism → alternatives → costs → decision/status → change/verification → next observation. For small changes keep documentation proportional.

## Lifecycle learning and upstream archaeology
Read [Engineering lifecycle](../../../docs/architecture/engineering-lifecycle.md). For a named decision, inspect baseline, trigger and before/after changes; find relevant PRs/issues and tests, differentiate actual upstream history from a plausible local experiment. Compare options on the same operation, including code consumption, error paths, maintenance and operational impact. If upstream rationale is unavailable, say so. Record what the owner needs to understand, a small reproducible exercise if helpful, actual delivery/operation evidence and the signal for reconsideration. Never add production technology solely for practice. No claim of deployment, incident response or mastery without actual evidence.
