# Business operating model — the BDSPro story and external comparisons

Status: contextual business architecture checkpoint, 2026-10-09. Distinguish **observed external patterns**, **desired BDSPro capabilities**, and **unvalidated BDSPro business hypotheses**.

## North-star narrative

A person comes to BDSPro searching for an apartment: reliable location, fair information, verified source, planning/map context and safe contact. Their account is an access mechanism, not a permanent "buyer" role. Later that person owns a property and wants to rent/sell; the user can **activate provider capability** subject to the type of activity and evidence. As professional activity grows they may be invited to work with an agency, participate in several groups and community discussions, and collaborate in a deal. Their professional identity should not fragment merely because their workspace context changes.

A company ABC opens its BDSPro presence, proves the legal entity exists and that its applicant is authorized to administer the profile. It buys marketing/CRM/listing tools; invites independent professionals into collaborative work without taking ownership of personal accounts. ABC guards customer contacts and confidential CRM history. An independent professional retains their own network and verified achievements, while business customer data remains appropriately scoped. A community gathers demand/supply by locality; a deal formalizes collaboration, commitments and evidence. Commission recognition follows lawful terms and financial workflow, not generic membership or "reputation points". 

Spatial intelligence provides position, property context, official planning/administrative information, external auction notices and provenance; geometry overlap does **not** certify title or legal use. Recognition and recommendation derive from verifiable, permissible business evidence rather than being bought by subscription. System trust must protect against scams, impersonation, unauthorized access, misleading map overlays and false reviews.

This entire experience should remain discoverable, comprehensible and friendly for low-risk actions while applying stronger controls to high-risk actions.

## Actors and business intent

| Actor | Need and durable distinction |
| --- | --- |
| Buyer/renter/interested person | relevant search, map, privacy-respecting contact; not perpetually "buyer" |
| Property owner / private seller or landlord | provide own property; authority to offer must be evaluated |
| Authorized representative | bounded authority to act for an owner; not necessarily owner |
| Real-estate professional/broker | professional conditions and relationship with operating company where required; not simply a user role |
| Group founder / community steward | build networks, moderation, collaboration; group not automatically legal Party |
| Legally recognized organization | brand, verification, branches, marketing, enterprise CRM and risk-managed membership |
| Developer and business service provider | regulated services/project offering, distributor relations and accountability |
| Deal partner/customer | distinct rights, obligations and privacy domains; may involve Person and Organization |
| Platform operator/admin | governance, disputes, verification and incident response; not equivalent to tenant admin |
| External government/official data authority | authoritative source with legal scope, licensing, temporal validity and correction rights |

## Organization creation/claim lifecycle

