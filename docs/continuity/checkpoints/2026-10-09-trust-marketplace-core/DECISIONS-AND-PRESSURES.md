# Decision & pressure register — 2026-10-09

## Governance / status
This new human review **reopens** some assertions marked CLOSED in 2026-10-06 acceptance. Historical acceptance remains evidence of previous reasoning, not permission to implement without reproof. Vocabulary:
- **PRINCIPLE** accepted design intention or risk boundary, not physically enforced yet;
- **LAB-PROVEN** PostgreSQL behavior observed in local disposable schema;
- **PROVISIONAL** preferred option but alternative remains viable;
- **OPEN** not yet justified / requires evidence;
- **DEFER** valuable later but current expense unearned;
- **REJECT** do not add without materially new evidence.

### Invariants / conceptual contracts

| ID | State | Claim | Value / minimum proof |
| --- | --- | --- | --- |
| T-01 | PRINCIPLE | Write Core owns durable facts; read directory/search/map/reputation are derived or qualified facts | Source authority + rebuild/change semantics |
| T-02 | PRINCIPLE | Business Party != Person subtype != Organization subtype != Account authenticator | Preserve identity through changes in credentials, roles and packages |
| C01 | PRINCIPLE, NOT YET DB-PROVEN | Each committed Party maps to exactly one Person or Organization subtype | PK/FK alone do not ensure total/exclusive coverage; experiment enforcement and concurrent writers |
| T-03 | PRINCIPLE | Declaration/request != proof/existence/representative authority != software role | New entity vs existing claim, competing claimants, dispute, review, revocation |
| T-04 | PRINCIPLE | Authentication success != permission to access resource | Actor, represented Party, current membership, scoped rights, owner, policy, assurance, purpose |
| T-05 | PRINCIPLE | Subscription/paid tier != legal rights, verification, trust level or platform privilege | Payment cannot bypass compliance/resource authorization |
| T-06 | PRINCIPLE | Membership != employment contract, legal representation, role grant or identity | Offboarding / role-revocation scenarios |
| T-07 | PRINCIPLE | User/Organization data and CRM tenant scope are confidential by default | No cross-tenant join/access merely because same Party ID; access-control application tests |
| T-08 | PRINCIPLE | Property != parcel != listing != asset != spatial planning feature | Distinct identifiers, lifecycle, provenance, ownership/authority |
| T-09 | PRINCIPLE | GIS geometry match != legal determination; source record != latest approved truth | CRS, temporal versions, confidence, approval and lawful reuse |
| T-10 | PRINCIPLE | Verified identity != scam-free actor or authentic property, and reputation != right to sell | Specific attestations and business evidence |
| T-11 | PRINCIPLE | Event/audit != current business state; privacy limits audit payloads | Who, action, when, subject, scope, evidence, retention |
| T-12 | PRINCIPLE | Earned commission != payable != paid != settled | Contract, legal eligibility, payer/payee, corrections, reconciliation |
| T-13 | PRINCIPLE | Marketing benefit/ranking cannot silently confer protected access or fabricate verified trust | Organic vs paid discovery, ranking version, abuse review |

### Reopened schema & policy decisions

| ID | State | Design issue / candidate | Counterexample / test that can change decision |
| --- | --- | --- | --- |
| D-01 | PROVISIONAL | Personal baseline one Account per Person, multiple authenticators | Enterprise managed identities could need independent Account boundary; distinguish separate IdP assurance from separate BDSPro account. Canonical accounts.person_id not UNIQUE |
| D-02 | OPEN | Identity linking, recovery and contact uniqueness; external_identities by (issuer,subject) | Confirm provider business needs; do not implement Apple Sign-In because old architecture offered it |
| D-03 | OPEN, PRIORITY | Organization registration/claim, proof of legal existence, representative authority, professional qualifications, duplicate handling | Existing entity with competing claimants; new entity lacks organization_id; legal identifiers may not be simple unique tax_code |
| D-04 | OPEN, PRIORITY | Organization ownership table currently targets a membership | Workspace administrator is not legal owner or representative; ensure chosen membership belongs to same organization |
| D-05 | PROVISIONAL | One durable membership per Person–Organization with explicit audit history may be simpler than multiple period rows | Historic return/reinstatement and period-scoped business references may justify episodes; DB-20 range proof not business proof |
| D-06 | OPEN | Multiple authorization roles vs one role, time-bound blocks, restricted context, branch scope | Member stays active after publish role revoked; same-organization integrity for role/membership/branch |
| D-07 | OPEN | Invitation role mandatory vs invitation without role; acceptance and seat assignment | Network invitation to collaborate before entitlement assignment |
| D-08 | DEFER | Employer-managed accounts, SCIM, automatic enterprise provisioning | Client requirement not met by personal Account + Membership + optional organization SSO and assurance |
| D-09 | OPEN | Group vs Community vs Deal ontology; who pays and who participates | Real broker network/team, social content community and legal deal participants are not same |
| D-10 | OPEN | Provider onboarding for owner, authorized representative, broker, company | Ability to draft vs permission to publish vs to contract; verified right to act |
| D-11 | OPEN | Property parcel identity, project/building/apartment, asset vs property, listing publication, property_*_details/type names | Multi-parcel/multi-property, disputed listing authority; avoid premature renaming |
| D-12 | OPEN | CRM contact linking and lead privacy; commercial lead-routing | Same customer appears in multiple tenants; no sharing of internal notes |
| D-13 | OPEN | Commission agreements/earned/payable/settlement, wallet need | Correct payable actor, legal brokerage conditions, recovery and dispute |
| D-14 | PROVISIONAL | Ranking, recognition and benefits are first-class product capabilities | Candidate 60/40 policy not validated; must have sample-size guards, versioning, fairness, grievance, fraud controls |
| D-15 | OPEN | Spatial provenance, versioning, CRS, official data access rights, auction intelligence vs execution | PDF source, stale source, changed effective date, CRS mismatch, no legal authority |
| D-16 | OPEN | CMS, marketing channels, SEO pages and campaign attribution | Marketing results/evidence and privacy rules; don't convert business noun into service |
| D-17 | OPEN | VNeID/eKYC/legal data providers and privacy architecture | Eligibility/contract, source permissions, minimization, security; no assumed government API availability |
| D-18 | OPEN | Infrastructure service boundaries of old tqd-service | Actual tile/render/ingestion/report load must earn separate deployment |
| D-19 | OPEN | Seller achievement benefits, expiry, downgrade/appeal and who pays incentive | Financial caps; no paid trust; must prevent fake reviews/Sybil manipulation |
| D-20 | OPEN | Role of platform admin vs tenant admin and service principals | Platform-level operational privileges must never derive from organization role or commercial tier |

