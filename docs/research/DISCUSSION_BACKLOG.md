# Program 01 — Discussion and Investigation Backlog

**Status:** initial hypotheses/questions only, 2026-11-13. No item below is an accepted technical decision for BDSPro. Each question should become a separate research note under `docs/research/notes/` only when actually investigated.

| ID | Research question / tension | Starting sources | Evidence and counterexample to seek | Initial status |
|---|---|---|---|---|
| Q01 | What actually counts as developer productivity: lines, activity, review quality, delivery, correctness, experience or sustained product outcomes? | S02, S03, S07 | Compare survey, telemetry and end-to-end outcome; risk of optimizing a proxy | Proposed |
| Q02 | How do real engineering teams form, reject and revise design/code-pattern proposals? | S03, S05, S04 | One accepted and one rejected proposal with discussion, caller changes, tests and maintenance consequences | Proposed |
| Q03 | When does separating packages/interfaces/architecture reduce work, and when does it increase cognitive and migration costs? | S08, S04, S05 | Compare competing module designs under a concrete workload, change request and review cost | Proposed |
| Q04 | How do incidents, migrations and failures turn into code changes and better operations rather than repeated outages? | S09, S07 | Original incident timeline, changes, rollback test and postmortem follow-through | Proposed |
| Q05 | Does AI shorten software delivery or move cost from generation to comprehension, verification and integration? | S10, S11, S07, S06 | Compare experiment vs survey; inspect tool version, developer experience and verification burden | Proposed |
| Q06 | Which problems do Go developers actually report, and how does the Go community evaluate possible solutions? | S06, S05 | Survey question/method + a specific proposal and reviewer discussion; compare with real repo adoption | Proposed |
| Q07 | How do research results survive changes in product requirements, business rules and long-lived codebases? | S01, S04, S08 | Longitudinal code changes and real business constraints, not architecture diagrams alone | Proposed |

| Q08 | How can a diagram, flamegraph, trace, query plan or photograph contribute to a **falsifiable technical argument**, rather than just tell a convincing story? | S14, S15, S16, S17 | Compare designed call path with sampled execution and raw artifacts; audit missing spans/links | Proposed |
| Q09 | How do engineers **discover the real problem** before selecting code, database, framework or architecture? | S01, S03, S04 + practical reasoning guide | Compare user observation, a rejected hypothesis, bounded experiment and subsequent outcome | Proposed |
| Q10 | How should failures, unknowns, reduction/removal of code and smaller experiments change project decisions? | S03, S04, S09 + practical reasoning guide | Trace an actual incident or change history and costs of decisions that were deferred/reversed | Proposed |
| Q11 | When do cross-disciplinary metaphors or photographs help engineers find mechanisms, and when do they introduce a false causal analogy? | S14 + visual evidence guide | First-party technical report, measured physical constraints, software counterexample | Proposed |

## Working a discussion thread

1. Select one ID, define its concrete claim and a competing hypothesis.
2. Read at least one first-party source beyond the abstract/landing page.
3. Trace relevant citation chains, a dissenting source and, if applicable, source-code/PR history.
4. Write a bounded test or investigation with disconfirming conditions.
5. Record conclusions with evidence level, applicability and unknowns; cite exact pages/sections or commit SHAs.
6. Return a **new, sharper question**, rather than closing a topic through an appealing summary.

## Relationship to later work

Program 01 builds a **method and evidence graph**. The future **3 tiers × 3 columns** growth/operations learning framework and curated **five Go monolith case studies** must be informed by that graph. Do not pick them in advance based on stars, directory layout or affiliation.

**Handoff for a new chat:** read `AGENTS.md` and all five `docs/research/` program files, pick a Q-ID, refresh URLs/versions and begin by identifying what prior sources actually establish versus what remains hypothesized.
