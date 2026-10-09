# BDSPro — Canonical Core V0.3: human operating-model review

**Review date:** 2026-10-09. **Repository:** `datvtph41107/bdspro`. **Branch:** `architecture/canonical-database-final`.  
**Scope:** 146 original tables -> **160 candidate tables** (142 original names unchanged + 4 renamed original responsibilities + 14 candidate additions). **NO production migrations, runtime services or legal accreditation.**

## 0. What was found before editing

At the start of this review, GitHub already contained a V0.2 160-table DBML and a 146-row traceability matrix, but the package entrypoint, dictionary and acceptance still said 146, and the DBML referred to a nonexistent `CORE-REVIEW-V0.2.md`. V0.3 reconciles these **existing authored changes** rather than replacing them or inventing a fresh schema.

Read together:
- [current DBML](BDSPro-CANONICAL-DATABASE.dbml) — candidate logical model;
- [data dictionary](CANONICAL-DATA-DICTIONARY.md) — **one-row** meanings for all 160;
- [original 146-table impact matrix](CORE-IMPACT-MATRIX-V0.2.md) — every baseline table and how the name/meaning changed;
- [directional scenarios and multi-lens vector analysis](CORE-VECTOR-REVIEW-V0.3.md);
- [2026-10-09 business checkpoint](../../continuity/checkpoints/2026-10-09-trust-marketplace-core/README.md);
- [2026-10-06 acceptance](CANONICAL-DATABASE-ACCEPTANCE.md) — earlier baseline, **not** latest unconditional approval.

## 1. Business operating story (not a table catalog)

1. **Person enters to consume**: search for a home, location/price/plan context, find reliable provider, retain privacy. Visitor/user is not permanently `BUYER`.
2. **Person becomes provider**: sells own apartment or acts with explicit owner authorization. Broker/professional qualification differs from self-owner. Draft and Publish have different gates; Account, publisher Party, actual property right and professional capacity are different truths.
3. **Organization enters**: existence of ABC, applicant's right to represent/manage it and qualifications to provide regulated services must be attested separately. Claim request ≠ authority. Two claimants ≠ two legal Organizations.
4. **Enterprise-first member joins**: ABC invites/provisions An even if An never used consumer BDSPro. Independent Account/Person, work access and enterprise SSO/managed identity have different lifecycle/cost. Leaving ABC revokes ABC access, not An's personal identity.
5. **Operating network expands**: ABC has geographic Branch, internal Team, cross-company collaboration Group with XYZ, local-interest Community, optional messaging/distribution Channel. Size is not taxonomy. Group members never automatically see ABC private CRM.
6. **Market action**: human actor publishes for ABC based on valid evidence on a specific Property, customer inquiry becomes tenant-scoped CRM Contact / Opportunity then qualified Deal. Legal contract parties and commission/payee obligations differ from task participants.
7. **Spatial intelligence**: point, parcel polygon, planning dataset approval/version, external auctions, GIS reports; overlay is evidence of intersection, not legal determination. Access/redistribution right of government datasets must be verified.
8. **Trust loops**: accuracy, incidents, source corrections, complaint handling, professional verification, reputation assessments, earned recognition and bounded benefits. Paid promotion ≠ organic relevance or legal approval.

Core economic goal: **trust + provenance + network** produce more qualified demand/supply connection, safer transactions, measurable utility and controllable operating costs. No absolute scam-elimination claim, no presumed access to VNeID/government APIs.

## 2. Mandatory conceptual invariants (not yet all physical)

| Contract | What must be true | Where to prove it |
| --- | --- | --- |
| C01 | Each committed Party has exactly one concrete Person/Organization subtype | DB transaction/constraint experiment; simple PK/FK insufficient |
| C02 | Applicant/requester is not automatically workspace owner or legal representative | Registration/Claim/Approval, role and document evidence |
| A01 | Account login ≠ authorization ≠ current enterprise assurance | API object-level/tenant/IdP policy tests |
| A02 | Organization/Branch/Team/Role relationships agree on owner Organization where relevant | Composite FK or justified transactional check |
| M01 | Person's Group presence ≠ Organization representation ≠ right to disclose other tenant CRM | resource grant, cross-organization negative tests |
| L01 | Listing publisher != owner != professional != human actor. Authorization for each listed Property | Source evidence, publication boundary and period validation |
| P01 | Property != Parcel != Listing != Asset; source geometry != legal title | cadastral source, CRS, time/version, attestation |
| R01 | Verification != rating; insufficient evidence != poor performance; paid tier != trust | policy version, evidence count and anti-fraud |
| F01 | Commission terms != earned != payable != paid != settled | legal obligation, deal/participation consistency, reconciliation |
| G01 | Source data public ≠ commercial redistribution permitted | source contract, purpose, retention and privacy review |

