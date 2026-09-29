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

At the beginning of a fresh context:

1. resolve the live `main` HEAD;
2. inspect local `git status --short --branch`;
3. read:
   - `docs/continuity/MEMORY.md`
   - `docs/continuity/CHECKPOINT.md`
   - `docs/continuity/DECISIONS.md`
   - `docs/continuity/HISTORY.md`
4. inspect source relevant to the current question;
5. inspect completed proof/tests for the exact source being discussed;
6. preserve unexplained or interrupted work;
7. continue only from the live checkpoint.

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

After a material decision, source boundary change, proof result or handoff,
update the relevant continuity documents and commit them.

Do not dump conversations into Git. Store compressed engineering truth.
