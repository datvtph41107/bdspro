# BDSPro Continuation Prompt

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
