# BDSPro Continuity Master

This is the durable entry point for continuing work in a fresh chat, machine or
context.

## Mission

Build BDSPro as a clean-slate Go backend while learning the language, tooling and
engineering mechanisms from first principles.

Do not reconstruct architecture by copying the previous repository.

## Current authority

Repository:

`datvtph41107/bdspro`

Default branch:

`main`

Preferred local workspace:

`~/projects/bdspro`

Previous repository:

`datvtph41107/bdspro-backend`

Previous Rebuild V2 branch at transition:

`architecture/rebuild-v2@c1245796e320d533ca29d7b92bb23fa28e8380a1`

The previous repository is historical evidence only. Its service topology,
package layout and architecture decisions are not inherited as authority.

## Recovery order

The canonical bootstrap is `docs/continuity/CONTINUATION-PROMPT.md`.

At the beginning of a fresh context:

1. resolve the live `main` HEAD;
2. inspect local `git status --short --branch` when a local workspace is available;
3. read, in order:
   - `docs/continuity/CONTINUATION-PROMPT.md`
   - `docs/continuity/MINDSET.md`
   - `docs/continuity/DEVELOPER-IMMERSION.md`
   - `docs/continuity/ENGINEERING-WORKFLOW.md`
   - `docs/continuity/RESPONSE-PROTOCOL.md`
   - `docs/continuity/PRACTICE-PROTOCOL.md`
   - `docs/continuity/CHECKPOINT.md`
   - `docs/continuity/DECISIONS.md`
   - `docs/continuity/MEMORY.md`
   - `docs/continuity/HISTORY.md`
4. inventory the complete tracked source tree; while the repository is small,
   read all tracked source before inferring architecture;
5. inspect source and completed proof/tests relevant to the current pressure;
6. preserve unexplained or interrupted work;
7. state the recovered current reality, question and next pressure;
8. continue only from the live checkpoint using the Matching Start mindset.

## Authority order

```text
1. live source / Git at exact commit
2. reproducible proof / tests for that source
3. CHECKPOINT.md
4. DECISIONS.md
5. MEMORY.md / HISTORY.md
6. external research
7. conversational memory
```

## Method authority

`docs/continuity/MINDSET.md` governs architecture/engineering reasoning.
`docs/continuity/DEVELOPER-IMMERSION.md` governs long-horizon professional
practice: fundamental fluency, repetition, operational drills, increasing
failure pressure and multi-role reasoning.
`docs/continuity/ENGINEERING-WORKFLOW.md` governs the everyday command/coding
rhythm, stack-specific reflexes, state/risk classification and primary-source
trace for Git, Bash, Docker, PostgreSQL and Go.
`docs/continuity/RESPONSE-PROTOCOL.md` governs how assistant responses separate
human work/understanding from the minimum evidence the assistant needs back, and
prefers IDE-first source review over unnecessary terminal extraction.
`docs/continuity/PRACTICE-PROTOCOL.md` governs the active coding, research,
source-tracing and debugging method.
`CHECKPOINT.md` governs the current target. Live source and reproducible proof
govern factual reality.

A future architecture or checkpoint may change without changing the mindset.
A mindset change must be explicit, evidence-backed and recorded.

## Hands-on working protocol

The user types the code and commands personally.

For each meaningful step:

```text
LARGER GOAL
  -> CURRENT QUESTION
  -> SMALL GOAL
  -> EXACT COMMAND / CODE
  -> WHAT IT CHANGES
  -> HOW THE MECHANISM WORKS
  -> EXPECTED OUTPUT
  -> REAL OUTPUT
  -> DIAGNOSIS
  -> LESSON
  -> NEXT QUESTION
```

Every command/code fragment should be explained at three levels:

1. immediate purpose;
2. underlying Go/Git/Linux/runtime mechanism;
3. how it advances the current engineering goal.

Do not hide important operations behind generators or scripts before the
underlying commands are understood.

Do not provide complete paste-ready implementations as the normal learning path.
Use the solution-escalation ladder in `PRACTICE-PROTOCOL.md`: requirement ->
keywords/syntax -> pseudocode -> focused fragment -> complete implementation only
when a real attempt, debugging/recovery need or explicit request justifies it.

For unfamiliar APIs and mechanisms, tracing primary documentation and, when
needed, implementation source is part of the coding work. The user should explain
the discovered contract in their own words before relying on it.

## Decision protocol

For material decisions use:

```text
QUESTION
FACTS
PROBLEM
REQUIREMENTS
OPTIONS
VALUE
COST
FAILURE MODES
MINIMUM EXPERIMENT
EVIDENCE
DECISION STATUS
NEXT PROOF
```

Statuses:

- OPEN
- HYPOTHESIS
- PROVISIONAL
- CLOSED
- SUPERSEDED

## External evidence

Use official Go documentation first, then mature production repositories and
style guides.

External projects are evidence, never templates to copy blindly.

For a borrowed idea, ask:

```text
what problem does it solve there?
what mechanism makes it useful?
does the same pressure exist here?
what does adopting it cost?
what proof would justify it here?
```

## Mutation safety

Before destructive or broad changes:

- inspect current branch and status;
- understand the files affected;
- prefer the smallest reversible change;
- do not reset/clean/rebase/amend as a recovery shortcut;
- preserve unknown work.

## Durable synchronization

After a material mindset change, decision, source boundary change, proof result
or handoff, update the relevant continuity documents and commit them.

Keep `MINDSET.md` stable unless the reasoning method itself changes.
Keep `CHECKPOINT.md` current as pressure moves.

Do not dump conversations into Git. Store compressed engineering truth.
