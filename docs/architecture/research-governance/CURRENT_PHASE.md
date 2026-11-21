# Historical phase — RG-00: Evidence/Context Governance

**Status:** DOCUMENTED / HISTORICAL. **Date:** 2026-11-20.

**2026-11-21 current implementation note:** The user has since authorized
startup and graceful refactors followed by FLC-01 failure-boundary hardening.
See [FLC-01](../flc01-startup-failure-boundary.md) for the newly active,
narrow scope; its verification requires final commit CI. The RG-00 restrictions
below describe their original docs-only phase, not a current blanket ban.
Other research topics remain OPEN pending their own decision gates.
**Target repository:** https://github.com/datvtph41107/bdspro
**Working branch:** `dev` (re-fetch live HEAD on every session).
**Purpose of this phase:** Establish an auditable, durable research operating system and shared context before any HTTP Core refactor. No new framework/package architecture is authorized by this record.

## Problem to solve now

Architecture discussions are spread over many sessions and partial summaries. A fluent source style or a single HTTP ingress example has previously led to overly early folder-tree/decision proposals. Historical architecture docs may describe Gin while the current runtime is Chi. A new assistant must reconstruct current reality without trusting outdated summaries, and must be unable to turn unverified hypotheses into accepted decisions.

## Requirements / non-goals

- Preserve intent and context across sessions in a short repository-local starting path.
- Require proof-oriented research: source URLs with SHA + line ranges, caller → callee → consumer, construction vs runtime, security/trust/state/error/lifecycle, alternative designs, real vs model tests and operations.
- Establish a cross-subsystem impact matrix before naming packages, approving boundaries or writing production code.
- Compare Gitea and Laravel DRX where relevant; Go/Chi/net/http is the semantic baseline, not one of the competitors.
- Use one active investigation at a time; store new ideas as OPEN until current gate closes.
- No AuthN/AuthZ, route contract, middleware, Redis, deployment, data migration, branch or PR changes in RG-00.
- Do not re-activate historical F40, P01–P07 or parallel lanes because an old doc calls them active; resolve conflicts against the latest explicit task.

## Phase artifacts

1. Root AGENTS.md points to this source of truth.
2. Architecture skill and research protocol enforce evidence and a one-phase rule.
3. Requirements/evidence method, source ledger, case template and session handoff exist and cross-link.
4. All new claims are labeled OBSERVED / INFERENCE / HYPOTHESIS / NOT VERIFIED.
5. Existing stale documents are documented as historical/conflicting; no silent edits erasing history.

## Known baselines (snapshots, not evergreen facts)

- BDSPro `dev`: `abe04e46269632ef671a92bab00319b52696e8f5` — confirmed when drafting; `modules/web/router.go` constructs a Chi mux; `routers/init.go` builds HTTP handler.
- Gitea: `fbf986d1c7ba1b0ad0078105b3adb0a73b769295` — reference mechanism only.
- Laravel DRX `main`: `767737d3a9530d0ec3ff94876752e40ccb302a15` — its `app/Http/Kernel.php` defines global, group and route middleware.
- See precise source anchors and limitations in [EVIDENCE_LEDGER.md](EVIDENCE_LEDGER.md). Do not extrapolate to uninspected files or newer commits.

## RG-00 exit criteria

- User has reviewed/accepted the research process and phase scope.
- Any corrections are reflected in AGENTS / evidence method / handoff.
- A *single* next research question is named with scope, baseline and evidence tasks, without presupposing its solution.

**Candidate next research question (not active/accepted):** HTTP ingress and routing ownership at `routers/init.go → modules/web → services/context`, including trusted client IP, CORS, logging/recovery, HTTP limits, request data and security gates. This is a study subject, **not** an approved `modules/httpmeta`, `routers/ingress` tree or architecture decision.

## Change/decision ledger

| Date | Event | What it proves | Not approved |
| --- | --- | --- | --- |
| 2026-11-20 | Requested durable agent/documentation update and strict source-first research | Intent and phase scope | No HTTP implementation, package migration, ADR or deployment change |

Next agent action: follow [SESSION_HANDOFF.md](SESSION_HANDOFF.md); verify live HEAD, inspect current source, present one focused question and its research boundary. No automatic phase advancement.
