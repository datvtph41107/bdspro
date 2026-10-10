# BDSPro — BUSINESS ATLAS V1 · toàn cảnh nghiệp vụ & áp lực hệ thống
Date: 2026-10-10 (Asia/Bangkok). State: HUMAN ANALYSIS / DESIGN HYPOTHESES; **NOT observed customer validation, legal opinion, SQL proof or implementation approval**.
Repository: datvtph41107/bdspro; design branch `architecture/canonical-database-final`; live application work belongs to `dev`.

## 0. Why this atlas exists
Restore the entire business causal reasoning across chats, not just table names. Prior work: 2026-10-06 Parties/C01; 2026-10-09 V0.3 160 candidate tables and 32 directional vectors; 2026-10-09 BUSINESS-00 reality gate; 2026-10-10 BUSINESS-01..03.5 source→share→inquiry. This atlas **covers** all remaining business domains at scenario/contract/pressure level; it does **not** purport to exhaust every legal rule, invariant, edge or test. Detail and subsequent proof remain iterative. Historical closures can be reopened by counterexamples.

**Business reality:** Agents currently prospect via Facebook groups, personal/group Zalo, phone, field visits, portals and spreadsheets. BDSPro initially has no native audience. The first product must be independently useful with zero native leads. Candidate wedge = Stock + Share Kit + Lead Follow-up for a small active team; competing alternatives = Stock-only, Follow-up-only, and existing CRM. Test user adoption, time and error reduction, privacy and willingness to pay. No premature auto-posting/scraping personal messages. Manager dashboards are not a substitute for worker value.

**Standing causal chain:** observed job and actors → who pays/benefits → context and authority → what is known vs merely claimed → meaningful action → one-row durable fact → lifecycle/state → invariant and failure → minimal transaction → read projection → operating/security/legal cost → field/SQL proof → KEEP/MODIFY/DEFER/REJECT. Split write truth from read convenience; one Party exactly one Person or Organization is proposed C01 **not yet DB-proven**.

**Four truth grades:** CLAIMED (reported by a person); SOURCE-CHECKED (reconfirmed with identified source); EVIDENCE-ASSESSED (reviewer reviewed specified evidence); AUTHORITATIVE-WITHIN-SCOPE (legally/contractually recognized for specified decision). No boolean `verified` encompassing identity, ownership, professional eligibility, price, listing and map data. These labels are design vocabulary, not automatic guarantees.

## 1. Actors, contexts and separation of responsibility
A human may act as buyer, renter, owner, landlord, broker, employee, group collaborator, buyer's representative, tenant admin and seller **at different times**. They are not permanent Person types. External visitor need not have Account. Legal Organization and workspace administrator are distinct. Developer/project seller, independent intermediary, enterprise employee, team lead, partner organization, channel operator, content editor, platform reviewer, bank/payment provider, cadastral source and AI/job worker are distinct capacities. Party ≠ Account ≠ Membership ≠ Role ≠ current assignment ≠ representation grant ≠ licensed profession ≠ ownership ≠ package entitlements. Resource rights require actor, capacity, tenant, resource, purpose, time, evidence and current policy. Organization, Branch, Team/Department, independent Group, Community, Channel and Deal are not synonymous. No cross-tenant CRM access merely from Group or shared Party ID.

**Field research:** record separately buyer (budget owner), agent (daily operator), manager, end customer and platform. A manager-only improvement that demands extra entry from agents can lower data reliability. Do not hardcode suggested 5 teams/15–25 agents or 2–4 weeks as proven study results.