## 3. Why only six composite FKs are added in V0.3

The V0.2 model had independently valid FKs that could still point to **two contradictory parent contexts**. These six composite references are proposed because the negative state is clear in business language:

1. `organization_primary_administrators.(organization_id,membership_id)` must match Membership's Organization: otherwise ABC's primary admin could be XYZ's member.
2. `organization_branches.(organization_id,manager_membership_id)` same issue for branch manager.
3. `organization_invitations.(organization_id,intended_role_id)` cannot invite into ABC using XYZ's Role.
4. `listing_publishing_authorizations.(listing_id,property_id)` must match Listing's included Property: authorization for another property is irrelevant.
5. `deal_commission_earnings.(deal_id,participation_id)` must match Participation's Deal: cannot assign earnings to a stranger from a different Deal.
6. `group_leaderships.(group_id,participation_id)` leader must participate in that Group.

DBML candidate composite FKs require matching composite unique keys on parent (four extra, logically redundant index definitions on `organization_memberships`, `organization_roles`, `deal_participations`, `group_participations`). **Costs:** indexes increase writes/storage and may require backfill/conflict reconciliation. FKs prevent inconsistent committed references, **NOT** legal rights, active membership, valid time, or authority to invoke the API. Their physical deployment still needs PostgreSQL compatibility test and benchmark. We intentionally do not add every possible same-tenant composite now.

## 4. Domains, value, cost and gating

| Domain | User/organization value | Risk and cost | Review gate |
| --- | --- | --- | --- |
| Party/Account/authentication | stable human/business identity, reliable recovery and login | duplicate person, takeover, incorrect identity linking | C01, OIDC issuer+subject, account cardinality test |
| Organization verification/claim | verified company, safe delegation, enterprise-first onboarding | fraudulent claims, review operations, legal data collection | ORG-01: new vs existing company, two claimants, offboarding |
| Branch/Team/Group/Community | internal control + inter-firm network + local demand | privilege bleed, hierarchy overhead, spam and moderation | ORG-NET-01: ABC/XYZ, scoped CRM and shared Listing |
| Property/Listing/Provider | truthful supply with useful publisher credibility | false listings, forged owner grants, expensive manual vetting | marketplace owner vs delegate vs broker publication |
| Project/Asset | long-lived project/portfolio identity | double ownership semantics and incorrect valuations | actual asset/property/project flows |
| Deal/Commission | fair documented transactions and commissions | financial liability, cross-deal corruption, reversals | payer/payee/contract, correct deal participant |
| CRM/Marketing | lead capture/routing, conversion and enterprise ROI | sensitive customer leak, consent/retention and wrong distribution | duplicate customer across tenants, valid lead assignment |
| Subscription/Payment | paid tools/seat limits without role confusion | invoice/settlement reconciliation and wallet compliance | subscriber vs beneficiary and merchant-of-record |
| Community/Communication | acquisition, referrals, content and reputation | stalking/spam, moderation/retention, unlawful public disclosure | report/block/messaging permission |
| Spatial/GIS/Auction | location relevance, official provenance, usable analysis | data-license cost, CRS error and misleading legal maps | official source, version and uncertainty proof |
| Reputation/Ranking/Benefits | encourage high-quality provider behavior | gameable data, fairness, economic credits cost | evidence sampling, appeal, provider cohorts and sponsored labels |
| Operational audit/events | recovery, observability, reconciliation | over-retention, leaked secrets, event duplication | transactional outbox/idempotency and retention |

## 5. Identity and enterprise-managed work: do not conflate

