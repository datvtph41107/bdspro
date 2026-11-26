# RESET-001 / RQ-01 — Baseline and business requirement discovery
Date: 2026-11-25. Status: OPEN for requirement discovery. Governance reset is owner-directed. Baseline dev@6fbb2104806d45ed1de8005609694acf0d1c7d1f (refresh on next session).

## Active work
First clarify the next real enterprise workflow with the owner: who acts, why, input, output, success/denial paths, data ownership, business rules and acceptance examples. Identify what exists in the current backend, what is missing, and which constraints are confirmed. Choose one workflow; none is selected yet. Do not infer backlog from the 100+ table schema or legacy endpoints.

Inspect only the source required to establish the baseline for that workflow and its prerequisite execution path. Record Go/stack understanding, existing evidence and gaps. Existing settings, router, DI, graceful and other core mechanisms are not scheduled for redesign by default. Investigate one of them only when a concrete blocker, risk, or requirement warrants it.

## Exit criteria
Owner confirms a bounded workflow and acceptance examples; relevant baseline SHA and code paths are recorded; missing facts and risk ranked by impact and evidence; next change or explicit no-change decision has a scoped rationale. No runtime modifications from this reset. Previous G02/F40/parallel tasks are parked; PRs are not automatically merged, closed or resumed.

## Professional lifecycle objective (owner clarification, 2026-11-25)
RQ-01 remains the only active case. No business workflow selected yet. Establish one real workflow and its acceptance criteria, identify baseline code and relevant Go/stack knowledge, then record a bounded implementation and verification plan including a release/operation plan if needed. Keep actual work and optional learning experiments separate. See [Engineering lifecycle](../engineering-lifecycle.md); do not start a DI/package-level migration as a learning exercise.

## RQ-01 discovery checkpoint
[Product case](../../product/RQ-01-product-and-first-workflow.md) and [source evidence](../../product/SOURCE_BASELINE.md) now define the first discovery requirement: identify one segment and real job. Contact intake/follow-up is an unconfirmed candidate, not selected implementation. Resolve buyer, user, data owner, workflow, acceptance and operational risks with product authority and prospective users. [Review gate](../../product/REQUIREMENT_REVIEW.md) is required proportionally. No new code, no auto-activation of architecture work.

## Active discovery topic — RQ-01A.01 round 2
Read [product objective checkpoint](../../product/discovery/RQ-01A-01-product-objective.md). Developer reports management/listing is the enterprise's first delivery demand, planning is strategic and data exists with unknown provenance/quality. Property/workspace management is the leading provisional objective, not approved business scope. Prior H-01 contact candidate and planning-first preference are parked, not deleted. Resolve management vs listing preparation vs public publication before closing topic 01 provisionally. No topic 02 or runtime changes yet.

## RQ-01A.01 — provisional close (round 3)
Latest checkpoint supersedes previous topic-01 working preferences: [product objective](../../product/discovery/RQ-01A-01-product-objective.md). Developer selected management + public listing on BDSPro, explicitly without validated concrete user scenarios. Topic 01 provisionally closed for discovery only; business acceptance and RQ-01 remain OPEN. Next eligible topic is RQ-01A.02 customer segment, not yet started. H-01 contact-first and planning-first are historical alternatives. No runtime work authorized; do not treat the provisional objective as confirmed requirement or release scope.

## Active topic RQ-01A.02 — round 1
[Customer segment checkpoint](../../product/discovery/RQ-01A-02-customer-segments.md) records developer choices and counterexamples. Topic 01 provisionally closed; topic 02 OPEN. No priority segment, payer, access to users, organization policy or property type confirmed. Public browsing with some authenticated actions and three property management cases are proposed, not approved. Resolve bounded actor scenario and access/authority questions in round 2; no topic 03 or runtime work.

## RQ-01A.02 — provisional close (round 2)
[Customer segments](../../product/discovery/RQ-01A-02-customer-segments.md): developer chose broker managing multiple properties as hypothetical first research persona; seeker complementary. Actual priority segment, payer, personal/company scope, offboarding and verification rules remain UNKNOWN. Topic 02 provisionally closed for discovery only; RQ-01 OPEN. Next eligible topic 03 business problem, not started. No schema/auth/organization redesign or other runtime work authorized. Gitea comparator is optional evidence, not a target.

## Active topic RQ-01A.03 — round 1
[Business problem checkpoint](../../product/discovery/RQ-01A-03-business-problems.md) records ten developer choices and legacy/market evidence. No actual pain or first workflow verified. Candidate broker record/retrieve/selectively publish/seeker contact/withdraw, privacy boundary proposed; no import/chat/dedupe/sync mandates. Legacy transaction documentation discrepancy recorded as unverified runtime risk, not fix task. Round 2 only; topic 03 OPEN, no topic 04 or schema/runtime work.

## RQ-01A.03 — provisional close (round 2)
[Business problems](../../product/discovery/RQ-01A-03-business-problems.md): W-03 selected hypothetical workflow (inventory intake, management, public listing, search/contact); developer favors bulk/import or external integration research, both separate/combined creation, public permitted contact method. Actual pain, source/rights/volume, integration modality, combined failure and update/withdrawal policy UNKNOWN. Topic 03 provisionally closed; next eligible topic 04 product scope, not started in this checkpoint. RQ-01 OPEN; no runtime/schema/architecture authorization.