## 2. Identity, authentication and account recovery
**Stories:** private buyer browses anonymously; chooses Account; later becomes seller; loses phone; enterprise ABC invites An before consumer signup; An holds roles at ABC and XYZ; employee offboards while personal profile persists; malicious claimant attempts recovery.
**Actions:** register/prove contact, authenticate, link federated issuer+subject, enroll/revoke authenticator, establish/revoke session, recover with assurance, link to Person without merging by email alone.
**Durable facts:** Party/Person/Organization; Account → Person; authenticated credentials and sessions are digital, not social identity. Person may exist without Account. Current `accounts.person_id` is not UNIQUE: account cardinality and managed enterprise IdP remain explicit review.
**Threat:** email recycling, SIM swap, credential stuffing, ambiguous identity merges, stale session, impersonation. Recovery and auth audit must limit secrets; passkeys/SSO/service principal only with justified use.
**Schema naming/open:** older `external_identities`, `passkeys`, `sessions`, `service_accounts`/keys in V0.3 are not policy closure; alternatives `account_login_connections`, `account_passkeys`, `account_sessions`, `api_clients`/credentials need proven scope before rename.
**Gate:** C01 subtype enforcement in concurrent PostgreSQL writes, recovery test, no automatic org data transfer; do not build every auth mechanism in first slice.

## 3. Organization onboarding, claim and authority
**Stories:** ABC legally exists but has no platform profile; two applicants claim ABC; founder requests create while employee requests access; enterprise invites An; false agency tries to impersonate brand.
**Actions:** registration request vs existing-entity claim; evidence intake; reviewer decision; grant workspace administration; activate provider only after separate business eligibility; invite and provision staff.
**Truth:** legal entity existence, applicant-to-org representation, platform primary administrator, permission roles, professional qualification and published seller authority are separate, revocable/time scoped.
**Failure:** first claimer steals brand; tax_code used as universal immutable identity; paid tier bypasses legal verification; manager Membership from wrong organization. Candidate tables `organization_registration_requests`, `organization_claim_requests`, `organization_representation_authorizations`, `party_verifications`, `organization_primary_administrators`.
**Gate:** review jurisdiction, evidence, disputes, SLA and legal costs; enterprise-first onboarding without forcing a prior personal consumer package. ORG-01 not yet lab-proven.

## 4. Teams, branches, groups, communities and sharing
**Story:** ABC has Hanoi Branch and inside-sales Team; An belongs to functional Team; ABC collaborates with XYZ through cross-company Group; community discusses local housing.
**Actions:** scoped invitations, membership, assignment, removal, group participation, moderation, content/publication, controlled cross-entity sharing.
**Truth:** Branch = internal operational/geographic unit; Team = functional org unit; Group = collaboration across independent actors; Community = audience/content; Channel = delivery surface. A Group is not naturally a smaller Organization; an employee is not automatically a Group member.
**Failures:** implicit CRM sharing, org role reused as group role, group ownership confused with company equity, staff leaving one org accidentally losing independent group history.
**Gate:** prove source/buyer cross-company exchange demand, contractual rights, safe opt-in, billing incentives before implementing Group/Community networks.

## 5. Property, parcel, project and asset
**Story:** An hears 'house 75 m²' through Zalo with ambiguous area basis; another broker supplies conflicting description; source may cover multiple cadastral parcels; project building contains apartments; investor tracks rental yield.
**Actions:** source intake, reconcile possible duplicates, establish stable property subject, attach qualified address/area/media/evidence, link to parcel/project, eventually portfolio/asset operations.
**Truth:** Source Intake = report by one source; Property = durable real-estate subject (not necessarily legally proven title); cadastral parcel = jurisdiction/version-qualified land record; Project/Tower/Unit separate; Asset = economic portfolio treatment. Property may be linked to parcel(s); a parcel overlay never proves ownership.
**Failure:** 75 m² assumed land area; dedupe by price/area; Property auto-created from every forwarded Zalo; publicizing full address or owner phone; stale price overwritten globally.
**Gate:** distinguish reported price from listing price, valuation and deal amounts; `property_categories` taxonomy not statutory land-use type; retain house/apartment subtype under renamed `property_houses`, `property_apartments`.

