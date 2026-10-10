# BDSPro — V0.3 directional-vector matrix: actors, proofs, privacy and cost

**Purpose:** an actionable reasoning map, not a decorative 1,000 arrows and not 1,000 falsely claimed tests.  
**Traceable baseline:** [146-table row-by-row impact mapping](CORE-IMPACT-MATRIX-V0.2.md).  
**Current logical model:** **160** candidate Tables. Ten lenses (Identity I, Authorization A, Scope S, Time T, Evidence E, Privacy P, Spatial D, Money M, Read UX R, Operations O) create **160 × 10 = 1,600 possible review intersections**. These are review opportunities, **NOT 1,600 proven invariants**.

## A. How to read one directional vector

```
REAL ACTOR + NEED
 -> Account / principal and represented Party
 -> Tenant/group/property/deal context
 -> Current rights + proof + purpose + time
 -> DURABLE BUSINESS TRANSITION / authoritative source owner
 -> downstream scoped read, report, map, marketing and economic effects
 -> threat/abuse/revocation/recovery case
 -> cost (FK, lock, index, review staff, permissions, performance, legal duties)
 -> experiment and decision (KEEP/MODIFY/DEFER/REJECT)
```

No edge is justified simply because two tables share a name or ID. A business graph edge can mean current participation, snapshot, evidence, ownership, data sharing, sponsored promotion or derived similarity. These are **not interchangeable**, and some should not become relational FKs.

## B. Concrete multi-actor pressure vectors

