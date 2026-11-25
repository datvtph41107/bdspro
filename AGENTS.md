# BDSPro — Engineering and AI working agreement

Effective 2026-11-25. This policy replaces Gitea-alignment as the development objective. It does not change the running application or approve a technical redesign.

## Read first
- [Engineering charter](docs/architecture/engineering-charter.md): business context, capacity and decision principles.
- [Current phase](docs/architecture/research-governance/CURRENT_PHASE.md): exactly one active problem.
- [Research protocol](docs/architecture/research-protocol.md) and [evidence standard](docs/architecture/research-governance/REQUIREMENTS_AND_EVIDENCE.md).
- [Decision register](docs/architecture/decisions/README.md), [handoff](docs/architecture/research-governance/SESSION_HANDOFF.md).
- For an architecture task, load [.agents/skills/architecture-research/SKILL.md](.agents/skills/architecture-research/SKILL.md).

## Non-negotiable working rules
1. Start with the enterprise workflow, actor, business outcome, constraints and acceptance criteria. Ask for missing decisions; do not infer requirements from a schema, old implementation, or another project. A requirement for security/correctness may justify preventive design without a production incident.
2. Establish actual BDSPro baseline (branch, SHA, scope, callers, dependencies, tests and deployment). Distinguish business facts, code facts, observed friction, risks, hypotheses and unknowns.
3. Learn the Go/stack mechanism first. Compare the smallest sufficient option (including no change) against alternatives. Gitea, Anthdm and other sources are optional evidence only; neither their code style nor a matching folder tree is an acceptance criterion. No mandatory DI or global state, router wrapper, manager, layer, queue or service.
4. Assess team learning time, implementation, test/debug, deploy, monitoring, incident recovery and recurring maintenance costs. Record assumptions and measurements separately; no invented numbers or incident history.
5. One bounded problem at a time. No automatic expansion, parallel architecture campaigns, or resuming old plans. A research result is not authorization to implement. Owner approves scope/behavior and design before nontrivial production changes.
6. Keep changes reviewable: problem ID, requirement, before/after code, rationale, alternative, risk, test evidence, rollback and follow-up. Use focused commits and a reviewable PR for significant changes; no automatic merge or force push. Confirm target ref. Do not create process overhead for trivial edits.
7. Preserve data, security, public behavior and existing contracts unless a scoped change is accepted. Verify resource lifecycle and failure paths. Never log secrets or claim production readiness from build/smoke alone.
8. AI is a research/coding assistant; the backend owner must understand and be able to debug the code. Teach actual execution, include minimal reproducible examples where useful, and disclose blockers. Never invent evidence, approval, results or history.

## History and authority
Existing code proves current behavior only where inspected. Existing ADRs record historical decisions, not universal rules. Previous Gitea alignment, F40, G02, R/P lanes and parallel playbooks are historical; they are not active work or authorization. Preserve them, link to them when relevant, and supersede rather than erase. Do not silently revert an accepted implementation. The active scope comes from the current owner request and phase page, not an old prompt.

States: OPEN (unknown), PROPOSED (choice awaiting approval), ACCEPTED (owner decision), IMPLEMENTED (code at SHA), VERIFIED (specified checks at that SHA), SUPERSEDED (new decision). Verification is scoped, not blanket production approval.
