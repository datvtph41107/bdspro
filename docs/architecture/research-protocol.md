# Investigation protocol — one problem, one decision
1. Identify the business actor, workflow, expected result, invariant and acceptance example. Classify owner-confirmed, observed, assumed or unknown. If a critical requirement is missing, ask rather than invent it.
2. Pin BDSPro baseline SHA and trace the relevant execution path, data, ownership, errors, lifecycle, security and callers. Look for actual history and existing tests. State search scope and gaps.
3. Establish the Go/stack mechanism using versioned official sources. Where useful, run the smallest standalone example; separate it from integration evidence.
4. Reproduce actual friction or a credible risk. State trigger, environment, affected component, expected response and measurable acceptance. A preventive requirement may suffice; a hypothetical incident must not be reported as real.
5. Compare keep/no-change, minimal change and a meaningful alternative when appropriate. Use optional external sources (Gitea, Anthdm, etc.) at pinned revisions; distinguish implementation from author intent and before/after history.
6. Evaluate learning, developer change surface, debugging/testing, deployment/recovery, recurring operational cost, security and reversibility. Use counts/measurements if available; estimates and unknowns must be labeled.
7. Decide OPEN/PROPOSED/ACCEPTED/REJECTED with owner and scope. No architecture change without bounded authorization.
8. Implement focused changes only; before/after code, regression tests, commands/output, risk, rollback, CI at final SHA. Distinguish IMPLEMENTED from VERIFIED. No merge without approval.
9. Update case/ADR when appropriate and handoff; observe actual behavior and reopen only on new evidence or requirements. Avoid documentation ceremonies for trivial changes.

See [case template](research-governance/DESIGN_CASE_TEMPLATE.md), [evidence standard](research-governance/REQUIREMENTS_AND_EVIDENCE.md), [current phase](research-governance/CURRENT_PHASE.md).

## Longitudinal follow-through
For nontrivial changes use [Engineering lifecycle](engineering-lifecycle.md): record the adopted mechanism, baseline and adoption trigger; track the actual local/CI/staging/release/production stage and observations at each applicable stage. Compare upstream history by finding its own problem, before/after commit, rationale, test and limits; if missing, label unknown. A repo snapshot is not proof of an earlier design or production result. Review real developer friction or runtime signals before proposing the next evolution. Not every fix needs upstream research, an ADR or production release.
