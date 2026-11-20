# BDSPro — Architecture Research Governance

**State:** RESEARCH PROTOCOL, not an accepted target architecture. **Established:** 2026-11-20.
**One active phase:** [CURRENT_PHASE.md](CURRENT_PHASE.md). **Resume here:** [SESSION_HANDOFF.md](SESSION_HANDOFF.md).

## Mission and constraints

Build a long-lived, operable Go business monolith whose technical core can be understood, modified, tested, deployed and debugged as domains and team size grow. We are researching *how to derive the requirements for the core architecture*, not choosing a folder tree or adding middleware because a reference project uses it.

A design must respond to a demonstrated engineering force and its scope: correctness, developer comprehension, change surface, security/trust boundary, deployment, lifecycle, concurrency, diagnosability, performance and operating cost. A proposed need is not automatically a production fact. Existing BDSPro code, tests, docs and ChatGPT answers are **audit evidence**, never automatic design authority.

## Authority and sources

1. **User requirements / explicitly approved constraints:** define goals and permission to change; prior sessions or historical plans cannot silently override latest scope.
2. **Language and protocol contracts:** Go specification, standard library (net/http, context, sync), HTTP RFCs, relevant official DB/Redis/Chi docs, OWASP for threat framing. Confirm versions and actual applicability.
3. **Current BDSPro source at an exact commit:** authoritative for *implemented behavior*, not automatically for design correctness.
4. **Gitea source at a pinned commit:** architectural comparison for Go route composition, context, modules, lifecycle and operations; verify its callers/consumers and rationale independently.
5. **Laravel DRX at a pinned commit:** contrasting mature business application showing global/group/route middleware, HTTP Kernel, named policies and security operations; Laravel/PHP framework services are not Go design requirements.
6. **Other implementations (Chi, Caddy, Go stdlib and real systems) only for a named open problem:** distinguish official semantics, open-source mechanism and unmeasured production claims.
7. **Tests, CI, runtime traces and performance/security measurements:** prove only specified scenarios at the pinned code/env; never invent output.
8. **Historical docs, sketches, old PRs and conversation memory:** navigation and hypotheses, NOT live source or acceptance.

When sources disagree, create a contradiction record with versions, behavior and implications; do not harmonize or silently prefer the better-sounding story. Source establishes behavior; a PR discussion / issue / upstream docs might establish intent, but inferred intent must be labeled INFERENCE.

## Architecture research pipeline

`Requirement / observed friction → engineering force → boundary + invariant → source call graphs → source contrast → failure/adversarial cases → alternatives (including doing less) → impact on the rest of BDSPro → cost and verification plan → explicit decision status → scoped implementation authorization → tests at final HEAD`.

**No folders, interfaces, managers, middleware order or architecture diagram may be presented as accepted target merely from one example route, one failure case or visual elegance.** Require the full caller/consumer graph, cross-subsystem consequences and falsification criteria first.

A task stays in **one active phase** until its completion gate is independently checked. Related topics and proposals remain parked OPEN. Separate a research phase from a later implementation phase: a documented proposal does not authorize code edits.

## Cross-system requirement model

For each proposed core capability ask how it behaves across: Web API, CLI/migrations, workers/jobs, process startup/shutdown, PostgreSQL/Redis, public API contracts, AuthN/AuthZ/security, deployment (binary Linux and Docker), logging/diagnosis, test isolation, package imports, concurrent requests, resource limits and future feature expansion. Mark **N/A with a reason**, or **NOT INSPECTED**; do not invent dependencies. See [REQUIREMENTS_AND_EVIDENCE.md](REQUIREMENTS_AND_EVIDENCE.md).

## Source-based deliverables

- [CURRENT_PHASE.md](CURRENT_PHASE.md): one active scope, explicit exit gate and parked work.
- [REQUIREMENTS_AND_EVIDENCE.md](REQUIREMENTS_AND_EVIDENCE.md): claim/evidence method, scenarios and architecture decision tests.
- [EVIDENCE_LEDGER.md](EVIDENCE_LEDGER.md): pinned observed facts, contradictory material and unverified claims.
- [DESIGN_CASE_TEMPLATE.md](DESIGN_CASE_TEMPLATE.md): reproduce one engineering question without broad unsupported verdicts.
- [SESSION_HANDOFF.md](SESSION_HANDOFF.md): minimal durable context across chats; refresh live refs before reuse.
- Existing [research protocol](../research-protocol.md), [engineering charter](../engineering-charter.md) and [ADR register](../decisions/README.md) remain applicable.

## Implementation policy

No code mutation in this governance phase. For later explicitly approved edits: confirm branch/HEAD/permissions, obtain a scoped changed-file list, write characterization tests, prepare exact migration/revert strategy, keep existing HTTP paths/status/response and security semantics unless separately approved, and verify changed code at final HEAD. Prefer smallest cohesive change over broad rewrite. Reports must say what was **not** measured, executed or validated.
