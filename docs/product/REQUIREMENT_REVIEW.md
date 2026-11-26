# Requirement challenge and justified anticipation
Status: working review policy, owner direction 2026-11-25. No assumption becomes accepted requirement through AI/doc repetition.

## Review each nontrivial request
- Business: actor, real job, problem evidence (interview, current workflow, support or usage), expected outcome, buyer, product authority, priority and exclusions. A requested solution is not automatically a requirement.
- Semantics: terms, data owner, action/permission, lifecycle, success/deny/error examples; identify conflicting stakeholder needs and ask targeted questions.
- Operational: how to deploy, observe, diagnose, recover, secure, support and retire it; specify environment, stimulus, affected asset, response, measurable acceptance or UNKNOWN. Estimate learning, implementation, recurring maintenance and support separately. Never invent customer, usage, incidents, SLA, or developer competence.
- Challenge: alternative including no build/no change, smallest useful workflow, what breaks without feature, credible counterexample, evidence missing, reversibility. AI must raise contradictions and may recommend DEFER; AI is not an independent reviewer or acceptance authority.
- Record: OPEN / PROPOSED / CONFIRMED by named product authority (or REJECTED / DEFERRED); technical ADR separately, implemented/verified/deployed separately. One short record proportional to risk. Product owner decides what/why; backend owner decides how with review; frontend validates UX; actual users validate outcome.

## Replace generic foresee-everything with bounded anticipation
For each concern record trigger, likelihood evidence or unknown, impact, relation to the confirmed workflow, cheapest sufficient mitigation, test, cost of delaying, and threshold/event for revisiting. Protect credible high-impact security/data/privacy risks before exposure; do not wait for harm. Forecast capacity when justified by a real commitment, measured baseline or explicit bounded assumption.
| Concern | Act when | Initial response candidate / revisit |
| --- | --- | --- |
| Cross-tenant data exposure | multiple independent owners in scope | enforce server-side scope and negative tests; before release |
| Employee departure | shared company data and membership confirmed | revoke authorized access, preserve approved company ownership; verify policy |
| Lost acknowledged writes | durable data promised | transaction semantics, backups and restore drill as appropriate |
| CRM independent service | documented independent scaling/deploy/isolation constraint | keep module boundary initially if sufficient; revisit on measurements or concrete delivery requirement |
| Queue | confirmed asynchronous/durable delivery requirement | direct execution or DB transaction where sufficient; revisit on latency/delivery failure evidence |
| DI/global config | real dependency/test/lifecycle friction or credible invariant | keep existing design if sufficient; compare narrow DI/parameter/package access on same operation |
No arbitrary volume threshold or capacity number. A useful modular boundary need not be a remote service. No generic future-proofing architecture, no blanket rewrite.

## Goal-to-design trace (active refinement)
Before reviewing implementation, state the goal and the problem/outcome for the user, operator or developer; separate reported facts from hypotheses, requested solutions and existing code. Define behavior examples including failure/deny, authority, business rules and data meanings. For data-bearing work, derive conceptual entities/relationships and invariants, then logical contracts, then compare physical schema, constraints, access patterns, transactions and migration alternatives with existing source. No table/entity or architecture is mandated by a name in legacy/current code. Choose smallest sufficient change or no change with explicit acceptance, operational cost and risks. Learn and test while doing work; revise upstream assumptions when evidence contradicts them. For unknown business demand, obtain evidence or explicit permission for a bounded hypothesis experiment, not an assumed release. For independent safety/compatibility/engineering needs, justify goal and scope without fabricating customer requirements. One concise trace per material decision, no mandatory document cascade.
