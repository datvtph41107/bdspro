# BDSPro — Proportional Engineering Standard
**Established:** 2026-10-10 · **Working code branch:** `dev` · **Stable integration/release branch:** `main`.
**Status:** Durable operating principle, **not** SQL implementation, DB schema approval, policy/legal certification, or automated CI gate.
**Read with:** `MINDSET.md`, `PRACTICE-PROTOCOL.md`, `ENGINEERING-WORKFLOW.md`, `UNIFIED-STATE.md`, and the current `CHECKPOINT.md`.

> **Do not buy complexity until it prevents a credible failure or enables an observed benefit worth more than its ownership cost. Also, don't wait for irreversible data loss/security violations to first appear when the risk is already identifiable.**

## 1. Decision function: reality and proportion

For a meaningful engineering choice (schema, query, package, branch, test, index, infrastructure, automation, policy), identify:

1. **Current reality:** who is acting, what data and code actually exist, whether this is disposable lab, shared dev, pilot or production.
2. **Pressure:** present failure, reproducible counterexample, required business operation, or *credible costly risk* (with a concrete failure path). Mere speculation about a future giant company is not pressure.
3. **Minimum sufficient mechanism:** what is the smallest tool/constraint/change that defeats that particular failure without creating another truth?
4. **Evidence:** before → prediction → user-written minimal test/query/code → observed PostgreSQL or runtime output → non-guarantees. A green command does not prove overall correctness.
5. **Ownership cost:** migration compatibility, cognitive overhead, dependency lifecycle, runtime operations, team coordination, query/lock contention, failure recovery and maintenance.
6. **Trigger to revisit:** a measurable symptom, newly introduced user/legal obligation, repeated errors, new teammate, data volume or deploy topology.

**Judgment:** KEEP now, TRIAL in a disposable lab, DEFER with trigger, or REJECT. Use the smallest explanation that preserves the decision. For an ordinary trivial change, this is a mental model, **not a six-section form**.

```
CURRENT STATE → ACTUAL PRESSURE → SMALLEST RESPONSIBLE RESPONSE
  → PREDICT → WRITE BY HAND → OBSERVE → EXPLAIN LIMITS
  → COST/VALUE DECISION → COMMIT ONLY AN EARNED CHANGE
```

### A warning against two symmetrical mistakes
- **Underengineering:** “No production yet, so tenant isolation, identity integrity, secrets and recoverability can wait.” No: data is already persistent, personal or cross-tenant risk can be credible before first incident.
- **Overengineering:** “We may have millions of customers, so add Redis, Kafka, microservices, enterprise IAM, global sharding and all 164 tables now.” No: those components impose maintenance and failure modes without a currently validated job.

## 2. Concrete BDSPro proofs, bought value and remaining doubt

| Observed history / source | Real mechanism learned | Value purchased | Cost or what it does **not** establish |
| --- | --- | --- | --- |
| **DB-01..14:** Party + Person/Org PK/FK, Account→Person; two Accounts could reference one Person without UNIQUE | FK proves referenced identity exists, not XOR Party subtype or Account cardinality | Meaningful relational identity baseline | Does **not** decide whether 1 Person ↔ multiple Accounts is desirable or that a Party must have exactly one subtype |
| **DB-15:** two open Memberships for Organization 6 / Person 1 | A normal FK does not constrain duplicates matching `ended_at IS NULL` | A counterexample makes the constraint need testable | Does **not** justify episodic Membership as the long-term product model |
| **DB-16..18:** partial UNIQUE initially refused existing duplicates; after clean-up `organization_memberships_one_open_uq` succeeded; repeated open insert failed | `UNIQUE (...) WHERE ended_at IS NULL` constrains a subset and checks existing rows | Prevent double-open Membership at DB write boundary | Requires data reconciliation/migration; not authorization or guarantee intervals never overlap historically |
| **DB-19..20:** close Membership ID 2, create ID 5 in one transaction; `same_boundary=true`, `is_overlapping=false` | Half-open time `[joined_at, ended_at)` plus transactional boundary | Defines reproducible timing semantics | It is still a design hypothesis whether episodes or a stable membership + history is the best operational model |
| **CORE-01 CI:** original C01 draft `000006` allowed Person→Organization transition inside one transaction; newer draft `000001` forbade it | “Exactly one at commit” and “kind immutable over time” are **two different contracts** | Reveals policy ambiguity before production | Green tests for one contract do not validate the other; no unreviewed branch code is current dev truth |
| **Fresh baseline CI:** PostgreSQL `ON DELETE RESTRICT` raised SQLSTATE `23001`; initial test expected `23503` | Read exact database failure; distinguish implementation defect from test expectation defect | Fast, accurate debugging; transferable knowledge | One green CI suite is not proof of all concurrent histories or a runnable Listing API |
| **Branch audit:** nine remote branches, four old pointers shared one SHA, PR #2/#3 carried mutually incompatible migration timelines | Branch/PR inventory is not architecture validation | Main+dev and short-lived exception branches reduce attention and integration drift | No benefit in branch proliferation; preserve history but don't merge conflicting experiment code |
| **Local migration reset:** user removed 10 Listing migration files while old Go/sqlc routes still refer to Listing tables | Physical DB, migration history and generated/runtime code are separate states | Necessitates explicit compatibility check before declaring environment working | `go test ./...` compilation alone does not show Listing API runs on new schema |
| **Business Atlas V0.4:** 164 **candidate** tables but no native marketplace audience at cold start | Target conceptual vocabulary is not executable/validated product demand | Provides optional architecture map for future decisions | 164 migrations now would buy maintenance cost and lock early assumptions, not demonstrable buyer value |