- A signed-in account **requests** onboarding for legal entity ABC. Submission proves only a request.
- Verify **existence** (identity/legal status), **authority** (applicant's right to administer/represent), **qualification** (right to perform specific regulated activity), separately.
- An existing ABC may already be registered/claimed by a different person. Duplicate tax code input is not proof of authorization; first request does not win.
- Distinguish legal person identity, claimed/verified profile, organizational access governance, subscription purchaser and beneficial/legal controller.
- Revocation/transfer/dispute/re-verification must not destroy ABC's stable Party ID or historic deals/data.
- Differentiate new entity registration (no canonical organization_id yet) vs claim existing one.
- "Owner" on a software workspace ≠ legal shareholder ≠ statutory representative ≠ billing contact.
- Entity duplicate detection must consider jurisdiction, registry authority, legal identifiers, branches, identity updates and uncertain/unverified declarations before choosing UNIQUE.

## Membership, account and identity management

Three **different** operations: (1) invitation to join; (2) authorization to work and optional paid seat allocation; (3) employer-managed work credential/account. Membership expresses participation. Role assignment expresses scoped software permission. Seat/entitlement expresses commercial capacity.

Baseline: **one personal BDSPro account may participate in many organizations/groups/deals**. Multiple login methods (email/phone/OIDC/passkey) do not mean multiple accounts. Store external identity by issuer + subject, not email alone. An IdP identity should not be linked to same Person by name/email matching alone.

Enterprise alternatives, in ascending cost:
A. personal account + current membership/roles and BDSPro auth;
B. same personal account, **extra organization-specific SSO/security assurance** when accessing ABC resources, with session/token revocation semantics;
C. independently managed enterprise work identity/SCIM only if independent provisioning/offboarding/control requirements are real and B inadequate.
An account's login success is not organization access approval; employee offboarding must revoke company access without necessarily closing personal account. Larger price plan does not require C.

GitHub organizations and Enterprise Managed Users; Microsoft Entra B2B collaboration/enterprise provisioning illustrate different control modes, not mandates to copy. Source pointers:
- https://docs.github.com/en/enterprise-cloud@latest/admin/concepts/enterprise-fundamentals/choose-an-enterprise-type
- https://docs.github.com/en/organizations/managing-membership-in-your-organization
- https://learn.microsoft.com/en-us/entra/external-id/what-is-b2b
- https://learn.microsoft.com/en-us/entra/identity/app-provisioning/how-provisioning-works
- https://openid.net/specs/openid-connect-core-1_0.html

## Community, business segments, and packages

Users can be buyers/owners/providers at different times. Group (task/network cooperation), Community (content/network participation), Organization (legal/business identity), Deal (transactional context) are different facts. Groups/communities do not automatically become Parties or take legal title; their subscription payer may be a Person/Organization, with benefits targeted at that workspace if commercially justified.

Revenue value for companies includes verified brand presence, client intake, marketing distribution, lead assignment/routing, performance analytics, team collaboration, branch scoping and quota/seats. An independent professional needs visibility, good profile and suitable leads. A group/community needs network/moderation/collaboration. Do not classify solely by company size or tax code. Scale, verified legal status, product spend, reputation and access rights are different axes.

CRM chain: Discovery -> Inquiry/Lead -> scoped Contact -> Opportunity -> assignment/activity -> Deal -> outcome/after-sales. A canonical Party link to a CRM contact does NOT make one company able to read another company's CRM. Official source examples:
- https://knowledge.hubspot.com/records/understand-objects
- https://www.zillow.com/premier-agent/

SEO/Marketing/Ads/CMS remain business research pressures (company/group/user public presence, channels, campaigns, attribution), not a mandate to populate new tables immediately. Paid promotion must be identifiable; promotion cannot override authenticity or privacy rules.

## Marketplace provider onboarding (seller journey)

A person can move from ordinary user to owner-seller, authorized representative or licensed/qualified professional, as appropriate. **No second Account and no mutually exclusive PERSON.type=SELLER**. Provider activation is an action-specific capability requiring appropriate verification/authority; draft may have lower requirements than publication or binding deal. The provider's identity != property ownership != listing publisher != right to publish != legal brokerage qualifications. A seller/advertiser badge is not a title deed.

Real-estate-specific legal obligations, including Vietnam brokerage rules, must be validated with counsel and current primary law before operational design.

Marketplace references:
- https://www.ebay.com/help/selling/selling-tools/seller-levels-performance-standards?id=4080
- https://www.airbnb.com/help/article/828
- https://www.airbnb.com/help/article/2673

## Recognition, ranking and benefits (FIRST-CLASS PLANNED CAPABILITY)

Keep separate:
1. **Verification badges:** what precise aspect/authority was attested, by whom, current validity. A verified person is not a verified property.
2. **Provider quality reputation:** scoped period, evidence count, dispute/anti-fraud controls, repeatable calculation policy version.
3. **Seller/provider achievements:** program eligibility, expiry, revocation/appeals, per-category comparability; little evidence != negative reputation.
4. **Ranking/recommendation:** valid item filter -> customer relevance (location/budget/requirements) -> item/source quality -> limited provider signals. No unconditional rank purchase.
5. **Benefits:** credits, tools, exposure opportunities or campaign eligibility as explicit bounded grants, not privileged API/security/legal access.
6. **Paid advertising:** disclosed paid placements, not fabricated verification or organic ranking.

Candidate LAB policy only (NOT launched/approved):
- 180-day evaluation window; normalized Listing quality Q and valid-inquiry timely-response R.
- Trial S = round(100 * (0.6 Q + 0.4 R)).
- Trial Established: S>=80, >=5 rated listings, >=10 valid inquiries, no confirmed critical breach.
- Trial Distinguished: S>=92, >=15 rated listings, >=30 valid inquiries, no confirmed critical breach.
- B with 1/1 quality and 1/1 response should return INSUFFICIENT_EVIDENCE, not automatically highest rank. A 9/10 and 16/20 yields 86 in experimental formula; C 100 with confirmed serious breach gets no elite benefit.
- Trial discovery Rank = 0.60 relevance + 0.25 listing quality + 0.15 normalized provider quality, NOT a production algorithm.
- Check cohort comparability, sample sizes, anti-Sybil, self-dealing, appeal, audit, drift, measurement, new-seller fairness and incentives for gaming. No identity/authorization derived from score. Cost of credits and fake engagement needs financial caps.

Sources for investigation: eBay Seller Levels, Airbnb Superhost; not a claim of validated BDSPro threshold:
- https://www.ebay.com/help/selling/selling-tools/seller-levels-performance-standards?id=4080
- https://www.airbnb.com/help/article/829

## Spatial intelligence and tqd-service

Legacy `datvtph41107/bdspro-backend/tqd-service/README.md` states planning, spatial, discovery, projections, report generation, usage/quota and PMTiles mode; PostgreSQL/PostGIS durable data, Redis runtime projection. It is evidence of real prior capability but NOT new service topology.

Distinguish:
- basemap, administrative boundary, cadastral parcel, planning/land-use dataset, property/listing, external auction notice, map projection, report;
- Point location vs Polygon parcel, map overlay vs legal conclusion, effective/legal time vs ingestion time, CRS transformation vs setting SRID;
- government/official source authority, access/license/reuse right, decision document, approval/replacement/version, confidence and precision; some localities have PDF only, missing or stale data;
- external auction intelligence vs actual licensed auction execution;
- stable property_id != cadastral parcel_id; splitting/merging of parcels and changes of administrative area must be modeled separately.

Potential pipeline: source -> lawful ingestion -> validation/version/provenance -> spatial core -> tile/map/search/report projections -> property/CRM/deal analytics, respecting data rights and privacy. Do not fabricate government data APIs, claim VNeID integration ready, or equate public publication with free commercial redistribution. Research OGC Features/Tiles:
- https://www.ogc.org/standards/ogcapi-features/
- https://www.ogc.org/standards/ogcapi-tiles/

## Financial responsibility

Subscription, advertising fees, brokerage commission, referral fee, invoice/payment/settlement are distinct. A deal/participation does not establish legal entitlement to commission. Terms -> eligible earned amount -> approved payable -> actual payment -> reconciliation (and adjustment/dispute). "Earned" != "Paid"; "paid" != "reconciled". Buyer and broker protection requires correct attribution, legal compliance and verified evidence; commission percentages are not a reputation or growth reward. Stripe Connect illustrates specialized platform transfer/reconciliation complexity, NOT Vietnam regulatory authorization.

## Trust, privacy, KYC and government links

National-scale ambition demands policy review, incident response, moderation, anti-fraud, provenance and right-sized assurance. VNeID/eKYC may become verified identity sources through authorized integration; no presumption of public API/contract/access. Identity proofing helps mitigate impersonation but cannot prove property title or eliminate scams. Minimize collection of national IDs/biometrics. Verification, scope, effective period, issuer, proof reference and revocation are distinct. Privacy must constrain CRM sharing, marketing/recommendation and visibility; audit should not leak sensitive evidence.

Vietnamese law/regulation to reassess against current consolidated texts at design time: Law 91/2025/QH15 on personal data protection, Decree 356/2025/NĐ-CP, Decree 69/2024/NĐ-CP on electronic identification and authentication, Land Law 2024, Real Estate Business Law 2023, auction and financial regulations. Do NOT encode legal conclusions based solely on this checkpoint.

## What external evidence proves — and does not

- GitHub/Entra prove **possible identity management patterns**; not that BDSPro needs managed accounts.
- Zillow/HubSpot show CRM, organization/customer/distribution models; not that BDSPro needs their service boundary.
- Airbnb/eBay show seller quality programs; not that trial thresholds are suitable locally.
- Government portals prove publication processes exist; not that BDSPro has access, license, complete or legally dispositive data.
- PostgreSQL lab proves SQL constraint semantics; not business demand or application authorization.

## High-value future metrics (hypotheses)

Genuine inquiry-to-response, lead qualification, matching usefulness, conversion/retention, data completeness and correction rate, claim fraud rate, cross-tenant incident rate, false verification decisions, dispute handling time, contribution evidence quality, time to revoke access, and financial reconciliation accuracy. Optimize for meaningful outcomes without amplifying unfairness, privacy leakage or spam.
