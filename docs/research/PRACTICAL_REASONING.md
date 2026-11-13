# Program 01 — Practical Technical Reasoning

**Status:** proposed, 2026-11-13. Synthesizes concepts from four user-supplied working texts; **the original uploaded files are not published to this public repository**. Their examples, project names, measurements and code snippets are illustrative unless independently verified.

## What this adds

Research how engineers **discover and frame problems, develop claims, understand mechanisms, judge user/business impact, make decisions, implement, learn from failure and revise their mental models**. Study the full life of software and neighboring computing disciplines, not just architecture or Go.

| Lens | Contribution | Failure mode |
|---|---|---|
| Thought supply chain / origin of the problem | Trace meaning and user need through requirements, analysis, decisions, code and product outcome. Distinguish discovering a problem from accepting a task | Solving symptoms instead of causes |
| First principles + contextual systems thinking | Drill down into how mechanisms work; zoom out to stakeholders, dependencies, costs and project phase | Reinventing a proven tool, or applying an attractive solution in the wrong context |
| Action and feedback | Admit uncertainty, take the smallest safe testable action, use failures as data, deliberately remove/defer work | Analysis paralysis and unsupported certainty |
| Logical analysis and operation | Surface → Root → Mechanism → Impact → Solution → Operation; test failure paths and long-term ownership | Jumping directly from symptom to implementation |

These are **useful lenses, not empirical laws or a mandatory process**.

## Seven-step reasoning loop

0. **Meaning/purpose:** Why does the system exist and what real human/business task matters?
1. **Surface:** What exactly was observed? Identify the artifact (issue, interview, trace, test, bug, PR).
2. **Root question:** Separate symptom from at least two plausible causes; identify who is affected, when and at what cost.
3. **Mechanism:** Trace code path, state, concurrency, transaction, data and network flow; distinguish what is drawn from what is actually observed.
4. **Impact:** Analyze functional correctness, UX, security, maintainability, team responsibilities, resource costs and operations.
5. **Competing options:** For each: **CLAIM → EVIDENCE → WARRANT → REBUTTAL → RESPONSE → FALSIFYING TEST**.
6. **Smallest bounded action:** Read source, reproduce a failure, run a controlled experiment, create a scoped patch or defer intentionally.
7. **Operational feedback:** Compare outcomes with predictions; inspect failure/recovery/rollback; update the decision and record the next question.

**Two directions:** drill down for mechanism, zoom out for context. Stop when further reading will not change the next justified test or decision.

## Evidence integrity: examples are not measurements

Precise-looking numbers in teaching examples (latency, percentage improvements, throughput, number of developers, lines of code or scale) are **illustrative** until their original data, environment and measurement process are available. A hypothetical "production outage" or sample Go code is not a documented incident. Do not convert such material into claims about BDSPro or third-party projects.

Technical statements worth questioning rather than accepting by authority:

- **Fewer lines of code = proportionally higher productivity:** not established; measure change cost, correctness, learning and delivered value.
- **JWT cannot be revoked:** too absolute; a purely stateless validator cannot itself know revocation state, but additional server-side checks/denylisting are possible.
- **Caching always makes requests faster by a fixed factor:** workload-dependent; stale data, stampedes, invalidation and operations may outweigh benefits.
- **Cursor pagination is always O(log n)** or **an authorization middleware guarantees resource-level permission:** depends on indexes, query, rules and execution path.
- **Monolith must grow into cache → replicas → queue → microservices:** illustrative sequence, not a law or required roadmap.
- **Cross-discipline physical design anecdotes** are mechanisms for asking questions, not historical evidence until tied to original engineering records and measurements.

Preserve the original thesis, then label critique and competing evidence rather than silently rewriting it.

## Issue worksheet

- **Question and purpose:** What work fails or is costly and for whom?
- **Observation:** exact artifact, timestamp/environment, raw data and provenance.
- **Hypotheses:** at least two mechanisms and their assumptions.
- **Visual mechanism:** call graph, query plan, state transition, request timeline; mark inferred edges.
- **Impacts:** business/domain, security, users, code, ownership, costs, operations.
- **Disconfirming test:** what outcome rejects each hypothesis?
- **Smallest action and actual result:** proposed versus executed, before/after and risks.
- **Decision and next question:** justified for this context, rejected, deferred or still unknown.

### Worked question (hypothetical, not BDSPro evidence)

A user says "search is slow". This may mean high HTTP latency, poor relevance or low perceived responsiveness. A query plan tests part of the first hypothesis; relevance evaluation tests the second. Neither alone proves the user's full experience. Do not add an index before defining and observing the actual problem.

## Cross-chat handoff

Read this with [PROGRAM_01.md](./PROGRAM_01.md), [VISUAL_EVIDENCE.md](./VISUAL_EVIDENCE.md), [SOURCE_REGISTRY.md](./SOURCE_REGISTRY.md) and [DISCUSSION_BACKLOG.md](./DISCUSSION_BACKLOG.md). New chat: identify **observed / inferred / refutable / user impact / next smallest action**. For BDSPro always refresh source at exact SHA and obey existing branch, ownership, test and PR controls. These lenses grant no autonomous code permissions.