## 6. Source intake, price confirmation and inventory operations — BUSINESS-01 / 03.2 / 03.3
**Story:** An gets house 75 m², 4.2bn VND via Zalo, in ABC context; calls S1 who reports 4.1bn; Bình hears 4.3bn from S2; multiple source claims may refer to one property.
**Actions:** quick capture, assign confirm work, append meaningful price observation, reconcile source/Property with evidence, identify staleness, prepare stock view.
**Truth:** one intake = managed lead on possible stock with custodian tenant, recorder, declared channel and facts; one source price report = a statement from known source about price at known/unknown event time, captured by a person; current display may derive from latest admissible report but not necessarily latest server insertion; different source reports may coexist.
**Failure:** price S1 overwrites S2, source confirmation equated to legal authority, duplicate source merges automatically, two updates lost, stale intake becomes verified Listing. Save-source and relevant task atomic where UI commits both; idempotency key differs from semantic dedupe. No auto eKYC for private notes. A draft/source may exist without Property or Listing.

## 7. Provider, seller authorization, Listing and publication
**Story:** owner directly sells house; broker acts for owner on behalf of ABC; representative grant revoked; Listing concerns multiple Properties; misleading ad removed.
**Actions:** provider enrollment conditional on capacity; validate source and representational grant; draft, review, publish, revise, expire/retract; offer and acceptance with evidence.
**Truth:** verified Party ≠ property owner ≠ active authorization to advertise specific Property ≠ current brokerage qualification; Listing = market exposure of one publisher; `listing_publications` = platform publication episodes; `listing_price_terms` = listing-specific offers, **not stock report**; publish-to-external social platform not automatically controlled.
**Failure:** hidden expired authority, publisher differs from authorized Party, price broadcast globally, 'verified' badge inferred from paid plan. Publishing gate must recheck authority and version. Marketplace traffic initially zero.
**Gate:** validate rights against Vietnam law, business type and jurisdiction before regulated publication. Prioritize internal stock/read convenience before public marketplace.

## 8. Share Kit, external channels, marketing CMS and SEO
**Story:** An produces copy/image pack then manually posts to Facebook/Zalo; price changes after export; marketing staff prepares articles/blog/news/SEO landing pages and paid campaigns; company asks which spend produced leads.
**Actions:** prepare from allowed facts; proofread; export with policy check; optionally record user-reported post; distinguish independently observed delivery; editorial draft→review→schedule→publish→correct/archive; manage canonical slug, redirects, SEO metadata, rights, author/reviewer, ad sponsorship disclosure.
**Truth:** preview ≠ export ≠ user-reported external post ≠ verified external publish ≠ view/click ≠ inquiry ≠ sale. Source snapshot should be version aware. A CMS article is neither Listing nor Community Post; ad campaign spend is neither organic reputation nor entitlement.
**Failure:** AI invents ownership/legal claim, photos disclose documents, revoked rights cannot retract downloaded copies, incorrect redirect/SEO index, ad report falsely attributes customer. No unofficial scraping or automatic personal chat access. Meta Groups API deprecated; Zalo OA permissions do not grant personal Zalo access.
**Schema:** no adequate first-class CMS/marketing editorial lifecycle in current 160 model. Record separate **CMS/SEO and Marketing/Ads GAP**, not auto-add many tables before editor/customer demand. Do not demand Listing to render a private draft.

## 9. Inquiry, Contact, Demand, Follow-up, Opportunity and appointments — BUSINESS-02 / 03.5 onward
**Story:** Minh sees an authorized Facebook post, messages An on personal Zalo and wants weekend visit; An must confirm stock; Bình may receive same person's later inquiry.
**Actions:** respond first; optionally capture inquiry in one small step; link allowed source/Contact; choose next action and due time; assign or hand off; schedule tentative vs agreed appointment; record visit outcome; qualify changing demand; promote to Opportunity/Deal only if real business transition.
**Truth:** acquisition source as customer/agent claim ≠ contact channel Zalo ≠ capture mechanism BDSPro. Inquiry = one meaningful request, not each chat bubble; Contact = tenant CRM relationship record, external visitor need not have platform Account/Person; demand = budget/area/transaction preferences, not `crm_opportunities.expected_amount` forecast; work item = current responsibility, Activity = what actually happened; appointment proposed/confirmed/cancelled/completed differ.
**Failure:** all messages become contacts/opportunities; CRM duplicates across tenants deduped globally; every open task auto-notified; attempt to see XYZ Contact through ABC matching; personal Zalo mined without permissions. Employee offboarding revokes ABC access but preserves legitimate ABC business history, not private contacts.
**Gap:** stock intake, reported prices, external Inquiry, Follow-up, structured Property Demand have clearer one-row semantics than existing generic Activity. Physical table count/gate remains separate decision.