## Active topic RQ-01A.04 — round 1
[Scope checkpoint](../../product/discovery/RQ-01A-04-product-scope.md): W-03 hypothetical, release policies unknown. User/Organization comparison is research, not redesign mandate. Round 2: clarify personal/shared workspace and seeker scope. Topic 04 OPEN; no topic 05 or runtime work.

## RQ-01A.04 — provisional discovery close (round 2; scope UNDECIDED)
[Scope checkpoint](../../product/discovery/RQ-01A-04-product-scope.md): compare individual and enterprise scenarios; actor/scope/resource/authority is research focus. Developer explicitly chose candidate capability map only, no scope approval. Organization representation, collaboration and record-control are distinct hypotheses. Next eligible topic 05 enterprise constraints, not started. RQ-01 OPEN; no implementation, schema or architecture authorization.

## Active topic RQ-01A.05 — round 1
[Enterprise constraints](../../product/discovery/RQ-01A-05-enterprise-constraints.md): developer-reported local-first, low-cost, sole backend test/ops responsibility, unclear scope/delegation and recovery/scale evidence. Legacy failure dependencies and counterexamples pinned; no runtime verification or architecture conclusion. Resolve core-first vs incremental approach, evidence gate and responsibility in round 2. Topic 05 and RQ-01 OPEN.

## RQ-01A.05 — provisional close; RQ-01A synthesis
[Enterprise constraints](../../product/discovery/RQ-01A-05-enterprise-constraints.md) round 2 records evidence-based technical decisions, bounded foundation, separate hands-on study and local synthetic data. Architecture acceptance method still UNKNOWN; proposed practice/evidence protocol only. [RQ-01A synthesis](../../product/discovery/RQ-01A-discovery-summary.md): 5/5 discovery discussions closed provisionally, release scope UNDECIDED, business validation incomplete, RQ-01 OPEN. Next product gate: review one real or explicitly hypothetical workflow and evidence gaps, obtain scoped approval before implementation. Separate optional first learning exercise: reproduce existing CLI test in local environment; not started. No schema/architecture/runtime or release authorization.

## Active technical learning checkpoint — developer takeover
[DEVELOPER_TAKEOVER.md](DEVELOPER_TAKEOVER.md) supersedes prior learning status. AI-authored baseline per developer report; owner is learning existing code from the beginning. WORK-001 build/selected test PASS (reported), WORK-002 deliberate assertion FAIL observed; restoration/retest pending. Next: locate actual CLI output producer and writer path in IDE. No new runtime, architecture, schema or deployment task. RQ-01 remains OPEN.

## Active unified backend development target
[BACKEND_DEVELOPMENT_TARGET.md](BACKEND_DEVELOPMENT_TARGET.md) defines product, engineering and practical developer objectives and proposed BT-01–07 acceptance targets. AI-authored current baseline per owner report; no assumed prior mastery. First proposed slice BE-01A: review CLI config preflight contract and existing source with integrated testing; restore prior deliberate assertion and trace behavior, then decide no change or a meaningful bounded improvement. WORK-002 restoration still pending. RQ-01 remains OPEN, no business scope/architecture/runtime authorization. Supersedes exercise-first sequence.

## Goal-first priority correction
The developer requests goal formation before selecting requirement/design/code tasks. [Backend development target](BACKEND_DEVELOPMENT_TARGET.md) goal-first correction supersedes proposed foundation-first ordering. RQ-01 goal/evidence review is the next decision focus using completed RQ-01A; W-03 remains hypothetical. BE-01A parked candidate pending purpose/priority; prior assertion restore/retest remains unreported. AI-authored baseline, no confirmed product scope, no database/schema/architecture/runtime changes authorized. Integrated testing and practical development continue per approved scope.

## 2026-11-25 — active daily plan
[WORK_PLAN.md](WORK_PLAN.md) is the current queue and handoff. RQ-01B.01 goal/evidence review active, one bounded outcome, W-03 hypothetical, RQ-01 OPEN. Next workdays conditional on gates, no deadlines assumed. AI-authored baseline and reported local checks unchanged; deliberate CLI assertion restoration pending. No new architecture/schema/runtime work.

## RQ-02 — active requirement topic
[Identity/authentication/authorization](../../product/RQ-02-identity-authentication-authorization.md) OPEN, RQ-02A.01 active analysis. User requested this focus; RQ-01B.01 parked without approval. Source facts pinned, requirements proposed, no schema/runtime work. [WORK_PLAN](WORK_PLAN.md) is current task queue.

## 2026-11-26 — IND-01 identity research
S1 scope selection DONE (individual research only). S2 account/domain semantics next; [RQ-02](../../product/RQ-02-identity-authentication-authorization.md) and [WORK_PLAN](WORK_PLAN.md) authoritative for current task. Seven proposed requirements; all acceptance tests NOT RUN, all physical/technical decisions pending. No implementation approval.
