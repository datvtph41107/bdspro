# Database Change Decisions — Business Atlas → V0.4 logical candidates
Date 2026-10-10. **Only logical DBML design, NOT executable DDL, migrations, production authorization or verified PostgreSQL.** All original 160 tables of V0.3 remain historical baseline; any added table is a research candidate subject to field validation.

## 1. Executive decision
The detailed BUSINESS-01/02/03.1..03.5 action contracts revealed **four distinct durable write facts missing from the existing 160-table scope**:
1. `source_intakes`: one ABC-scoped managed intake/observation of a possible property source, not Property, Listing, ownership or published offer.
2. `source_price_reports`: one price statement from a named/known-as source, received/recorded with scope, price basis and provenance; neither a Listing price nor Asset valuation; only append/compensating correction after confirmed.
3. `crm_inquiries`: one meaningful external customer request as recorded by ABC, not a Zalo transcript, mandatory Contact or Deal; claimed acquisition source/channel separated.
4. `crm_followup_tasks`: one outstanding/closed responsible work unit, separate from `crm_activities` historical action facts.
These are **candidate** schema additions because multiple independent business counterexamples justify separate lifecycle/constraints; they are not requirements to immediately deploy all four, and still must compete against a simpler pilot.

## 2. Per candidate one-row and negative examples
### source_intakes
One row = one original captured source case in one Organization's working stock, with original recorder, reported features and optional later link to Property. Privacy scope is ABC, **not automatic permission for every ABC employee**; independent personal workspaces are not automatically solved by setting generic owner_type/id. Unknown area kind stays unknown. Do not UNIQUE category/price/area. Property link does not confer legal title or advertising permission. Require source/org consistency when joined to inquiries and price reports.

### source_price_reports
One row = one price report recorded by a human about the source intake with declared amount/currency, basis, observation time (nullable) and recorded_at. Different sources may contradict each other. No global 'latest official price' flag. Ignore arbitrary server insert ordering when event times conflict. Positive amount and legal currency checked at physical stage, immutable business history with correction flow. No automatic update of `listing_price_terms`.

### crm_inquiries
One row = an inquiry intake event/request in ABC, with recorder, contact channel and optionally a **claimed** acquisition origin/attribution evidence, optional Contact and Source Intake. Contact can be absent for a brief valid inquiry; external Minh has no platform Account. Contact and Intake, when linked, must be same org. Do not dedupe across tenants or by bare phone. Person-level marketing or contact consent is a separate rights question.

### crm_followup_tasks
One row = responsibility for an action within ABC, assigned to an active authorized Membership when committing assignment, with optional source/inquiry, due time, terminal completed vs canceled, version. `OVERDUE` derived, not stored; task completion ≠ successful client response. Same-org composite FK only establishes member belongs to ABC, **not** member active at that time or authorized for the specific resource; runtime/transaction prove the latter. CHECK terminal exclusivity required physically.

## 3. Proposed V0.4 field choices / deliberate omissions
- Scope to Organization **for first pilot**: do not claim support for personal stock/cross-company Group data until custody/transfer/rights design is reviewed separately. This avoids false polymorphic `subject_type+id`.
- Source channel = declared classification, not transcript; avoid storing raw Zalo messages/photos or provider auth tokens by default.
- No `share_kit_snapshots`, `external_postings`, `crm_property_demands`, `cms_articles`, `campaigns`, `consent_records` automatically: each is a **genuine gap/proposed future capacity**, with business use case, legal basis and write semantics to settle before tables.
- No inferred property ownership, listing authorization, actual Facebook post, conversion, real-time feed or commission from any of the four.
- Pricing, dates, transaction intent, source confidence remain qualified assertions rather than generic `verified` status. No new current-price mirror on intake until canonical derivation/concurrency semantics are proved; price reports are historical records.