| Vector | Who / initial business need | Path: authoritative facts → effect | Critical negative case | Candidate impact / cost question |
| --- | --- | --- | --- | --- |
| V01 Personal discovery | buyer wants apartment near transit | Person/anonymous intent → eligible Listings + Map → lead | sponsored listing falsely promoted as verified | Listings, map read, privacy; what latency/budget? |
| V02 Private owner | property owner wants to lease own house | Person → property/authority → provider capacity → publish | verified Person ≠ proven owner | Person, Property, Listing, verification |
| V03 Broker acting for firm | company broker publishes client home | Account → Org Membership/Role → owner delegation → Property → Listing | Role is correct but mandate revoked yesterday | Authorizations, time gate, legal costs |
| V04 New company | ABC not yet registered on BDSPro | application declaration → legal registry matching → evidence → Organization | approval silently creates unverified entity | registration, verification, duplicate prevention |
| V05 Two claimants | A/B both claim ABC | two claims → reviewer decision → one workspace control | first requester automatically wins | claim, risk operations, uniqueness |
| V06 Enterprise-first staff | ABC brings An with no Account | invitation / SSO → proven Person/Account → Org membership → Team | invite email accidentally bound to another Person | identity/account recovery, seat provisioning |
| V07 Work offboarding | An leaves ABC | revoke active grants/enterprise assurance → retention of ABC records | An still accesses ABC via cached token | membership, Team, session revocation |
| V08 Dual employers | An works with ABC and XYZ | same Person, two memberships → tenant-scoped work | sees XYZ's CRM through shared Party ID | role, record auth, no cross-tenant join |
| V09 Branch manager | ABC HN branch appoints manager | Organization → Branch → matching Membership | manager is XYZ's employee | composite FK and temporal manager validity |
| V10 Team CRM | ABC routes lead to sales team | Team → Org membership → scoped Contact/Opportunity | Team assignment grants all-org CRM | team, CRM, policy + index cost |
| V11 Partner Group | ABC and XYZ sell project jointly | two org memberships in Group → explicit shared objects | group membership exposes all private CRM | group org participation, ACL, grant expiry |
| V12 Local community | neighborhood group shares local info | Community → post/review → discoverability | self-generated spam becomes reputation | moderation, ranking, rights, deletion |
| V13 Group governance | founder leaves shared Group | Group → leadership from valid participation | lead id belongs to another Group | group participation composite FK |
| V14 Property identity | one offer covers 2 parcels | Property ↔ parcel sources → Listing properties | GPS point treated as land title polygon | version/link provenance; CRS and uncertainty |
| V15 Planning notice | new official local plan appears | source license → dataset version → feature → map/report | stale PDF shown as current final legal decision | legal source and temporal data governance |
| V16 External auction | user wants bidding notice map | official notice → publisher/version → pin/report | informational portal interpreted as licensed platform-run auction | auctions vs notification feed |
| V17 Listing publication | ABC authorizes listing of Property X | property included in Listing + authorization → publication | authorization points to unrelated Property Y | composite FK, evidence at publish |
| V18 User inquiry | renter contacts ABC | Listing → inquiry/consent → ABC-scoped CRM | hidden lead shared with XYZ Group by default | privacy, retention, assignment |
| V19 Lead routing | company buys CRM tools | Subscription → beneficiary org → lead routing rules | seat/payment treated as blanket record access | entitlement and authorization distinction |
| V20 Deal participants | ABC+XYZ work on one sale | participation → qualified contracting parties → agreement | group members automatically become legal payees | deal, legal capacity, privacy |
| V21 Commission earning | participant earns commission on Deal A | contract terms → eligible earning → payable/payment | earnings references participation on Deal B | composite FK, reconciliation, idempotency |
| V22 Refund/dispute | customer contests payment | payment attempt/settlement → invoice reconciliation → adjustment | settled_at interpreted as irreversible funds | financial storage, dispute/reversal |
| V23 Reputation evaluation | company seeks earned recognition | valid evidence → versioned policy → assessment | 1/1 sample gets elite badge | cohort/fraud, sample-size threshold |
| V24 Marketing benefits | provider eligible for credit | assessment/plan → bounded benefit → campaign | paying buys legal badge/rank or leaks CRM | cost cap, separation organic/sponsored |
| V25 Managed SSO | corporate client demands higher assurance | current Account auth → Org SSO proof → current membership → resource | IdP disabled but old token accesses CRM | step-up freshness, revocation SLA |
| V26 Fraud claim | forged authorization document | evidence submission → source verification → reviewer/appeal | uploaded document marked legal truth without verification | document privacy, source/auth quality |
| V27 VNeID integration | user consents to legal identity check | eligible provider contract → minimal attestation → Party verification | copied raw ID/biometric for convenience | consent/basis, access, security, vendor contract |
| V28 Network provider transfer | An moves ABC→XYZ | revoke ABC, create XYZ relationship → partition reputation | ABC lead notes copied to personal profile | attributable achievements vs employer records |
| V29 Offline GIS ingestion | province publishes scanned plan | PDF/survey CRS → controlled ingestion → QA → dataset | geometry digitized without tolerances | QA staff, version, correction |
| V30 Incident response | platform employee mistakes tenant ACL | audit/action evidence → quarantine/revoke → notify/restore | log contains citizen identifiers and exposes more | least privilege, encryption, retention |
| V31 CMS/SEO | developer needs legal project landing page | approved public content → indexed page → discovery | draft private property documents exposed publicly | content governance, SEO proof |
| V32 Economic incentives | reward qualified lead contributions | evidence → attribution rules → reward cap → marketing benefit | pays for fake/duplicate lead | anti-Sybil, costs, disputes |

## C. Boundary matrix: each area, core fact and forbidden shortcut

