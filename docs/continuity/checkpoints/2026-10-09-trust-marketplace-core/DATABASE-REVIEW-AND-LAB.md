# Canonical database review + PostgreSQL proof ledger — 2026-10-09

## Source authority and scope

Target: `docs/database/final/BDSPro-CANONICAL-DATABASE.dbml` on design branch `architecture/canonical-database-final`. Inspected 2026-10-09: **146 Table declarations**. This is a table-inventory and boundary-risk review, **not** a row-by-row acceptance of every column, constraint, business policy or legal assertion. Continue full per-table review from table 1. Old design acceptance in `CANONICAL-DATABASE-ACCEPTANCE.md` has been reopened wherever new actor scenarios contradict its assumptions.

Statuses: KEEP candidate means retain semantic boundary pending physical review; MODIFY needed for meaningful gap; REOPEN semantic/cardinality assumption; DEFER until real trigger; NOT PROVEN unknown.

## Complete table inventory, ordered by current DBML

### Identity + authentication, 1–13
1 parties; 2 persons; 3 person_profiles; 4 organizations; 5 accounts; 6 account_emails; 7 account_phones; 8 password_credentials; 9 external_identities; 10 passkeys; 11 sessions; 12 service_accounts; 13 service_account_keys.

- KEEP candidate: Party/Person/Organization identity, Person profile not authentication truth.
- MODIFY/REOPEN: subtype exactly-one committed (C01) not enforced just by two FKs; accounts.person_id lacks UNIQUE yet multi-account need unproven; determine external IdP issuer+subject linking, contact uniqueness, recovery.
- Enterprise work identity NOT automatic; separate from enterprise-authenticated personal account.
- Service account = non-human system principal, not enterprise-provisioned employee; passkeys/apple sign-in require evidence of need.

### Organization & authorization, 14–24
14 permissions; 15 organization_roles; 16 organization_role_permissions; 17 organization_invitations; 18 organization_memberships; 19 organization_membership_blocks; 20 organization_role_assignments; 21 organization_ownerships; 22 organization_branches; 23 organization_branch_assignments; 24 organization_branch_locations.

- REOPEN: previous `organization_memberships` = repeated participation episodes not justified by business actors alone; compare stable relation + audit vs episodes.
- VERIFY integrity: assignments must reference role and membership belonging to SAME Organization. Branch assignments/managers must belong to owning Organization. `organization_ownerships.organization_id` and referred membership's organization must match; technical owner != legal corporate owner/representative.
- MODIFY: organization legal verification, claim new/existing business, disputes, authority evidence, revocation, compliance/qualifications; `tax_code` scalar alone is not proof. `organization_invitations.intended_role_id NOT NULL` presupposes a role even for simple network invitation. `organization_membership_blocks` life cycle distinct from role removal; justify each.
- Access assurance (e.g. mandatory enterprise SSO for resources) distinct from membership and role; no forced second Account.

### Group and shared foundations, 25–34
25 groups; 26 group_invitations; 27 group_participations; 28 group_leaderships; 29 group_admin_assignments; 30 administrative_areas; 31 amenities; 32 document_types; 33 media_objects; 34 documents.

- OPEN: group collaboration vs community media/social vs deal, scope of leadership/admin, participant Person vs Organization, paid beneficiary.
- MODIFY: documents/files need provenance, sensitive classification, permitted viewers, retention and proof/evidence boundary; a document object doesn't grant access.
- Administrative areas need boundary/history/version semantics, especially changing geography.

### Property/project, 35–48
35 property_types; 36 properties; 37 property_locations; 38 property_land_parcels; 39 property_house_details; 40 property_apartment_details; 41 property_amenities; 42 property_media; 43 property_documents; 44 property_valuations; 45 real_estate_projects; 46 project_buildings; 47 project_properties; 48 project_documents.

- KEEP separation property facts from market exposure. REOPEN naming for `property_types`, `property_house_details`, `property_apartment_details` after defining taxonomy.
- MODIFY: point vs polygon, self-reported property data vs legally verified parcel, multi-parcel/multi-Property, cadastral changes, document/reliability/effective dates, source claims; project legal/operational context.

### Listing & asset, 49–62
49 listings; 50 listing_properties; 51 listing_price_terms; 52 listing_publications; 53 listing_media; 54 listing_offers; 55 listing_offer_acceptances; 56 assets; 57 asset_ownerships; 58 asset_valuations; 59 asset_costs; 60 asset_income_entries; 61 asset_leases; 62 asset_documents.

- KEEP candidate: Listing != Property != Asset. Listing.publisher_party_id Party may be Person or Organization, but does NOT prove owner, broker qualification or right to publish.
- MODIFY: provider activation, publication authorization, evidence of owner/agency rights, draft vs publication, temporal listing changes, multiple properties and asset meaning. Physical ownership of assets may be distinct from property title and authorized marketing.

