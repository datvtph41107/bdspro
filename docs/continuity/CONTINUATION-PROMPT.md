# BDSPro Continuation Prompt

> **2026-10-10 LATEST CONTINUATION OVERRIDE — EVIDENCE-DRIVEN PAIR ENGINEERING.**
> For a new conversation use the exact [portable prompt](checkpoints/2026-10-10-evidence-driven-auth-handoff/CONTINUE-PROMPT.md), then read the [current handoff checkpoint](checkpoints/2026-10-10-evidence-driven-auth-handoff/README.md) and **[EVIDENCE-DRIVEN-REASONING.md](EVIDENCE-DRIVEN-REASONING.md)**. These are authoritative for the user's explicitly confirmed learning/communication method: actor + visible state + operation + causal mechanism + equivalent alternatives + counterexample + real test + limits + action. They extend MINDSET/PRACTICE/RESPONSE protocols. Do not make unsupported "simpler/safer/better" assertions or offer only abstract tables. User codes/operates locally; assistant is a concrete proactive reviewer/mentor; do not repeatedly ask whether to continue.
>
> **Latest verified GitHub event:** commit [545d405e](https://github.com/datvtph41107/bdspro/commit/545d405e00052971f21c3953d101ac63c808933d) removed legacy Listing migration files from dev; residual Listing/sqlc source remains. Later method/handoff commits are **remote docs-only**; local Git/DB state must be checked from the user's own output, not inferred. No fresh user Auth migration or login/session execution is yet proven. All older next-step/branch-state paragraphs below are preserved as historical and yield to live source/proof and the new handoff. Continue user-authored AUTH Core implementation in narrow, falsifiable slices; do not generate the 164-table canonical baseline wholesale.

> **2026-10-10 CURRENT METHOD + BRANCH RESTORATION:** [REPOSITORY-WORKING-CONTRACT.md](REPOSITORY-WORKING-CONTRACT.md) is the **single** Git/local/GitHub/DB standard. `main` has been restored from its exact historical last commit `3f90f7789`, **not certified for production**; `dev` remains active code branch and was GitHub default at restoration. When user says **"Tiếp tục"**, inspect live GitHub branches and exact SHA plus user's reported **local** `git status`/database results before changing any source. User writes SQL/Go/CI; assistant gives context, asks prediction, reviews actual output and failure boundaries. Ignore superseded earlier passages claiming `main` missing, active `architecture/canonical-database-final`, merge PR#2/#3 or already implemented local Core. Next gate: preserve user's uncommitted Listing migration deletions, synchronize local `dev` safely, then user-authored CORE-01 SQL lab. Do not restore removed branches by ritual.

---


> **CURRENT LOCAL ↔ GITHUB AUTHORITY — 2026-10-10:** Live GitHub contains **only `dev`**, also the default branch; `main` and earlier checkpoint/implementation branches were removed. When user says **"Tiếp tục"**, inspect live `dev` and the user's actual **local** `git status`, then read the **Repository Core Contract** in [ENGINEERING-WORKFLOW.md](ENGINEERING-WORKFLOW.md), the current [CHECKPOINT.md](CHECKPOINT.md) and [PRACTICE-PROTOCOL.md](PRACTICE-PROTOCOL.md). The user **writes SQL, Go and CI**; assistant supports reasoning, focused tests and code review, not automatic implementation or Git mutations. The user's last-reported local state included ten deleted Listing migration files and being behind remote; this is **not proof of its current state**. Old paragraphs about `main`, `architecture/canonical-database-final`, ENG-00 and PR #2/#3 are historical records only, not current instructions.

---


> **2026-10-10 canonical handoff override:** When user says **"Tiếp tục"**, recover [2026-10-10 Business-to-Core checkpoint](checkpoints/2026-10-10-business-to-core/README.md) on branch `architecture/canonical-database-final`, then inspect real `dev` source and start ENG-00→CORE-01 instead of restarting BUSINESS-01/02/03 analysis or implementing 164 tables. Business Atlas V1 covers all BDSPro business families at analysis level; four V0.4 source/CRM tables are logical candidates only. Later legacy prompts below remain for historical/engineering details; no unverified claim of actual implementation or automatic memory sync.

---


> **2026-10-09 design-branch continuation override:** For canonical DB core review on `architecture/canonical-database-final`, first read [2026-10-09 Trust/Marketplace Core checkpoint](checkpoints/2026-10-09-trust-marketplace-core/README.md) and its four linked detailed documents (including `ORGANIZATION-NETWORK-AND-PROVIDER-ONTOLOGY.md`). They supersede contradicted 2026-10-06 semantic closures as review decisions; earlier acceptance stays historical evidence. The short prompt in that checkpoint is the preferred design-branch handoff. The generic `dev` bootstrap below remains for source implementation/Go workflow and must not be silently replaced.

This file is the single durable entry point for recovering BDSPro work after a
new chat, lost context, machine change, interruption, or long pause.

## One prompt to resume

Use this exact prompt in a fresh context:

```text
Continue BDSPro from durable state, not conversational memory.

Repository: datvtph41107/bdspro
Active development branch: dev
Current default remote branch: dev (verify live, do not assume a separate main branch)

First read docs/continuity/CONTINUATION-PROMPT.md from the live dev branch and
follow it exactly. Use the live GitHub branch/default state and local Git output as truth; never infer a main branch.

Reconcile live Git/source and reproducible proof before trusting durable prose.
Reconstruct the Matching Start mindset and the current checkpoint before
proposing or mutating architecture or source.
```

That prompt is intentionally small. The repository owns the evolving recovery
procedure so the bootstrap prompt itself does not need to be rewritten whenever
the project advances.

## Recovery contract

In a fresh context:

1. Resolve the live default `dev` HEAD and actual GitHub branches. Do not assume `main` exists. If a local
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
- `main`: historical former branch removed in the current repository; only treat as present after checking live Git.
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