## 10. Demand matching, recommendations and quality
**Story:** Minh budget <= 4.5bn, house 70–85 m²; some sources stale or without authorization.
**Actions:** first filter authorized visibility and acceptable freshness, then deterministic area/type/budget features, then explain an action: confirm source, show candidate, ask preferences, don't spam.
**Truth:** match is derived/hypothesis, not permission, ownership or sale. Eligibility→matching→next-best-action. No magic AI win-rate without verified outcome samples. No cross-tenant ANN/vector-cache leak.
**Failure:** incomplete data masquerades high confidence; old sources treated as ready; more notifications harm clients; optimizing clicks rather than visits.
**Gate:** compare against manual matching under same marketing/spend; track verified useful actions, not vanity 99% scores.

## 11. Deal, contract, co-brokerage and commissions
**Story:** qualified viewing leads to negotiation; ABC and XYZ collaborate; parties and professional representatives sign documents; contractual commission percentages differ; developer/owner pays agency; agent An claims share.
**Actions:** promote a qualified case to Deal intentionally; invite participants in company or Group context; negotiate/record terms; contract with actual legal parties; record payment obligations, evidence, exceptions, dispute, close/reopen.
**Truth:** Deal context Organization OR Group exactly one, not both; participating Person ≠ legal counterparty; contribution attribution ≠ commission contract; commission term ≠ earned ≠ payable ≠ paid ≠ settled. Some legal payees are Organizations. Seller/buyer obligation cannot be inferred from CRM Lead owner. Deal invitation separate participation. A Deal may involve multiple Properties/contracts but legal commercial meaning controls relationships.
**Failure:** broker changes ownership by changing owner_id, settled commission inferred from closing Deal, duplicate disbursement after retry, paying unqualified party, conflicting revenue shares, returning employee losing evidentiary history.
**Gate:** implement only after real deal process/contracts and legal/payment accountability; review legal capacity, audit immutability and reversals. No wallet by default.

## 12. Billing, plans, seats, entitlements, invoices, payments
**Story:** ABC buys Team package for six employees, upgrades, removes An, adds new hire; private An can have separate personal plan; charge recurs; payments fail and retry.
**Actions:** quote plan/price version, create subscription to paying Party, allocate seats by explicit rule, grant feature entitlements, invoice, provider attempt, settle/reconcile, suspend/restore, refund/dispute with traceability.
**Truth:** subscriber Party ≠ logged Account ≠ resource owner ≠ legally eligible publisher. Paid entitlement != authorization. Plan price is time/version/period/currency qualified; invoice != payment; attempted != succeeded != settlement. Usage meter idempotent; quota not synonymous with resource access.
**Failure:** upgrades overwrite plan history, employee package ownership transferred to company, failed webhook provisions service, duplicate charge, payment data in plaintext.
**Gate:** first pilot can invoice manually per contracted team if lawful; build billing automation only when recurring transactions justify it. Safe security, export and org isolation are not upsells.

## 13. Assets, leases, investment and portfolios
**Story:** owner tracks a house as rental investment; tenant leases it; repair expense and rent flow; co-investors contribute money; property is not necessarily company Asset.
**Actions:** portfolio admission, valuations, beneficial/economic share evidenced, lease, costs/income, obligations, cash reconciliation, withdrawal/disposal.
**Truth:** Asset = economic treatment of Property, property title ≠ automatically `asset_ownerships`, valuation ≠ asking price, rental invoice ≠ received rent, investment commitment ≠ actual funds.
**Failure:** duplicate asset per same property without portfolio policy, false ownership percentages, expenses capitalized incorrectly, investor return unsupported. Defer until paying owner/property-manager workflow.