## 4. Enforcement map (DBML != SQL proof)
| Desired guarantee | Candidate enforcement | Remaining proof |
|---|---|---|
| Same Org source report | `(organization_id,intake_id) -> source_intakes(organization_id,id)` composite FK | null behavior; actual migrations; tenant auth |
| Same Org inquiry/source | composite FK | client purpose + current rights |
| Same Org inquiry/contact | `crm_contacts(organization_id,id)` unique target + composite FK | contact record-level read rights |
| Same Org task/inquiry/source | composite FK | current task delegate permission |
| Same Org assignee | `organization_memberships(organization_id,id)` unique + composite FK | valid membership and role timing |
| Positive prices/areas | CHECK | numeric constraints, correct area basis |
| Complete or cancel, not both | CHECK and transition SQL | replay/concurrent updates |
| Version monotonic | conditional UPDATE / transaction | pgx/sqlc negative concurrency test |
| One user intention saved once | scoped idempotency record + outcome | repeat with same/different request_hash |
| Exact-one Party subtype | deferred trigger/transaction candidate (C01) | DB-21 lab not yet executed |
| PII protected | backend authorization, data minimization, retention, secure storage | cross-tenant, export/media tests |
| Source history immutable | permissions/change API and history pattern | correction event and replay tests |

## 5. Existing V0.3 consistency debt (NOT silently fixed)
`crm_activities`, CRM Opportunity bridges and a number of Org/Team/Group links still have independent FKs; same-org composite safety must be enumerated end-to-end. `appointments` currently assumes external participants already are Person. `crm_opportunities.expected_amount` is not Minh's property budget. `idempotency_records` lacks an explicit stored response/outcome contract. `listing_publishing_authorizations` is Listing+Property scoped and cannot approve draft stock independently. Media rights/provenance and CMS editorial marketing remain open. DBML selected six composite constraints in V0.3 have NOT been official parser or PostgreSQL verified.

## 6. Branch and migration behavior
Design branch may now carry **V0.4 164 candidate tables** (160 V0.3 + 4 documented new responsibilities) if synchronized into DBML/dictionary/README. Table additions are not approved as deployed or mandatory. The old 146-row impact matrix remains a trace to original baseline, so do **not** rewrite it as if original 146 grew. `dev` stays untouched.
Before physical SQL, reconcile actual live `dev` schema, choose smallest subset by BUSINESS-00 evidence, use versioned EXPAND/BACKFILL/COMPARE/SWITCH/CONTRACT and adversarial transactions. Do not stage full 164 migration.

## 7. Open candidate decision register
- G01 personal vs ABC source privacy/sharing/attribution; no first-owner fallback.
- G02 source price current-selection with late reports, multi-source pricing rights.
- G03 identity of external Contact/Inquiry and lawful retention/erasure basis.
- G04 task vs simple `next_action` in early pilot, due_at and notification behavior.
- G05 appointment with unregistered participants and attendance evidence.
- G06 structured demand as distinct lifecycle vs Opportunity attributes.
- G07 share kit export receipt and media before Property.
- G08 CMS/blog/news/SEO editorial and ads spend/conversion; known missing schema family.
- G09 platform moderation / abuse / incident / role separation.
- G10 Organization legal claim, enterprise-first provisioning, Membership episode.
- G11 Deal legal parties/commission earned/payable/paid distinction.
- G12 payment provider reconciliation, invoices, wallet necessity.
- G13 project inventory and developer distribution workflows.
- G14 data rights/version/CRS and geospatial legal caveats.
- G15 reputation policies, revocations and paid ranking boundary.
All items must be marked OPEN/DEFER until shown by evidence, not omitted from the business overview.

## 8. Acceptance status
Semantic reasoning: **REVIEW CANDIDATE**. DBML consistency after editing: check 164 distinct tables, all exactly once in TableGroups, all FK targets exist, composite key targets present; this is NOT DBML parser, DDL or concurrency proof. Implementation state: NOT IMPLEMENTED. Market proof: BUSINESS-00 NOT YET COMPLETED.