| Boundary | Durable truth | Derived convenience | Forbidden shortcut |
| --- | --- | --- | --- |
| Party | exactly one concrete subtype | Party directory/avatar | two Person/Org rows for same Party |
| Credential | Account/authenticator/federation | login UI | provider email becomes identity |
| Org claims | applicant / organization evidence and decision | verified badge | knowledge of tax code grants admin |
| Org access | membership, role, assurance, resource permission | tenant dashboard | JWT login alone grants CRM |
| Internal Team | assignment to same Organization | team workload/lead routing | Branch and Group both treated as Team |
| Cross-org Group | explicit organization/person collaboration | shared project feed | group join reads tenant CRM |
| Listings | publication episode + property authority | search cards & promoted feed | uploader automatically owner |
| Property/GIS | source-qualified parcel and planning dataset | interactive map overlay | intersection is legal title |
| CRM | Organization-private client relationship | CRM Party lookup | linked_party_id causes cross-tenant read |
| Deal | participants/contractual parties and commercial terms | deal summary | Participation equals legal payee |
| Billing | subscriber/payer, entitlement beneficiary, invoice/payment | quota meter | subscription grants administrative role |
| Reputation | policy version, attestations/assessments and grant | rank/badge | credit card purchases trust score |
| Community | content ownership, moderation action, audience | recommendation feed | public social content implies contact consent |
| Audit/integration | event evidence and delivery state | dashboards | operational subject_kind/id becomes business FK |

## D. Evaluation protocol per selected row in original 146

Open [CORE-IMPACT-MATRIX-V0.2.md](CORE-IMPACT-MATRIX-V0.2.md) for each of the 146 original identities. For each row cross only the lenses that materially apply to the current business scenario. The ten dimensions are:

- I: entity truth / key / Person-Organization/Account distinction
- A: caller, represented Party, delegated authority, resource action
- S: tenant / group / team / branch / data sharing
- T: effective time, offboarding, expiry, supersession
- E: proofs, claims, issuer, disputes and remediation
- P: privacy and legal purpose, retention, disclosure
- D: CRS/geometry/parcel/planning, accuracy and versions
- M: price, subscription, payer/payee, commission and budget
- R: search/map/directory, ranking, response time and UX
- O: migration, transaction, concurrency, indexing, monitoring, recovery

```
CASE-ID | Actor | Business purpose | Protected data | Source fact | Scope
Expected transition | Invalid transition | Current FK/constraint
Alternative A / B | Bought value | Cost/incident blast radius
Single discriminating test | Confidence | Decision | Revisit trigger
```

**Counting rule:** 146×10=1,460 baseline potential intersections. 160×10=1,600 now, but **only documented scenarios and tests count as evidence**. Do not invent 1,600 business invariants or add a generic JSON "vector" table.

## E. Targeted engineering proof sequence

1. **C01:** invalid Party without / with both subtypes at commit, concurrency, SQL migration cost.
2. **ORG-01:** company registration vs existing claim, two claimants, proof validity, authority change, enterprise-first no-Account.
3. **ORG-NET-01:** ABC/XYZ Membership, branch/team composite correctness, Group member denied ABC CRM.
4. **MARKET-01:** owner and authorized broker publish same Listing only with property-scoped mandate; revoke between check and publish.
5. **FIN-01:** cross-Deal participation denied commission; payer/payee and settlement correction.
6. **SPATIAL-01:** original PDF, CRS conversion, overlapping polygon with uncertainty, dataset version/report provenance.
7. **TRUST-01:** insufficient evidence, fraud, appeal, expiry of benefit, fair search ranking & sponsored labels.

Do not start microservices, national integrations, escrow wallets or production billing until real contractual pressure and legal review.

## F. Sources useful for architecture investigation

- https://csrc.nist.gov/pubs/sp/800/63/4/final — identity proofing vs authentication vs federation.
- https://api-security.owasp.org/editions/2023/en/0xa1-broken-object-level-authorization/ — object-level authorization.
- https://www.rfc-editor.org/rfc/rfc7644 — provisioning protocol candidate (not a mandate).
- https://www.ogc.org/standards/ogcapi-features/ — interoperability of spatial features.
- https://docs.github.com/en/organizations/organizing-members-into-teams/about-teams — internal teams and permissions.
- https://learn.microsoft.com/en-us/entra/external-id/what-is-b2b — external collaboration identity.
- https://knowledge.hubspot.com/records/assign-access-to-records — CRM access control patterns.

**This file is a human review instrument.** Security, compliance, legal proof and effectiveness require implementation-specific and jurisdiction-specific validation.