## 14. Auction
**Story:** legitimate auction organizer admits bidders and collects valid bids; bid is time/bound and result auditable.
**Actions:** validate authority, registration eligibility, deposit rules where relevant, place idempotent time-ordered bid, close, dispute and publish qualified result.
**Truth:** registration ≠ bid, bid ≠ winner, winner ≠ ownership transfer, auction notice ≠ official legal result.
**Failure:** concurrency admits two winners, bid after close, falsified official data, deposit/refund accountability. Regulated domain; defer without partner/legal readiness and throughput proof.

## 15. Community, social network, chat and notification
**Story:** local housing community discusses neighborhoods, follows broker, reads news; group collaboration leads to direct messages; moderation responds to fraud report.
**Actions:** join/leave, post/comment/reaction/follow with abuse controls, report/moderate, message with consent/visibility, notify according to permissions.
**Truth:** Community ≠ Company ≠ external Facebook Group; Community Post ≠ Listing; platform message ≠ Zalo message. Moderation decision, content history and legal removal rights may require own evidence.
**Failure:** spam/misrepresentation/defamation, unauthorized private contact disclosure, children/privacy issues, mass-notification abuse. No native audience at launch; defer feature surface except minimal operational notifications.

## 16. Reputation, badges, trust and sponsored ranking
**Story:** consumer sees broker with completed deals vs newly paid premium badge; disputed deal evidence later overturned; new provider requests verification.
**Actions:** evidence adjudication, policy versioning, bounded assessment, dispute/correction, revocable benefit, label paid promotion separately.
**Truth:** verified legal identity ≠ qualified broker ≠ authorized publisher ≠ honest outcome; paid plan cannot create trust status; reputation assessment needs evidence scope, time, dispute and policy version. One metric cannot claim universal quality.
**Failure:** fake transactions, purchased trust, opaque biased score, privacy disclosure of private deals, rewards after evidence revoked.
**Gate:** reputation policy/assessment/grant tables are research candidates only; defer until third-party verifiable signals, appeal operations and an audience that uses signals.

## 17. Planning, GIS, cadastre and government evidence
**Story:** buyer asks if a property overlaps an announced road planning polygon; source dataset later superseded; parcel data differs from location pin; government portal usage is licensed.
**Actions:** acquire lawful data, store source/jurisdiction/license/CRS/version/effective dates, import geometries, overlay with precision/error, generate report with source manifest, correction, retire outdated.
**Truth:** spatial intersection ≠ title, zoning permission or final legal ruling; dataset `published_at` ≠ downloaded_at ≠ effective_from; parcel polygon ≠ Property; GIS result is derived context.
**Failure:** wrong EPSG/CRS, boundary errors, mixed administrative eras, outdated government plan presented as official current, prohibited redistribution, enormous ETL/search cost.
**Gate:** verify per-province data rights/source, cost and willingness to pay before PostGIS pipeline implementation. Historical `tqd-service` an adjacent research/operating context, not proof of licenses.

## 18. Content, marketing operations and platform governance
**Story:** marketing team writes blog, articles, company pages, SEO landing pages, sponsored ads; tenant requests takedown; fake broker uploads stolen media; support handles complaint.
**Actions:** editorial workflow with author/reviewer and version; scheduled publishing; slug and redirect; sponsored placement disclosure; media moderation and provenance; incident reporting, appeal and restore; abuse throttling.
**Truth:** CMS Article is not Community Post and not a Listing; marketing campaign ≠ user-consented messages; content availability ≠ legal accuracy. Platform operator has separate privileged access and audit obligations, no silent tenant-CRM read.
**Failure:** indexing private content, paid ad promoted as verified organic, takedown fails in caches, moderator edits evidence invisibly. **CMS/Marketing is a known missing capability from 160-table V0.3**, not a validated product priority. Govern from day one if public features enabled; full CMS after demand.