Source of historical results: `docs/continuity/checkpoints/2026-10-09-trust-marketplace-core/DATABASE-REVIEW-AND-LAB.md`; CI/branch inventory: `UNIFIED-STATE.md` and PR #2/#3. **Historical lab state was reset by the user**; never claim these rows exist in the current local DB.

## 3. Proportional rigor: when to pay each cost

| Stage / environment | **Minimum valuable now** | Upgrade only when this pressure exists |
| --- | --- | --- |
| **A — Disposable learning lab** | Identify DB, predict behavior, write SQL personally, use direct `psql`, save meaningful results, learn PK/FK/UNIQUE/transactions/SQLSTATE; feel free to create and discard lab schema | No need for migration per failed query, full Go service, CI, PR, dashboards or distributed tooling |
| **B — Earned Core on local dev** | Approved one-row meaning, migration per coherent schema change, smallest write/read path, first negative test of core invariant, compatibility with old queries, coherent Git commit | Use deferred trigger/locks when FK/check/unique provably cannot enforce a required invariant; integration/concurrency tests when cross-row writes/races threaten it |
| **C — Shared dev / team / pilot** | Current membership and record authorization, lawful personal-data handling, test DB isolation, backup/restore appropriate to retained data, request retries where real network failure matters, basic app logs | Introduce branch/PR review when collaboration or risky parallel work needs it; create CI when repeatable manual builds/tests become slow/error-prone; add metrics when real user behavior is being measured |
| **D — Production / sustained usage** | Verified migration and rollback/forward plan, predeployment backups, observability for key failure modes, access/secrets controls, incident recovery, compatibility rollout, tested operational procedures | Add staged deployment, stronger CI/CD, optimized query/indexes, queue/cache/search or extra services **only** as throughput, reliability, isolation or operational costs justify them |
| **E — Growth or regulated capabilities** | Specific new actors, volume, jurisdiction, contract, audit and failure costs drive design | Add GIS licenses/ETL, financial settlement, cross-org Group, moderation, reputation, enterprise IAM, distributed architecture only when business and operational gates are met |

Stages overlap: a severe risk moves protection forward. Basic PK/FK, tenant isolation, credential safety and privacy **are not optional paid features**. An extra synthetic benchmark or automated load framework is optional until the query/data profile warrants it.

## 4. Rules by area

### SQL & migration
- **One semantic responsibility / reversible change unit, not one file per table nor one 164-table INIT.** Multiple tables/constraints in one migration are acceptable if they must establish one invariant together.
- Disposable experiments belong in lab SQL first. An accepted core fact belongs in migration once the user has demonstrated its behavior. A migration is **schema history**, not the business event “An updated price.”
- **Accepted migration files are historical artifacts.** Rewrite only on an explicitly disposable, confirmed clean baseline; don't silently rewrite applied shared/prod migrations. On populated DB prefer expand → backfill → validate → switch → contract as needed by compatibility pressure.
- For heavy DDL: measure data size, locks, transaction support, rollback practicality and users affected. Don't require sophisticated online migration patterns for three empty local tables.

### Query and integrity
- **Invariant ≠ feature:** PK/FK/UNIQUE/CHECK cannot each express every cross-table business rule. Prove the gap before choosing trigger/locks. Pair database constraints with server authorization where actor/purpose matter.
- Add indexes because queries/constraints need them; inspect realistic execution plans and statistics for performance claims. Don't extrapolate from two lab rows to a million rows. `EXPLAIN (ANALYZE, BUFFERS)` executes queries: isolate side-effecting plans or use safe tests.
- Distinguish `statement_at`, `recorded_at`, `effective_at` and latest-eligible *business* fact. Do not replace evidence history with last write wins.

### Go/code boundaries
- Start with readable small Go functions and explicit SQL, pgx/sqlc as currently chosen. Do not inherit packages/layers/service topology because an OSS repo has them.
- Extract package/function/interface only when multiple responsibilities, change frequency, isolation or testability create demonstrated pressure. Avoid abstraction as a prerequisite for comprehension.
- Treat Go compile, sqlc generation, DB query execution and full application behavior as different test surfaces.

