# BDSPro — Engineering & AI Agent Entry Point

> Canonical working policy for architecture research and subsequent code changes. This document is a **research and decision protocol**, not a claim that today's BDSPro architecture is correct. Read it before changing code.
>
> Updated 2026-11-17. This documentation-only update is explicitly requested for remote `dev`; it is not acceptance of implementation or a blanket change to branch/merge protection policies.


## Current implementation focus — Canonical Process Settings & Ownership (2026-11-21)

**Current accepted decision:** [ADR-G02-02](docs/architecture/decisions/ADR-G02-02-canonical-process-settings.md).
**Canonical execution and package ownership:** [Canonical Process Settings & Source Ownership](docs/architecture/canonical-settings-and-ownership.md).

BDSPro has one process configuration authority in `modules/setting`:
`LoadWebConfig`/`LoadMigrateConfig` are pure and serve static verification;
`LoadWebSettings`/`LoadMigrateSettings` publish once. Process-owned
DB/Redis/Auth/Graceful initializers read `setting` globals directly
(Gitea pattern). `db.Connect(ctx, source)`, `RedisManager.Client(ctx, address)`
and `NewJWTMaker(secret)` stay explicit reusable primitives.

Keep `cmd.runWeb/prepareWeb/serveInstalled/listen` responsible for CLI
and process orchestration; `routers` owns route composition/startup steps;
resource packages own their Close; `services/<domain>` owns business
transactions; models own entity invariants. No separate config loaders
in DB/Redis/Auth and no multi-hop runtime `WebConfig` handoff.
The former CI ban on resource imports of `modules/setting` was superseded
by a check that **only** `modules/setting` may parse configuration.
`config check` must never publish, create log files or open resources.

The earlier file placement and FLC-01 notes below remain historical;
do not confuse previous decisions with the current canonical contract.
Verify compile, race tests, migrations, Docker and HTTP smoke at the
final implementation SHA before calling the code VERIFIED.

## Previous implementation focus — Source Ownership & File Placement (2026-11-21)

**Read [Source Ownership Audit](docs/architecture/source-ownership-audit.md) first.**
The user authorized a narrow code-organization refactor on `dev` based on
specific source/code-path evidence, not a new architectural framework.
BDSPro follows a Gitea-like distinction between `cmd.runWeb` (process
entrypoint), `cmd.serveInstalled` (resource/HTTP coordination),
`modules/setting` (configuration decoding plus explicit settings-driven
logger activation), `modules/log` (writer ownership),
`modules/graceful` (shutdown) and `routers` (HTTP composition).

File placement is determined by **single owner, caller/consumer, effect,
invariant and package import boundary**, not by file length or a desire to
make every file short. Avoid generic `helper.go`, `runtime_config.go`,
one-line forwarding wrappers or `modules/util` files without a real shared
contract. The CLI `config check` command stays in `cmd/config_check.go`;
the actual INI parser stays in `modules/setting`.
Retain the pure `LoadWebConfig` for `config check`, and never conflate
config validation with opening runtime resources. Verify final CI before
marking the new placement VERIFIED.

## Previous implementation focus — FLC-01 (2026-11-21)

The user has explicitly authorized a narrow implementation to improve startup
failure boundaries and long-term code readability. Current code is on `dev`;
the prior docs-only RG-00 rules below are historical scope, not a prohibition
on this explicitly approved work. Read
[the FLC-01 failure case](docs/architecture/flc01-startup-failure-boundary.md)
and [runtime bootstrap contract](docs/architecture/bootstrap.md) before
modifying shutdown or resource startup again.

FLC-01 kept `startup.Run`, graceful ownership, DB/Redis closers,
migration policy and public HTTP contract. Its historical guidance to avoid
importing `setting` in resource packages was **superseded** by ADR-G02-02;
continue avoiding generic framework/util helpers without an owner.
Route registration happens before resource startup; cooperative startup work
must observe shutdown cancellation rather than waiting until hammer.
The separate readiness/retry/observability proposals remain OPEN.

## Historical user-directed research focus — RG-00 (2026-11-20)

**Read [Research Governance](docs/architecture/research-governance/README.md) and [Current Phase](docs/architecture/research-governance/CURRENT_PHASE.md) before treating any older task as active.** This was a documentation-only architecture/context-governance assignment on `dev` before the subsequent explicitly authorized bootstrap and FLC-01 implementations. It did not authorize historical F40/P01 work or blanket changes to HTTP/security policy.

- A target folder tree, abstraction, naming convention, security "best practice" or recommendation must not become a conclusion until source-based requirement discovery and an end-to-end producer/consumer and cross-subsystem impact trace support it. See [requirements and evidence standard](docs/architecture/research-governance/REQUIREMENTS_AND_EVIDENCE.md).
- Compare actual pinned BDSPro, **Gitea and Laravel DRX** mechanisms where relevant; use Go/net/http/Chi/HTTP and security standards to define invariants. Never infer Go architecture directly from Laravel framework APIs.
- All research claims need an evidence class, reproducible SHA/path/line/call graph and limitations. Missing evidence stays OPEN; an illustration/standalone demo cannot claim actual BDSPro integration success.
- Use [evidence ledger](docs/architecture/research-governance/EVIDENCE_LEDGER.md), [case template](docs/architecture/research-governance/DESIGN_CASE_TEMPLATE.md) and [session handoff](docs/architecture/research-governance/SESSION_HANDOFF.md) so a new chat can resume without importing stale assumptions.
- Historical documents referencing Gin, P01/PR #218, parallel lanes or F40 may be useful but conflict with the verified current `dev` Chi source or the one-phase directive. Check reality before adopting them. No automatic new phase, PR, production edit or security change.

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
