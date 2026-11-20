# Design case template — one engineering question

Copy this into a new dated case file only when one research question is explicitly active. Unfilled fields are `NOT INSPECTED`, never assumed true.

## 1. Case identity
- Case ID / phase / investigator / date:
- User-stated objective and explicitly forbidden changes:
- Scope / non-goals / decisions not being made:
- Working branch + exact SHA / other project SHAs:
- Public contract/semantics that must not change:

## 2. Problem independent of code shape
- Engineering force (observed vs hypothetical):
- Who produces the inputs? Who consumes them? Which process and trigger?
- System invariant and precise guarantees:
- Example of violation in production terms; whether actually reproduced:
- Baseline minimal Go/net/http/Chi or direct call without abstraction:

## 3. Three source traces, not three isolated snippets
| Source | Entry → constructor → registration → runtime → caller → callee → side effects → cleanup | SHA/file/lines | Fact vs inferred intent |
| --- | --- | --- | --- |
| BDSPro actual | | | |
| Gitea | | | |
| Laravel DRX (when relevant) | | | |
| Official contract / alternate Go source | | | |

Attach the actual code excerpts (not paraphrases), exact import edges and full data/error/continuation path. A historical file/README is not substitute for the live source.

## 4. Cross-system and operational impact
| Area | Inspected / N/A / NOT INSPECTED | Concrete callers, constraints, observed evidence |
| --- | --- | --- |
| Web route and public API | | |
| CLI / migrations | | |
| Workers/background/process lifecycle | | |
| DB/transactions/Redis ownership | | |
| Auth/AuthZ/privacy/threat model | | |
| Request context/concurrency/cancellation | | |
| Docker / Linux binary / reverse proxy | | |
| Logging/metrics/tracing/response | | |
| Tests/CI/dependencies/imports | | |
| Feature evolution and deletion | | |

## 5. Security/failure scenarios
- Trust boundary / assets / attacker capabilities / assumptions:
- Fail-open vs fail-closed, operator-visible signals:
- Error before/after response commitment:
- Reproduction input, command and ACTUAL output or NOT RUN:
- Negative case or counterexample that would disprove design:

## 6. Genuine alternatives and costs
| Option | Implementation/code shape + ownership | Specific import/caller changes | Test/ops/learning cost | Unresolved |
| --- | --- | --- | --- | --- |
| A — retain/minimal | | | | |
| B — small refactor | | | | |
| C — more structural (only if justified) | | | | |

No folder tree is an accepted target until ownership, all consumers, dependency graph, constraint and change surface are reviewed. Mark all predicted benefits as inference; give measurements only if executed with environment and units.

## 7. Decision gate
- Status: OPEN / PROPOSED / ACCEPTED / REJECTED / SUPERSEDED
- Evidence supporting candidate; evidence against it:
- Gaps and the smallest next *research* action:
- User/reviewer approval link or statement (for ACCEPTED):
- If code is authorized: changed files, sequence, rollback, compatibility invariants, tests on final HEAD:
- Post-implementation verification links / CI / unresolved risk (VERIFIED only with actual results):

## 8. Session handoff
- Summary of facts verified:
- Disputed/contradictory evidence:
- Last inspected SHAs and branch:
- What was NOT done:
- One next action, staying within the active phase:
