# Requirement and evidence standard

## Requirement discovery
Record actor/business objective, trigger, preconditions, workflow, success and denial outcomes, data/invariants, scope and exclusions, owner and acceptance examples. Mark confirmed/assumption/unknown. For a quality scenario record stimulus, environment, affected component, expected response and measure; don't invent an SLA. Technical choices are not business requirements. Constraints include team capacity, learning, deployment, budget, stack and compatibility. Separate mandatory safety/data/security requirements from preferences and optional improvements.

## Evidence classes
- OWNER CONFIRMED: business decision or reported context; not an independently audited metric.
- SOURCE FACT: exact repo/commit/path/line and caller/callee; proves inspected behavior, not author intent.
- OFFICIAL CONTRACT: versioned specification; not proof of BDSPro conformance.
- HISTORICAL INTENT: dated PR/issue/ADR and matching diff; missing rationale remains unknown.
- CONTROLLED TEST: command, version/environment, inputs, actual output; scope lab vs BDSPro vs production.
- MEASUREMENT: baseline, units, load, environment, sample and results.
- INFERENCE / HYPOTHESIS / NOT INSPECTED / NOT RUN: disclose limits, proposed falsification and next action.

## Decision quality
Compare the same requirement under keep/no change and alternatives. Trace only affected paths but include cross-boundary impact where relevant. Consider failure, learning, development and recurring operational costs, reversibility, migration/security and test isolation. Distinguish an actual incident from a risk that can be reproduced or reasoned about. Lack of traffic is not proof of safety; no test result is not a passing test. Do not score or quantify cost without evidence. Optional reference sources must not dictate target syntax. See [protocol](../research-protocol.md).

## Product evidence and challenge
Use [RQ-01](../../product/RQ-01-product-and-first-workflow.md) and [requirement review](../../product/REQUIREMENT_REVIEW.md). Separate reported business intent, direct user evidence, hypotheses and implementation facts. Record a designated product authority, buyer, end user, data subject, workflow and counterexample. A feature cannot be confirmed by AI, existing code or a schema. Assess anticipated risks by concrete trigger, relevance, impact, cost and revisit signal; protect credible security/data risks proactively.