### Git/main/dev
- **`dev`** is user's active coding branch; **`main`** stable review/release baseline. They're not required to be byte-identical. A branch is a temporary risk/isolation tool, not a KPI.
- Create a short-lived branch for real parallel ownership, larger risky refactor or urgent fix on deployed stable main. End with review/integration and branch deletion when resolved.
- Before sync/cleanup inspect Git status, branch heads, remote comparison and local uncommitted files; never use `reset --hard` or blanket merge for convenience. Historic evidence may remain via commits/bundles without keeping many active branches.

### CI/automation/observability
- **Automation buys value when repeated manual work or omissions cost more than build+maintenance.** User will personally design and write CI later, after direct CLI/Go/PostgreSQL workflows are understood. An existing CI workflow is research evidence, not a mandate to copy.
- Do not add dashboards or alerts without an operator, response action and meaningful signal. A metric that cannot change a decision may be noise.
- Default to lowest-cost observation that proves the current question: SQL output → focused test → Go integration → CI → metrics/load tests as warranted.

### Business/financial reality
- BDSPro is cold-start for audience; human brokers use Facebook/Zalo/phone/spreadsheets. A feature must be useful without assuming platform-owned lead traffic.
- Evaluate net value: user time saved and errors avoided **minus** extra capture/update steps, onboarding, support, privacy and infrastructure costs.
- Source claim ≠ verified Property ≠ public Listing; Contact ≠ Inquiry ≠ Opportunity; Account ≠ Person ≠ Organization membership. Avoid paid package granting legal publication rights.
- 164 logical table candidates are a map, not a roadmap to deploy all tables. User/payer field testing gates expensive product expansion; narrow lab core work can proceed before PMF without pretending PMF is proven.

## 5. Severity changes rigor, not stage labels

Use qualitative risk assessment rather than fake numeric precision:

- **High consequence or hard to undo (identity corruption, tenant leak, funds, credential theft, destructive shared migration):** strong constraints, defensive tests and protective operation **before** exposure, even in an early release.
- **Medium consequence and reversible (read-model stale, local test query, internal task status):** simple mechanism plus targeted test and clear corrective procedure.
- **Low consequence and cheap to redo (formatting a lab SQL exercise, folder naming, no-user UI wording):** experiment and correct quickly; no complex governance.

Ask: *How many users are affected? How likely is failure? Can we restore from the truth? What is the manual fallback?* This is contextual rigor, not a permanent compliance checklist for each line.

## 6. Minimal decision record — only when the choice matters

For a substantive architectural or schema decision, keep at most a short note (expand only if warranted):

```text
PRESSURE: what observable failure/job/credible risk?
OPTIONS: simplest acceptable approach vs more complex one
CHOICE: what now, at which stage?
PROOF: actual SQL/error/test/user evidence and what it DOES NOT prove
COST + REVISIT TRIGGER: what we own and when to reconsider
```

For tiny routine edits, keep this in conversation/commit reasoning. For reproducible invariants, preserve an explicit SQL/CI result. For cross-cutting design decisions, update `DECISIONS.md` / `CHECKPOINT.md` **at material checkpoints**, not after every command.

## 7. Operating example for the next BDSPro task

**Next task:** User's clean local `dev` database and migration reset, CORE-01 Party/Person/Organization lab.

1. Verify real local Git/DB state; don't pull over uncommitted deletions.
2. User defines one-row identity and predicts PK/FK behavior for bare Party and double subtype.
3. User writes minimal SQL in a disposable lab and presents exact output; assistant diagnoses.
4. Determine whether XOR and immutable subtype are *both* accepted business invariants, or only XOR. Green old CI under a different interpretation is not authority.
5. Choose smallest PostgreSQL enforcement that works under transactions/concurrent sessions **if required**; user implements migration and Go contract.
6. Check sqlc / old Listing code compatibility before calling the whole app working.
7. Commit one coherent user-authored result on `dev`. CI arrives when deployment/repeatability pressure is earned.

**Do not do now:** auto-generate 164 tables, merge incompatible PR #2/#3, add full SaaS security product suite, create microservices, copy complete SQL/Go answers, or require extensive process to review a three-table lab.

## 8. Supporting research — primary sources, not BDSPro-specific proof
- DORA — **Working in small batches**: https://dora.dev/capabilities/working-in-small-batches/ . Smaller batches shorten feedback loops and decrease failure-triage cost; does not mean one PR per SQL statement.
- PostgreSQL — **Partial indexes**: https://www.postgresql.org/docs/current/indexes-partial.html . Uniqueness may be constrained to rows satisfying a predicate; DB-15..18 supply BDSPro-specific evidence.
- GitHub — **GitHub flow**: https://docs.github.com/en/get-started/using-github/github-flow . Short-lived branches can isolate changes/reviews; not all work requires one.
- Google SRE — **Eliminating toil**: https://sre.google/workbook/eliminating-toil/ . Repetitive predictable manual operations motivate measured automation; Google's target numbers are **not** prescribed to BDSPro.
- DORA — **Team experimentation**: https://dora.dev/capabilities/team-experimentation/ . Learning requires permission to adapt hypotheses and specifications with evidence.
