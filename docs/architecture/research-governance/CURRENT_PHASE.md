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