## 19. Security, privacy, compliance and operating economics across all domains
**Boundaries:** authentication; current scoped authorization at object/field/action; purpose limitation and customer privacy; resource ownership distinct from access; fail-closed exports; ciphertext/secrets and safe media; audit with minimal PII; back-ups **restored in drills**; incident response; retention/erasure exceptions and legal holds; transaction/idempotency and concurrency; tenant isolation across read, search, media, jobs and caches; report only observed facts. Threat-model AI output as untrusted claim. Review applicable law by action, not one `legal_verified` boolean.
**Sources (verify and interpret with legal counsel before production):**
- Vietnam Law on Real Estate Business 29/2023/QH15 as amended/effective from 2024-08-01: https://vbpl.vn/TW/Pages/ivbpq-thuoctinh.aspx?ItemID=169027
- Vietnam Personal Data Protection Law 91/2025/QH15 and Government Decree 356/2025/NĐ-CP effective 2026-01-01: https://vbpl.vn/TW/Pages/vbpq-toanvan.aspx?ItemID=187276
- Business Reality Gate for channel/market caveats: `docs/database/final/BUSINESS-REALITY-GATE-2026-10-09.md`.
**Operational economics:** first user benefit – entry time – rework – onboarding – subscription – platform hosting/support/privacy risk. Costs of complex policies, data reconciliation, sanctions and moderation matter more than raw table count.

## 20. Priorities by business proof, not schema completeness
**P0 — learn:** BUSINESS-00 observe Facebook/Zalo/Sheets/manual work, adoption and WTP; compare Stock-only, Follow-up-only, Stock+Share. Do not claim pilot already done.
**P1 — candidate first paying slice:** Account/Organization minimal context, source intake + price report, manual share preview with permission gate, Inquiry, follow-up, simple team read view; no native marketplace lead assumption. Software implementation only after entry gate; disciplined lab proofs can begin separately.
**P2 — conditional by demand:** dedupe/provenance, property identity linking, actual appointments, structured demands, CRM opportunity, multi-team access, external distribution evidence.
**P3 — contingent business expansion:** publish Listing with verified scoped mandates, enterprise provisioning, paid plans, CMS/SEO/ads, project distribution, cross-company group.
**P4 — regulated/network-heavy:** Deal money flow, auction, investment, wallets, PostGIS government sources, large Community and reputation.
**Not implied:** P1 success guarantees P4 demand, or all 160 candidates should be migrated.

## 21. Never confuse coverage with closure
Atlas covers every existing family (Identity, Org, Group, Property/Project, Listing/Asset, Deal/Auction, CRM, Billing, Community/Chat, Appointments, Planning/GIS, Operational events, Reputation) plus missing CMS/Marketing and real source/inquiry semantics. **Open decisions remain:** C01 enforceable exactly-one; Account cardinality; Member episode; new Organization/claim legitimacy; eligibility and authority; personal-vs-org source custody; tenant scope of Group sharing; source report vs Listing; publication rights; external channel attribution; CRM Contact/Inquiry/Follow-up/Demand granularity; consent processing basis/retention; Deal payees, commission and payment reconciliation; legal+licensing of GIS; editorial CMS and sponsored ranking; sustainable P&L and actual user adoption. Do not mark these as proved merely because they appear in a document.

## 22. Restoration / next concrete work
Read `docs/continuity/checkpoints/2026-10-10-business-to-core/README.md`, then `BUSINESS-TO-CORE-IMPLEMENTATION.md`, `VS01-CONTRACTS-AND-DB-GAPS.md`, and `DATABASE-CHANGE-DECISIONS.md` in that folder. After a new chat says "Tiếp tục", use repo/live code and that checkpoint, identify last verified gate, then continue **one next pressure**; do not restart generic brainstorming or claim unseen code runs.