### Deal and auctions, 63–84
63 deals; 64 deal_properties; 65 deal_invitations; 66 deal_participations; 67 deal_customers; 68 deal_partners; 69 deal_admin_assignments; 70 deal_leaderships; 71 deal_commission_terms; 72 deal_commission_earnings; 73 deal_investment_commitments; 74 deal_investments; 75 deal_contracts; 76 deal_contract_customers; 77 deal_contract_properties; 78 deal_contract_payments; 79 deal_documents; 80 auctions; 81 auction_properties; 82 auction_registrations; 83 auction_bids; 84 auction_results.

- Deal financial responsibility may attach to Person AND Organization; do not infer legal payer/payee from person-only participation. Need ensure earnings participation belongs to the same deal.
- Terms != earnings != approved payable != payment != settlement; legal constraints and proof.
- Auction notice ingestion/intelligence != conducting auctions/bidding. Do not implement execution just because tables exist; legal investigation first.
- Avoid generic owner_type+owner_id without FK integrity for durable legal relationships.

### CRM, 85–94
85 crm_contacts; 86 crm_labels; 87 crm_contact_labels; 88 crm_pipelines; 89 crm_stages; 90 crm_opportunities; 91 crm_opportunity_contacts; 92 crm_opportunity_properties; 93 crm_activities; 94 crm_opportunity_deals.

- CRM Contact is tenant-scoped private relationship; optional `linked_party_id` ≠ permission to share other tenants' CRM. Investigate inbound lead, consent, ownership, qualification, attribution, assignment, retention/deletion.

### Billing/usage, 95–110
95 plans; 96 plan_prices; 97 plan_entitlements; 98 subscriptions; 99 invoices; 100 invoice_lines; 101 payments; 102 invoice_payments; 103 payment_attempts; 104 payment_settlements; 105 bank_accounts; 106 wallets; 107 wallet_entries; 108 usage_meters; 109 usage_events; 110 quota_allocations.

- Commercial entitlement separate from roles/verification. Subscriber Party vs beneficiary Group/Community/workspace/Person: require real product policies and payee. Seats != invitation != active member unless commercial terms say so.
- Usage metering and payment idempotency correctness are durable; wallet/custodial money services DEFER absent legal and operational need.

### Community/network/content, 111–131
111 communities; 112 community_memberships; 113 community_posts; 114 community_post_media; 115 community_comments; 116 community_post_reactions; 117 community_comment_reactions; 118 friendships; 119 follows; 120 conversations; 121 conversation_participations; 122 messages; 123 message_attachments; 124 message_reads; 125 appointments; 126 appointment_participants; 127 appointment_deals; 128 appointment_properties; 129 notifications; 130 notification_deliveries; 131 notification_preferences.

- Moderation, blocking, spam prevention, consent, audience, deleted/redacted content, reporting and data protection needed before scaling network/reputation.
- Follow/friend/invitation interactions do not automatically create verified working relationships or deserved reputation.

### Planning/GIS, 132–142
132 planning_sources; 133 planning_datasets; 134 planning_layers; 135 planning_features; 136 planning_labels; 137 planning_feature_labels; 138 planning_documents; 139 planning_reports; 140 planning_report_properties; 141 planning_news; 142 planning_news_saves.

- KEEP PostGIS-capable spatial domain; MODIFY source authority, decision doc, license/access, version, effective vs ingested time, CRS precision, source completeness; report must identify underlying dataset versions.
- Property coordinate point cannot be used as proof of exact parcel boundary. Official planning overlay != legal determination.
- Historical tqd-service included planning, spatial, discovery, related-entity projections, reports, usage/quota, PMTiles mode; do NOT inherit deployment topology.

### Operational/audit/integration, 143–146
143 audit_events; 144 outbox_events; 145 inbox_events; 146 idempotency_records.

- Different truth from domain records; define transactional write/consumer delivery, dedupe, privacy-safe payloads, retention, recovery and policy-based audit access.
- `subject_kind+subject_id` is acceptable for operational evidence only when deliberately lossy; not durable foreign-key business ownership.

## Discovered capability gaps (NOT automatic new tables)

- Organization registration/claims, legal verification, representative authority and dispute/transfer.
- Provider/seller qualification, property offering/publication authority and verified badge attestations.
- Reputation policy version, evidence metrics, review/demotion, benefits, sponsorship separation and ranking projections.
- Enterprise conditional SSO/security assurance, optional managed identity/provisioning after a real client trigger.
- Marketing/SEO/CMS/campaign/attribution with privacy and evidence obligations.
- National VNeID/eKYC/government sources with lawful access/licensing and least-data storage.
- Reliable source provenance/accuracy/versioning for map/parcel/planning/auction intelligence.
- Financial payable/commission conditions, reconciliations, dispute/correction flows.
- Platform admin vs tenant/organization administration; anti-fraud, moderation and safe complaint handling.