A newcomer may start from **ABC's enterprise invitation** with no earlier BDSPro Account. That justifies invitation/pending activation, not automatically a second employee Account. Three access tiers:
1. Person's BDSPro account + Membership/Role/tenant ACL;
2. Same Account + organization-scoped SSO/step-up assurance;
3. IdP/SCIM managed workplace principal only if client contract or security boundary needs independent provisioning/recovery/offboarding.

For each tier ask who owns credential recovery, who may revoke work access, whose property/work belongs to employer vs person, and how reputation remains attributable without CRM leak. Identity proofing, authentication and federation differ (NIST SP 800-63-4; see links below).

## 6. Four decision statuses, not premature 'final'

- **KEEP** business boundary (e.g., Party vs Account; Property vs Listing; Offer vs Commission).
- **MODIFY** when table/reference semantics contradict an observed actor/scenario; V0.3 composite refs are logical candidates.
- **DEFER** when a domain is plausible but no user, data, economic payer or operational load justifies it yet (full wallet custody, microservices, general hierarchical Departments/Channels, broad Managed Accounts).
- **REJECT** harmful inference (purchase = verification; group role = tenant access; JWT login = current authorization; GDPR-like claims of universal consent without applicable purpose; blockchain = true evidence).

Legacy `v0/v1/v2 CLOSED` means historical design phase completed, **not** legal/business/security review finished. Do not implement all 160 in one migration.

## 7. Confidence, negative tests, operational price

Each change must record: **actor → input → authority/source → durable fact → invariant → decision → downstream projections → reversible failure**.

Prioritize negative tests: two claims for ABC; no prior Account enterprise invite; illegitimate XYZ member as ABC admin; XYX Role on ABC invitation; Group contributor reads ABC CRM; broker without property grant; external Parcel mismatched CRS; stale regulatory dataset; payout participation from other Deal; minimum-sample gaming in Reputation; expired Benefit used; revoked enterprise identity with still-valid BDSPro token.

For each candidate table/entity answer **one row means**, who owns mutable truth, lifecycle, tenant, provenance, retention, query costs, index costs, migration conflict, and who pays. If no real business pressure, DEFER.

## 8. Next implement/review slice

**First:** ORG-01 human review of legal entity creation vs claim vs represented workspace authority and enterprise-first newcomer. Then C02 lab (`organization_claim_requests` for *existing* Organization; registration request has no mandatory `organization_id`). Run PostgreSQL transaction/constraint/security negative tests under appropriate app role, never confuse root psql insertion with permission proof. Preserve DB-01..DB-20 lab as earlier evidence, not production migration.

**Second:** ORG-NET-01 ABC Team/Branch ↔ independent ABC+XYZ Group ↔ CRM object scope and delegated publisher right. Test six composite relations against sample database and concurrent writes before approving DDL.

**Third:** MARKET-01 and R01 evidence/provider activation, then targeted CRM/Deal/Billing/GIS gates based on pressure. User personally authors SQL/Go in lab per `PRACTICE-PROTOCOL.md`; do not auto-rewrite live `dev`.

## Sources (comparison, not legislation or automatic product requirements)

- NIST 800-63-4: https://csrc.nist.gov/pubs/sp/800/63/4/final
- OWASP API1:2023 BOLA: https://api-security.owasp.org/editions/2023/en/0xa1-broken-object-level-authorization/
- SCIM 2.0 RFC 7644: https://www.rfc-editor.org/rfc/rfc7644
- OGC API Features: https://www.ogc.org/standards/ogcapi-features/
- GitHub Teams: https://docs.github.com/en/organizations/organizing-members-into-teams/about-teams
- Microsoft Entra B2B: https://learn.microsoft.com/en-us/entra/external-id/what-is-b2b
- HubSpot CRM access: https://knowledge.hubspot.com/records/assign-access-to-records
- eBay Seller Levels: https://www.ebay.com/help/selling/selling-tools/seller-levels-performance-standards?id=4080

**Legal/source caveat:** Vietnamese personal data/eID/real-estate/land/auction/financial laws and government data licenses must be verified for the exact use case before operational roll-out; neither this checkpoint nor a schema grants government database integration rights.
