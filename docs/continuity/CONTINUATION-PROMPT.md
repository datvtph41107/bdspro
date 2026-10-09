# BDSPro Continuation Prompt

> **2026-10-09 design-branch continuation override:** For canonical DB core review on `architecture/canonical-database-final`, first read [2026-10-09 Trust/Marketplace Core checkpoint](checkpoints/2026-10-09-trust-marketplace-core/README.md) and its three linked detailed documents. They supersede contradicted 2026-10-06 semantic closures as review decisions; earlier acceptance stays historical evidence. The short prompt in that checkpoint is the preferred design-branch handoff. The generic `dev` bootstrap below remains for source implementation/Go workflow and must not be silently replaced.

This file is the single durable entry point for recovering BDSPro work after a
new chat, lost context, machine change, interruption, or long pause.

## One prompt to resume

Use this exact prompt in a fresh context:

```text
Continue BDSPro from durable state, not conversational memory.

Repository: datvtph41107/bdspro
Active development branch: dev
Stable/default branch: main

First read docs/continuity/CONTINUATION-PROMPT.md from the live dev branch and
follow it exactly. Treat main as the stable/default baseline unless live Git proves
otherwise.

Reconcile live Git/source and reproducible proof before trusting durable prose.
Reconstruct the Matching Start mindset and the current checkpoint before
proposing or mutating architecture or source.
```

That prompt is intentionally small. The repository owns the evolving recovery
procedure so the bootstrap prompt itself does not need to be rewritten whenever
the project advances.

## Recovery contract

In a fresh context:

1. Resolve the live `dev` HEAD and the stable/default `main` HEAD. If a local
   workspace is available, inspect its branch, status, index, uncommitted work and
   remote-tracking state before changing anything.
2. Read this file, then read in this order:
   - `docs/continuity/MASTER.md`
   - `docs/continuity/MINDSET.md`
   - `docs/continuity/DEVELOPER-IMMERSION.md`
   - `docs/continuity/ENGINEERING-WORKFLOW.md`
   - `docs/continuity/RESPONSE-PROTOCOL.md`
   - `docs/continuity/PRACTICE-PROTOCOL.md`
   - `docs/continuity/CHECKPOINT.md`
   - `docs/continuity/DECISIONS.md`
   - `docs/continuity/MEMORY.md`
   - `docs/continuity/HISTORY.md`
3. Inventory the complete tracked source tree before inferring architecture.
   While the repository is small, read all tracked source. As it grows, keep a
   complete tree-level inventory and read the current target plus its direct
   owners/dependencies; never invent behavior from filenames or stale prose.
4. Inspect the research/evidence relevant to the current question. Do not copy
   mature OSS structure without recovering the pressure that justified it.
5. Reconcile contradictions using the authority rules in `MASTER.md`.
6. Preserve interrupted/unexplained work. Never reset, clean, rebase, amend or
   overwrite merely to make recovery easier.
7. Recover the active-coding contract from `PRACTICE-PROTOCOL.md`. Do not provide
   paste-ready implementation as the default learning path.
8. State the recovered current reality, current question and next pressure before
   proposing a source/architecture change.
9. Continue from the current pressure. Do not skip ahead to a desired final
   architecture.

## Current database-design milestone

When the canonical database finalization package exists on the recovered branch,
read it immediately after the normal continuity documents and before proposing a
database mutation:

1. `docs/database/final/README.md`
2. `docs/database/final/CANONICAL-DATABASE-ACCEPTANCE.md`
3. `docs/database/final/BDSPro-CANONICAL-DATABASE.dbml`
4. `docs/database/final/CANONICAL-DATA-DICTIONARY.md`
5. `docs/database/final/LEGACY-MIGRATION-MAP.md`

That package is the accepted target review baseline. It does not replace live
migrations/source and does not authorize an all-at-once schema rewrite.

The database-specific one-prompt handoff is:
`docs/database/final/CONTINUATION-PROMPT.md`.

## Recovery invariant

The goal is not merely to recover files. It is to recover the same engineering
reasoning posture:

```text
CURRENT REALITY
    -> INTENT
    -> NEW REQUIREMENT
    -> OBSERVED PRESSURE
    -> WHY CURRENT FORM IS NO LONGER ENOUGH
    -> SMALLEST RESPONSIBLE RESPONSE
    -> EARNED BOUNDARY
    -> LANGUAGE/RUNTIME MECHANISM
    -> OSS / HISTORICAL EVIDENCE
    -> TRADE-OFF
    -> DECISION
```

If a future assistant starts from a preferred pattern, folder tree, service map,
DDD decomposition, microservice topology, or "best practice" before recovering
this chain, it has drifted from the BDSPro working model and must return to the
last proven pressure.

## Source roles

- `datvtph41107/bdspro`: active source and durable learning system.
- `dev`: active development and moving continuity branch.
- `main`: stable/default baseline; do not assume it contains the latest active checkpoint.
- old BDSPro repositories: historical evidence and real-case material only.
- mature OSS: evidence of mechanisms and pressure-tested boundaries, never a
  template.
- conversation memory: orientation only; it cannot override live Git, proof or
  durable repository state.

## Synchronization rule

After a material change in mindset, current pressure, source boundary, decision
or proof, update the appropriate continuity file in the same meaningful
milestone.

Do not store raw chat transcripts. Store compressed, reproducible engineering
truth.
