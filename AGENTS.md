# BDSPro — Engineering & AI Agent Entry Point

> Canonical working policy for architecture research and subsequent code changes. This document is a **research and decision protocol**, not a claim that today's BDSPro architecture is correct. Read it before changing code.
>
> Updated 2026-11-17. This documentation-only update is explicitly requested for remote `dev`; it is not acceptance of implementation or a blanket change to branch/merge protection policies.

## Start here (in this order)

1. [Engineering charter](docs/architecture/engineering-charter.md): goal, source hierarchy and non-negotiable engineering standards.
2. [Evidence-driven investigation protocol](docs/architecture/research-protocol.md): how to establish the problem **before** proposing an abstraction.
3. [Source evidence map](docs/architecture/source-evidence-map.md): pinned Gitea, BDSPro legacy CRM v4, BDSPro `dev` anchors, with scope/limitations.
4. [G02 research agenda](docs/architecture/g02-runtime-agenda.md): open questions about application composition, startup, dependency and lifecycle.
5. [Decision register and ADR template](docs/architecture/decisions/README.md): proposed vs accepted decisions and their evidence gates.
6. [AI research skill](.agents/skills/architecture-research/SKILL.md): repeatable task workflow, evidence and handoff output.

## Ground rules for all agents

- **Do not treat existing BDSPro code, historical documents, comments, tests or previously accepted PRs as architectural authority.** They are observations and candidates for critique. A passing test establishes tested behavior, not architectural validity.
- **Do not treat Gitea as infallible or copy its syntax.** Study exact code, dependencies, caller/callee, runtime semantics, historical trade-offs and relevant upstream documentation. State where evidence stops.
- Begin with the technical problem, invariant or design force (runtime correctness, package ownership, evolution/operational cost). Show alternatives and counterexamples **before** `NÊN/KHÔNG NÊN/CHƯA NÊN`.
- Separate **observed code facts**, **reasoned mechanisms**, **hypotheses** and **unverified production claims**. Never invent benchmarks, incident reports, outputs, SHAs, links or line numbers.
- Keep responsibility, compile-time imports, runtime dependency, resource ownership, activity lifetime, errors and observability distinct. Do not invent managers, registries, layers, interfaces or generic frameworks merely for symmetry.
- If the current architecture is inconsistent with a justified target, report the inconsistency; do not preserve it just because it exists. Conversely, do not rewrite code before architecture questions are answered.
- For implementation work: determine the current branch/HEAD; record affected call graph, contracts, migration and compatibility constraints; inspect and preserve real user-visible behavior unless a change is approved; review tests and CI at **final** HEAD.
- Never use `log.Fatal`, retry, rollback, DI, global state, graceful shutdown, etc. as self-justifying pattern names. Explain their failure boundary, effect, owner and cost.
- Source of truth for implementation is the actual source at the commit being reviewed, not this documentation. Historical descriptions may be stale.
- No secret leakage, unsafe destructive migrations, silent credential logging or unapproved changes to business/security semantics.

## Explicitly distinguish three work states

- **Research / OPEN:** competing models and open questions; no implementation authorization.
- **Decision / ACCEPTED:** evidence-backed ADR with trade-offs, owner and approval/acceptance recorded.
- **Implementation / VERIFIED:** code at pinned SHA, passing appropriate tests/CI and documented limitations.

An ADR's existence does not make it accepted. A test pass does not prove a design choice was necessary.

## Existing material and coordination

- Existing [Gitea-alignment playbook](docs/architecture/gitea-alignment/README.md), [G02 historical implementation note](docs/architecture/g02-application-bootstrap.md), [parallel-work coordination](docs/architecture/parallel-work/README.md), and other architecture notes are **historical evidence / workstream records**, not substitutes for fresh verification. Do not silently destroy them.
- Earlier lane rules and PR procedures may remain relevant to an active task; check latest GitHub state and explicit user instructions. Do not infer permission to edit code or merge from this file.
- Preferred working branch for currently requested BDSPro work: `dev` unless the user changes it. Before code writes, respect current repository protections and ask for approval only if a required decision/permission is truly missing.

## Required answer for an architecture task

`Problem → system forces → source evidence (Gitea + other sound sources) → actual mechanism/call graph → alternatives + costs → counterexample → BDSPro gap (not presumed correct) → open/decision status → implementation and verification gates`.

The point is **durable technical judgment**, not alignment theater or premature refactoring.

## Active migration program F40 — Event, Queue, Realtime, Audit (2026-11-18)

- **Requested target:** replace Asynq and PostgreSQL `job_outbox` with a Gitea-inspired queue/worker architecture; align typed notify, pub/sub, websocket, message, subscriber and audit with evidenced Gitea patterns. This is an **implementation direction**, not an assertion the replacement has been implemented or migration approved.
- **Read first:** [F40 foundation and Gitea source evidence](docs/architecture/f40-gitea-subsystem-foundation.md) and [F40 sequential task board](docs/architecture/f40-task-board.md). Follow the gates, one task at a time; mark VERIFIED only with actual source/test evidence and final HEAD.
- **Critical:** no drop of `job_outbox` while producers/consumers or queued rows remain. Preserve security email recovery, existing audit transaction invariants, user authorization and domain semantics. Do not replace Asynq with Redis Pub/Sub, or confuse notifications with queue jobs.
- Current `cache-queue.md`, `runtime.md`, old ADRs and migration files are pre-cutover baseline evidence until source and tests show actual replacement; don't silently rewrite history.
