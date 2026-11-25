# BDSPro engineering charter — requirement-led evolution
Effective 2026-11-25. Owner direction: reset the engineering approach, not the runtime. See [RESET-001](decisions/ADR-RESET-001-requirement-led-evolution.md).

## Context and capacity
Owner-reported facts: startup with two developers (one frontend, one backend); REST API experience, limited Go/production-risk/testing experience; business requirements not fully clarified; no observed traffic/incident baseline supplied. Earlier backend was organized as approximately six or seven services plus gateway, RPC and protobuf before documented needs or decision evidence were available to the owner. These are context statements, not a comprehensive historical audit or a claim of zero security defects. Validate legacy behavior and business rules from relevant sources and stakeholders.

A single backend developer bears learning, implementation, review, migration, deployment, debugging and incident response. Every persistent technical mechanism must be supportable by that capacity. The goal is a useful business product, with correct, understandable, maintainable Go code and a traceable development history.

## How architecture evolves
Business need → validated workflow/invariants → stack constraints and quality scenarios → simplest adequate implementation → use/test/operate → observed friction or new requirement → targeted decision → focused change → verification and feedback.
Design proactively for credible security, data integrity, privacy and costly irreversible risks; do not wait for harm. Avoid speculative optimization. Evolution may remove mechanisms as well as add them.

## Evidence hierarchy by question
Business intent: owner/stakeholder and agreed examples, not source shape. Current behavior: pinned BDSPro source and reproduced tests. Mechanism: versioned Go, HTTP, PostgreSQL, library contracts. Cost and incidents: actual observations and measurements, or labeled estimates and simulations. Alternatives: relevant upstream implementations, history/PRs and engineering literature. Gitea and Anthdm are examples, never target architecture or required comparators.

## Economics and ownership
Account for learning, code changes, test fixtures, diagnostic burden, deployments, rollback, dependency upgrades, and recurring operational work. Consider the cost of doing nothing, cost of a wrong decision and reversibility. A stack choice must satisfy the actual use case and fit the team's support capability. No claim of savings without evidence.
Keep security and data guarantees independent of architectural style. The owner decides scope, acceptable tradeoffs and readiness. AI provides analysis, alternatives and evidence; it cannot substitute for understanding or business acceptance.

## Historical boundary
Past decisions and experiments remain available in Git and existing docs. Do not treat lack of documented rationale as proof the code is bad. Do not invent a historical migration story or retrofit a justification. Mark unknowns; assess the current behavior before proposing removal.

## Dual outcome: product value and professional competence
The owner intends BDSPro to be the longitudinal record of real Go backend engineering, from requirement discovery and coding to release and production operation. The product is not a disposable tutorial or a technology showcase. Develop competence by working on genuine product needs and credible preventive risks; keep optional experiments outside production scope. A checked-in feature proves implementation, not production experience; deployment and operational claims need dated evidence. Follow [Engineering lifecycle](engineering-lifecycle.md) and preserve decision history, actual feedback and lessons without sensitive customer data.