## Working methodology

For each table, before deciding schema:
1. Real actor/event -> purpose/outcome -> right to perform it.
2. One-row assertion and identity, owner of write truth, lifecycle, affected party.
3. Counterexamples including unauthorized access, second applicant, time and concurrency, duplicate event, refund/reversal, outdated government dataset.
4. Alternatives: simplest adequate representation, constraints, service/application policy, audit and evidence.
5. Cost: operational work, query/index/write amplification, migration, support, privacy exposure, incident blast radius.
6. Decision: KEEP/MODIFY/DEFER/REJECT with evidence, scope and revisit trigger.
7. User predicts and types minimal lab SQL; observe real PostgreSQL; **constraint proof != business need != API authorization**.
8. ONLY after entire core review / explicit implementation approval modify canonical schema or production migrations.

Practical tracing prompts:
- Why does the object exist? What problem if omitted? What concrete flow pays for complexity?
- Is the requirement mandatory now, eventually plausible, or speculative?
- Which fact cannot be recomputed? Is it private/regulated? Who may change it?
- Who pays? Who gains visibility/reputation? Who is legally responsible?
- Who can revoke it and what remains after revocation?

## Chosen next pressure

ORG-01: registration vs claim vs transfer of legal organization. C02: request/declaration is not authorization. First prove even Person with authenticated Account and correct tax code cannot automatically own/administer business. Use a dedicated claim request when the canonical Organization already exists; for new legal entity requests avoid a mandatory FK to a non-existent Organization. Check ambiguous/multiple applications and authority/verification before designing approval schema. Distinguish application actor from reviewing actor. No need to implement national eID integration before defining the contract.

## Evidence signposts (comparison, NOT blueprint)

- GitHub Organizations, Enterprise Managed Users, SAML: https://docs.github.com/en/enterprise-cloud@latest/admin/concepts/enterprise-fundamentals/choose-an-enterprise-type
- Entra B2B collaboration: https://learn.microsoft.com/en-us/entra/external-id/what-is-b2b
- HubSpot CRM objects: https://knowledge.hubspot.com/records/understand-objects
- Zillow Premier Agent: https://www.zillow.com/premier-agent/
- eBay Seller Standards: https://www.ebay.com/help/selling/selling-tools/seller-levels-performance-standards?id=4080
- OGC spatial specs: https://www.ogc.org/standards/ogcapi-features/
- Official current laws/regulations **must be revalidated for each application**, especially personal data, electronic identification, real estate brokerage and regulated payments.


## 2026-10-09 evening — Enterprise-first, internal network and provider role refinement

See [Organization/Network/Provider ontology](ORGANIZATION-NETWORK-AND-PROVIDER-ONTOLOGY.md) for full scenarios, real comparisons and model implications.

| ID | State | Decision / pressure |
| --- | --- | --- |
| D-21 | PRINCIPLE | Enterprise-first onboarding is a first-class B2B journey: Organization can invite or provision newcomer who had no consumer BDSPro Account. Provider-managed work identity remains conditional and is NOT automatically required. |
| D-22 | PRINCIPLE | Organization, operational Branch, internal Department/Team, independent/cross-org Group, Community, Channel and Deal are distinct **semantic responsibilities**. A Group is not necessarily smaller than an Organization or a legal child. |
| D-23 | OPEN | Whether Department, Team, Class/cohort and Channel deserve standalone canonical tables; compare branch/role scopes, lead-routing needs, cross-org participation and privacy costs before changing DBML. |
| D-24 | PRINCIPLE | Provider qualification, seller role, human action actor, represented Party/publisher, asset-specific authority and reputation are independent. Person cannot be permanently classified buyer vs seller or broker purely by role field. |
| D-25 | OPEN | Multi-organization affiliation for Group, sponsor attribution, who can publish/share which Listing, network moderation, owner of shared work and cost payer/beneficiary. |
| D-26 | OPEN | Rules to attribute reputation to Person vs Organization vs Team and cross-company deal, including departure/offboarding without transferring private CRM or falsifying achievements. |

**Next gate:** ORG-01 legal organization registration/claim plus enterprise-first onboarding; then ORG-NET-01 intra-org Team, cross-org Group and authorized Listing publication. No production migration or DBML rewrite approved.
