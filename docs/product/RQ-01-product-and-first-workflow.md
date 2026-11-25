# RQ-01 — Identify the first customer and valuable workflow
Status OPEN / discovery. 2026-11-25. Owner reported team: one frontend, one backend; backend REST API experience, learning Go and testing/production ownership; no supplied traffic/incident baseline; business requirements not yet sufficiently clarified. User reports taking backend responsibility and choosing monolith to reduce distributed complexity. Treat these as owner report, not audited cost savings or claims about other individuals.

## Decision 0: product boundary before features
Separate BDSPro operator, paying/approving customer, actual user, and real-estate contact/data subject. The operator might sell software or operate a marketplace; customer may be an individual broker, brokerage, developer/investment group, or consumer. None is confirmed. Legal entity, workspace/tenant, organization, account, person, employee/membership and contact are different concepts; don't equate them or require all as tables. CRM may be brokerage workflow or the operator's own sales/subscription management; do not merge these without a business decision.

| Candidate segment | Job hypothesis | Evidence needed |
| --- | --- | --- |
| Individual broker | Track inquiries, properties, follow-up | real broker workflow, current tools, frequency, pilot |
| Brokerage/team | Shared lead assignment and continuity when employee leaves | manager and staff examples, visibility and ownership |
| Buyer/seller/renter | Search listings or inspect planning data | real search journey, data supply, accuracy expectations |
| Developer/investment group | Coordinate deals, inventory, investments/contracts | actual deal and approvals, money/data responsibility |
| BDSPro operator | Manage subscription customers, support and sales | actual sales model and operator workflow |

## First requirement (discovery gate, not implementation)
RQ-01: Establish ONE target segment and ONE end-to-end job with a real business decision maker and prospective user. Record a recent real example (sanitized), current process/tool, pain and consequence, frequency/volume or unknown, buyer and direct user, desired outcome, data ownership, success/denial/failure examples, and evidence of willingness to trial or other agreed validation. No artificial SLA, customer claim or invented interview. If no real customer yet, clearly state pre-market hypothesis and arrange validation through the owner; don't label it confirmed. Business acceptance requires designated product authority, not AI.

## Proposed first vertical slice to challenge (only if brokerage job confirmed)
H-01: A brokerage staff member records a buyer inquiry and a responsible staff member can retrieve it for follow-up in the authorized workspace. Value hypothesis: no loss of a contact/follow-up during handover; measure actual current missed follow-ups and retrieval time only if available, agree pilot acceptance before implementation.
- Trigger: inquiry from phone/chat/in-person; data fields minimal and justified, lawful handling confirmed separately.
- Actor: authorized staff; manager, external person and platform operator are distinct. Can an individual broker use it without an organization? Does a contact ever log in? Who owns record after staff departure? Resolve before choosing identity/tenant schema.
- Illustrative acceptance: authorized staff creates contact with agreed fields; authorized actor retrieves it; an unrelated tenant/unauthorized actor cannot see or change it; rejected input does not create a record; after acknowledged success record persists through restart; failed DB write returns controlled failure without false success; decide duplicate and correction behavior, audit/retention/deletion based on policy. All are PROPOSED, not customer-approved.
- Dependencies only as needed: identity/access, data scope, minimal contact persistence, API/UX and diagnostics, test/deployment/backup. Existing team API is not a prerequisite by default. If single-user first, no automatic organization/team; if multi-tenant, server-side isolation and access controls required from first exposure. Existing protections preserved.
- Exclude until separately justified: generic CRM engine, arbitrary role designer, payment, AI, full pipeline/deals, gRPC/service split, queue, realtime, hot reload. Existing capabilities not deleted by this case.
- Counterexample: if primary product is planning lookup for consumers, contact CRM is wrong first slice; choose a real planning-search journey instead and validate data accuracy/provenance and access. If investment deal management is primary, clarify financial workflow and invariants first.

## Stakeholder interview (answer with recent example, not a desired feature list)
1. Which product is being sold first: brokerage work management, independent broker tools, property/marketplace search, planning data, deal/investment, or another job? Who pays, who uses, and is there a real pilot?
2. Walk through the last actual inquiry/property search/deal from trigger to outcome. Which actor uses what tool? Where is delay, error, missed work, or cost?
3. Who controls customer data, who can see/edit/share/export/delete, and what happens on employee departure, account loss, company split or closure? Is personal use allowed? Can one person join multiple companies? Does a contact need login?
4. What is the smallest outcome a user would accept and use? Which actions must be denied? What happens when DB/network/provider is unavailable? What existing API/data compatibility must be preserved?
5. What actual volume, business hours, outage/data-loss tolerance, hosting/budget, retention, support, backup/recovery and privacy constraints are known? Mark unknowns; no speculative scale.
6. Who approves product scope and policy? Who will test UX/API, approve backend risk, authorize release and handle support?

## Exit and handoff
Owner confirms segment, buyer/user, workflow and acceptance examples (or explicitly rejects and replaces H-01); product decision authority and evidence recorded; risks and minimal technical prerequisites identified; compare current behavior at pinned SHA. Then open only the chosen scoped implementation/design case. No runtime edits, migration or deployment authorized by discovery. See [baseline](SOURCE_BASELINE.md) and [review](REQUIREMENT_REVIEW.md).

## RQ-01A.01 provisional objective checkpoint
See [topic 01 round 3](discovery/RQ-01A-01-product-objective.md). Current developer-selected hypothesis is property management plus public listing on BDSPro, not H-01 contact-first or planning-first. Prior candidate remains historical; no workflow or detailed release scope accepted. Topic 01 provisionally closed, next eligible topic 02 customer segment; RQ-01 remains OPEN. Validate buyer, user, real job, publication and data ownership policies before technical design.