## Laboratory evidence ledger

Environment: PostgreSQL 18.6 (Docker), `bdspro_test`, schema `c01_lab`. Details are based on recorded user SQL outputs; future chat should confirm live DB state before mutation.

- Earlier DB-01..DB-14: Party/Person/Organization FK/PK, Account->Person, basic organization Membership and time windows. Plain FK/PK do **not** enforce exactly-one subtype C01. Multiple Account rows 2 and 3 can reference Person 1 because no UNIQUE; this establishes database possibility, not business justification.
- DB-15: intentionally duplicate open Membership (6,1) inserted; DB-16: attempt to build partial UNIQUE index failed due to duplicate; DB-17: clean duplicate and successfully create `organization_memberships_one_open_uq` on `(organization_id, person_id) WHERE ended_at IS NULL`.
- DB-18: a second open Membership insert was rejected by partial UNIQUE.
- DB-19: end membership ID 2 and create ID 5 in one transaction; used same `NOW()` boundary. ID 1 historical ended.
- DB-20 result:
  ```text
  old_membership_id=2; new_membership_id=5
  same_boundary=true; is_overlapping=false
  ```
  Half-open ranges `[joined_at,ended_at)` touch but do not overlap; does not establish that episodic Membership is the right BDSPro business model.
- DB-21/C02 is **proposed, not confirmed as run**. Previous suggested lab `organization_claim_requests(organization_id NOT NULL)` only represents claims to an EXISTING Organization, not initial request for a new legal entity. Do not assert table/rows already exist.

### SQL learning loop
Explain scenario, invariant, prediction -> user writes focused SQL -> executes under psql -> observes exact output/error -> explain which mechanism was proven and which business/security claims were not. Separate DB constraint integrity and actual application/DB-role authorization. Avoid giving full copy/paste implementations without user attempt; use IDE/database client according to work surface.

## Immediate next work

1. Start **ORG-01** at `organizations`, `parties`, `persons`, `accounts`, `organization_ownerships`, `organization_memberships`.
2. Tell a precise story with (A) new ABC registration, (B) existing ABC competing claims, (C) representative transfers/offboarding, (D) organization imposes stronger SSO for CRM. Distinguish legal entity identity, application, evidence, workspace authority and compliance.
3. Identify one-row assertions and comparison of alternatives; do **not** create broad claim/verification table suite yet.
4. Decide first narrow PostgreSQL exercise to prove **request != authority**. If using claim FK to Organization, scope experiment to *existing Organization only*. Test duplicate requests / transaction separately after policy is defined.
5. Return to C01 physical exactly-one subtype enforcement and Account cardinality when needed for ORG-01; defer big restructuring and production migration.
6. Record each KEEP/MODIFY/DEFER/REJECT with actors, proof, cost/benefit and trigger. Extend documentation once new evidence arrives.

## Preserving the source branch

The checkpoint documents intentionally create **no DBML or Go/migration changes**. New chat should check branch/HEAD (and local uncommitted work) rather than assuming this remote branch equals local `dev`. Never rewrite migrations or copy the 146-table target into production at once.

## One sentence for future assistant

The goal is **not** to optimize for the number of tables; it is to turn trustworthy business reality into the minimal durable data contracts that can serve a scalable Vietnamese property/community/data ecosystem while protecting privacy, money, authority and verifiable information.

## ADDENDUM — organization networks, teams, channels, provider capacities

Full semantic scenario and business comparison: [ORGANIZATION-NETWORK-AND-PROVIDER-ONTOLOGY.md](ORGANIZATION-NETWORK-AND-PROVIDER-ONTOLOGY.md).

Reopened candidate gaps from the existing schema: `organization_branches` and `organization_branch_assignments` cannot automatically stand in for functional Team/Department; `groups` has no explicit enterprise affiliation and `group_participations` targets Person only; `community_memberships` similarly targets Person; `listings.publisher_party_id` does not establish actual account actor or asset-specific authority; `crm_contacts.organization_id` is tenant scope and not shared Group data. No corresponding official `teams` / `channels` tables exist in candidate DBML. These observations justify **review**, not automatic table addition.

Add to ORG-01 case set: employer ABC starts An's BDSPro onboarding where An had **no earlier Account**; employment end revokes enterprise resource access. Add ORG-NET-01: An belongs to ABC's internal sales Team, ABC and XYZ share a project Group, An publishes a Listing for ABC with valid asset authorization; after transfer to XYZ personal reputation is preserved while private ABC CRM is not exposed.

Current lab remains DB-20 observed, DB-21 proposed/unexecuted; no SQL changed by this checkpoint.
