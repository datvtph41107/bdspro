# BDSPro task contract: investigate, implement, verify, learn

Use this record for nontrivial tasks, particularly authorization, credential/session, database, externally observable API and concurrent/async changes. Reduce formality for tiny mechanical edits; never reduce evidence for a high-risk change.

| Gate | Engineer must answer | Required output |
| --- | --- | --- |
| G0 — Baseline | Which exact SHA, live path and failure/reported need? Is the observation direct or historical? | Commit, files, runtime evidence, unknowns |
| G1 — Contract | Who acts on which resource? What is denied? What must stay true after retry, rollback, crash and concurrent writes? | Acceptance cases written before solution |
| G2 — Design | Why is this owner correct? What is the smallest solution? What alternative was rejected and why? | Data shape, dependency/transaction boundary, risk |
| G3 — Implementation | Does the relevant test fail on the wrong behavior and pass on the fix? Can the real API/CLI prove it? | Diff, test commands, results, runtime artifact |
| G4 — Acceptance | Did an independent reviewer verify the same patch/commit? Is CI required before merge? What changes to guidance/checks remain? | Verdict, CI URL, review, open risk, knowledge update |

## Evidence quality

- **Observed:** code/command/runtime evidence from the identified SHA and environment; include exact scope.
- **Inferred:** an explanation consistent with observations but not independently executed.
- **Unverified:** proposed behavior or documentation claim without executable proof.
- **Contradicted:** a claim falsified by source, test, or measurement; correct the claim and capture the counterexample.

`go test` green is not sufficient to claim correct authorization ordering, real provider behavior, production readiness, p95/p99 improvement or safe deployment. Never call a test skipped or unable to run a pass. CI green after merge is not equivalent to a mandatory pre-merge gate.

## Reusable task record

- **Intent:** Problem, impact, why now. Explicit non-goals.
- **Baseline:** Repository, branch, full SHA, PR dependencies and known code state.
- **Ownership:** Actor/action/resource; route, context, service/model, SQLC and transaction owner.
- **Acceptance:** Positive, denied, boundary and failure cases. Add concurrency, rollback, replay and outage cases when relevant.
- **Reference:** Applicable BDSPro decisions, Gitea source at an identifiable revision, and independent language/database/vendor sources. Mark facts versus interpretation.
- **Design:** Minimal option, alternative, data shape, failure policy and removal/rollback cost.
- **Execution:** Files changed, commands actually run, results and commit/CI artifact URLs.
- **Review:** Independent verdict PASS / FAIL / INCONCLUSIVE, unsolved risks, approval and merge state.
- **Learning:** Which source of truth and regression guardrail changed? Which earlier guidance was superseded?

## Permission rules

Review and deployment authority are not implied by an agent's ability to run Git commands. An agent may prepare a branch and PR when asked; shared-branch merge, force push, data deletion and production changes require authorization and appropriate project protections. Security-sensitive changes should require independent review of the final patch, not only the author-agent's self-evaluation.

## Measuring whether this protocol helps

For a pilot, compare equally scoped tasks with and without the protocol. Record elapsed time and human attention, first-pass acceptance, escaped bugs, rework, verification detection rate, false findings, compute/token cost and rollback cost. Use the same acceptance tests and independent evaluator. Count *accepted tasks* rather than raw PR count, and never claim a speedup without a baseline, variance and error accounting.
