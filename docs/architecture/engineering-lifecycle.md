# BDSPro — Product engineering and professional lifecycle
Date: 2026-11-25. Owner-directed clarification of RESET-001. This is a working policy, not a production claim, implementation target or deployment authorization.

## Purpose and boundary
BDSPro is a real enterprise product and the sole backend developer's longitudinal Go engineering record. Learn by understanding, building, verifying, releasing, operating and improving real capabilities. Business outcomes, security, data integrity and the two-developer team's capacity come first. Do not turn production into a tutorial or adopt technology just to demonstrate a skill. Use isolated experiments for learning that is not yet justified by product requirements.

## One case, full lifecycle
| Stage | Evidence / questions |
| --- | --- |
| Discover | Actor, workflow, owner-confirmed outcome, acceptance examples, business invariants, constraints, unknowns |
| Establish baseline | Exact branch/SHA, source call path, current behavior, historical decisions and gaps, actual friction or credible preventive risk |
| Learn and compare | Go/library semantics, smallest adequate code, current vs alternative on same scenario; relevant upstream before/after, history and limitations |
| Decide | Owner approval, why now, why not keep current code, learning/dev/test/debug/deploy/operation/recovery costs, reversibility |
| Implement | Small diff, focused commits, PR, before/after behavior and regression evidence |
| Verify | Actual commands, versions, environments, results, gaps; owner can reproduce and explain mechanism and failure paths |
| Release | Actual build/release SHA, config/migration/backup/rollback as applicable, deployment authorization, environment and smoke outcome |
| Operate | Actual signals, observations and incident response if any; dated environment and release evidence, sanitized links |
| Revisit | Compare expected vs observed outcomes, retain/change/defer with reason, next bounded case |

Not every change reaches every stage immediately; mark NOT RUN, NOT DEPLOYED, N/A or UNKNOWN with reason. Keep case records small, use PRs and existing ADRs rather than duplicate bureaucracy. Never publish secrets or customer data in Git; link sanitized operational evidence or access-controlled records. Do not manufacture incidents to justify architecture. Security and data-integrity risks can justify preventive design without waiting for an incident.

## Why inspect actual repositories?
A source snapshot answers how code works at that revision. Git history and explicit PR/issue statements may answer why it changed; tests establish behavior under stated conditions; deployment and operational records establish actual use. A code pattern alone does not prove the author's motive, production benefit or BDSPro fit. Read the caller, callee, consumer, owner, failure and cleanup paths, not only folder names or signatures. Compare revisions when history matters; do not infer a historical sequence absent evidence. Gitea, Anthdm and other projects are optional references, not targets.

### Example: dependency ownership — verified external anchors
- Gitea [PR #17075](https://github.com/go-gitea/gitea/pull/17075), merge a4bfef265d9e512830350635a0489c2cdcd6508f: author explicitly reports growth of models and moves common database functions into models/db. This establishes the reported motivation for package decomposition.
- Gitea [PR #17100](https://github.com/go-gitea/gitea/pull/17100), merge 9302eba971611601c3ebf6024e22a11c63f4e151: author explains simplifying DBContext/GetEngine and allowing context-based calls instead of engine arguments. This is a subsequent contract change, not proof of a global DI-to-package-state migration.
- Anthdm [Gobank snapshot](https://github.com/anthdm/gobank/tree/b72b004ffbfd32e7f60850e4862c85e386af4c84): candidate for explicit store injection study; inspect main.go/api.go and consumers before claiming exact mechanics. Snapshot does not establish historical rationale or production usage.
- BDSPro historical [ADR-G02-02](decisions/ADR-G02-02-canonical-process-settings.md) records the prior decision concerning configuration handoff and one process configuration. It is not authority to apply package state to every resource. Reopen only with concrete evidence or an unmet requirement.

A possible controlled comparison (not a historical claim): one pool, one API, startup once. Compare constructor injection vs package access using identical behavior. Then introduce a second independent instance, startup error, parallel test, shutdown and a new consumer one at a time. Record actual wiring changes, isolation, error behavior and debugging effort. Keep one resource choice until there is a reason to revisit. Do not migrate production code simply to perform the comparison. No runtime comparison has been executed as part of this documentation change.

## Evidence of personal experience
Separate studied (source/contracts), reproduced (local lab), implemented (actual project code), verified (specific test/environment), deployed (actual release and environment), operated (real observation or incident), and reflected (lesson and revised decision). These are dimensions of evidence, not a mandatory ranking or a self-awarded competency certificate. Record what the developer personally understood, ran, investigated or reviewed and where AI assisted. Track actual learning effort if available; no fictional timing or incident claims.

## Start
RQ-01 is the only active case: clarify one business workflow, pin its baseline, understand the execution and choose the smallest sufficient change or no change. Its eventual release/operation evidence belongs to that case. Do not resume old architecture campaigns.

## Practice-led mentoring — RQ-01A.05 checkpoint
For unfamiliar mechanisms, default to a teaching loop: one specific task; ask the owner to predict behavior and identify the relevant code; provide a bounded exercise and safe commands; owner personally runs, observes, debugs and explains; AI reviews evidence, gives incremental hints and counterexamples, then shows a worked solution when requested or needed. Do not withhold help on explicit implementation requests or turn every routine change into an exercise. Do not count reading, AI execution, or generated code as the owner's practical experience. Distinguish intended output from observed output. Record minimal evidence (SHA, environment, command, outcome, reason, limitation, actual effort if known), not ceremonial documentation. Use synthetic/disposable environments. Do not weaken existing controls to learn. Separate product decision from educational experiment. [Go test tutorial](https://go.dev/doc/tutorial/add-a-test), [DORA small batches](https://dora.dev/capabilities/working-in-small-batches/), [DORA test automation](https://dora.dev/capabilities/test-automation/), [SRE monitoring](https://sre.google/sre-book/monitoring-distributed-systems/) are learning references, not mandatory tooling.
Proposed first exercise, not started: reproduce an existing local CLI configuration test before changing code, using [TestConfigCheckCommand at baseline](https://github.com/datvtph41107/bdspro/blob/6fbb2104806d45ed1de8005609694acf0d1c7d1f/cmd/config_check_test.go#L13-L31). Inspect fixture, CLI arguments and assertion, record actual git SHA and Go version, run go test ./cmd -run '^TestConfigCheckCommand$' -count=1 -v; inspect a deliberate failing expectation only in a disposable copy and restore it. No new runtime work, dependencies or deployment for this exercise. If toolchain/build fails, investigate that first, not redesign. Owner's practical results remain NOT RUN by owner until reported. Sequence later practice by actual need: unit/integration/HTTP, local runtime, isolated deployment, recovery, observation; no live exercise without approval.
