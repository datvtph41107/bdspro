# BDSPro — Canonical Core V0.2: 146-table impact & directional-vector matrix

> **V0.3 supplement:** [Core review](CORE-REVIEW-V0.3.md), [32 directional scenarios and 1600 intersection framework](CORE-VECTOR-REVIEW-V0.3.md). The original **146**-row source-to-target mapping below is preserved; six composite FK candidates have since been added to the unchanged 160-table V0.3 count. This mapping labels review pressures, not physical acceptance.


**Date:** 2026-10-09. **Branch:** `architecture/canonical-database-final`. **Status:** HUMAN REVIEW INVENTORY, NOT COMPLETE PHYSICAL ACCEPTANCE.  
Canonical V0.2: **160 named Tables**, containing **146 original roles** (4 renamed, 142 named unchanged) and **14 additional candidate responsibilities**. All 160 are in TableGroups; explicit FK references resolve to defined tables. That is structural validation only, not complete enforcement, workload tuning, law or security validation.

## What a "vector" is

A vector means the directional causal dependency from a real actor/problem to a state transition, source of truth, downstream read/economic effect and risk. We do not use abstract system graph geometry as a substitute for proof.

### Nine independent review lenses

- **I** = Identity, keys, legal/business ownership of truth.
- **A** = Actor, authorization, delegated authority, credentials and resource permission.
- **S** = Scope, tenant boundaries, organization/team/group/branch context.
- **T** = Time, creation/expiry/revocation, legal effective vs system processing time.
- **E** = Evidence, source provenance, verification, case resolution, appeals.
- **P** = Personal/privacy/regulated data, purpose limitation and confidentiality.
- **D** = Spatial/geographic accuracy, CRS, legal mapping and versions.
- **M** = Money, payer/payee, entitlement, commission and reconciliation.
- **R** = Read/UX/discovery/projection, search and ranking.
- **O** = Operations/recovery, locks, async events, logs, migration and unit economics.

The vector codes in the table below mark the **leading lenses**, not a guarantee that other lenses do not apply. Across 146 tables × 10 possible lenses, this produces **1,460 potential review intersections**, not 1,460 proven invariants. Actual engineering decisions require the concrete story, relevant facts and a discriminating experiment.

## Original 146-table traceability matrix

| Original no. | Original table | V0.2 table | Domain | Review state | Lead vectors | Key unresolved question / observed pressure |
|---:|---|---|---|---|---|---|
| 001 | `parties` | `parties` | Identity/auth | **REOPEN** | I·A·P·O | Exactly-one Person/Organization subtype still unproven at commit |
| 002 | `persons` | `persons` | Identity/auth | **REOPEN** | I·A·P·O | Distinct human Person despite multiple legal/work roles |
| 003 | `person_profiles` | `person_profiles` | Identity/auth | **REVIEW** | I·A·P·O | What is credential ownership and lifecycle? |
| 004 | `organizations` | `organizations` | Identity/auth | **MODIFY** | I·A·P·O | Legal identity separate from claim/representation and tenant workspace |
| 005 | `accounts` | `accounts` | Identity/auth | **MODIFY** | I·A·P·O | Multi-account Person still open; corporate-first join does not force extra managed Account |
| 006 | `account_emails` | `account_emails` | Identity/auth | **REOPEN** | I·A·P·O | Email is recovery/contact channel; uniqueness and ownership policy unproven |
| 007 | `account_phones` | `account_phones` | Identity/auth | **REVIEW** | I·A·P·O | What is credential ownership and lifecycle? |
| 008 | `password_credentials` | `password_credentials` | Identity/auth | **REVIEW** | I·A·P·O | What is credential ownership and lifecycle? |
| 009 | `external_identities` | `external_identities` | Identity/auth | **REOPEN** | I·A·P·O | Issuer+subject identity link; avoid auto-link by matching email |
| 010 | `passkeys` | `passkeys` | Identity/auth | **REVIEW** | I·A·P·O | What is credential ownership and lifecycle? |
| 011 | `sessions` | `sessions` | Identity/auth | **REOPEN** | I·A·P·O | Corporate SSO requirement and revocation latency must be bounded |
| 012 | `service_accounts` | `service_accounts` | Identity/auth | **REOPEN** | I·A·P·O | Non-human principal; avoid confusing employee accounts |
| 013 | `service_account_keys` | `service_account_keys` | Identity/auth | **REVIEW** | I·A·P·O | What is credential ownership and lifecycle? |
| 014 | `permissions` | `permissions` | Organization | **REVIEW** | A·S·T·E·O | Which org boundary and effective authority? |
| 015 | `organization_roles` | `organization_roles` | Organization | **REOPEN** | A·S·T·E·O | Permission bundle in organization, not legal authority |
| 016 | `organization_role_permissions` | `organization_role_permissions` | Organization | **REVIEW** | A·S·T·E·O | Which org boundary and effective authority? |
| 017 | `organization_invitations` | `organization_invitations` | Organization | **MODIFY** | A·S·T·E·O | Can target accountless employee; role optional; authorization of inviter |
| 018 | `organization_memberships` | `organization_memberships` | Organization | **REOPEN** | A·S·T·E·O | Episodes vs durable relationship remains contested; offboarding |
| 019 | `organization_membership_blocks` | `organization_membership_blocks` | Organization | **REVIEW** | A·S·T·E·O | Which org boundary and effective authority? |
| 020 | `organization_role_assignments` | `organization_role_assignments` | Organization | **REOPEN** | A·S·T·E·O | Prove role and membership belong to SAME organization |
| 021 | `organization_ownerships` | `organization_primary_administrators` | Organization | **RENAME** | A·S·T·E·O | RENAME workspace admin; no legal ownership inference; SAME-org FK |
| 022 | `organization_branches` | `organization_branches` | Organization | **REOPEN** | A·S·T·E·O | Branch is geographic/operational unit, not arbitrary team/department |
| 023 | `organization_branch_assignments` | `organization_branch_assignments` | Organization | **REOPEN** | A·S·T·E·O | Membership assigned must match branch.organization_id |
| 024 | `organization_branch_locations` | `organization_branch_locations` | Organization | **REVIEW** | A·S·T·E·O | Which org boundary and effective authority? |
| 025 | `groups` | `groups` | Group | **REOPEN** | A·S·T·P | Independent/cross-firm collaboration, not necessarily organization child |
| 026 | `group_invitations` | `group_invitations` | Group | **REVIEW** | A·S·T·P | Who represents whom and who may access shared data? |
| 027 | `group_participations` | `group_participations` | Group | **REOPEN** | A·S·T·P | Person joining Group != representing own Organization |
| 028 | `group_leaderships` | `group_leaderships` | Group | **REVIEW** | A·S·T·P | Who represents whom and who may access shared data? |
| 029 | `group_admin_assignments` | `group_admin_assignments` | Group | **REVIEW** | A·S·T·P | Who represents whom and who may access shared data? |
| 030 | `administrative_areas` | `administrative_areas` | Shared/reference | **REVIEW** | I·E·P·O | Boundary/codes change over time; geography needs versions |
| 031 | `amenities` | `amenities` | Shared/reference | **REVIEW** | I·E·P·O | Who authors, proves and may view reference/evidence? |
| 032 | `document_types` | `document_types` | Shared/reference | **REVIEW** | I·E·P·O | Who authors, proves and may view reference/evidence? |
| 033 | `media_objects` | `media_objects` | Shared/reference | **REVIEW** | I·E·P·O | Sensitive evidence storage scope, encrypted references and no public default |
| 034 | `documents` | `documents` | Shared/reference | **REVIEW** | I·E·P·O | Evidence access/retention/redaction distinct from document link |
| 035 | `property_types` | `property_categories` | Property/project | **RENAME** | I·T·E·D·R | RENAME classification; confirm taxonomy independent from legal land-use |
| 036 | `properties` | `properties` | Property/project | **REOPEN** | I·T·E·D·R | Property differs from legal parcel, Listing and Asset |
| 037 | `property_locations` | `property_locations` | Property/project | **MODIFY** | I·T·E·D·R | Point accuracy source and CRS; not cadastral boundary proof |
| 038 | `property_land_parcels` | `property_land_parcels` | Property/project | **REOPEN** | I·T·E·D·R | Self-declared parcel descriptors != authoritative cadastral entity |
| 039 | `property_house_details` | `property_houses` | Property/project | **RENAME** | I·T·E·D·R | RENAME to housing-specific subtype, validate subtype constraints |
| 040 | `property_apartment_details` | `property_apartments` | Property/project | **RENAME** | I·T·E·D·R | RENAME to apartment-specific subtype, validate project/building relations |
| 041 | `property_amenities` | `property_amenities` | Property/project | **REVIEW** | I·T·E·D·R | What source and legal identity supports this fact? |
| 042 | `property_media` | `property_media` | Property/project | **REVIEW** | I·T·E·D·R | What source and legal identity supports this fact? |
| 043 | `property_documents` | `property_documents` | Property/project | **REVIEW** | I·T·E·D·R | What source and legal identity supports this fact? |
| 044 | `property_valuations` | `property_valuations` | Property/project | **REVIEW** | I·T·E·D·R | What source and legal identity supports this fact? |
| 045 | `real_estate_projects` | `real_estate_projects` | Property/project | **REVIEW** | I·T·E·D·R | Developer != legal project approval; verify source and life cycle |
| 046 | `project_buildings` | `project_buildings` | Property/project | **REVIEW** | I·T·E·D·R | What source and legal identity supports this fact? |
| 047 | `project_properties` | `project_properties` | Property/project | **REVIEW** | I·T·E·D·R | What source and legal identity supports this fact? |
| 048 | `project_documents` | `project_documents` | Property/project | **REVIEW** | I·T·E·D·R | What source and legal identity supports this fact? |
| 049 | `listings` | `listings` | Listing | **REOPEN** | A·T·E·R·P | Publisher Party != human actor != owner/authority to offer |
| 050 | `listing_properties` | `listing_properties` | Listing | **REOPEN** | A·T·E·R·P | Authorized publisher must be scoped to each Property |
| 051 | `listing_price_terms` | `listing_price_terms` | Listing | **REVIEW** | A·T·E·R·P | What proves offer scope at publish/accept time? |
| 052 | `listing_publications` | `listing_publications` | Listing | **REOPEN** | A·T·E·R·P | Publication at a point in time needs valid authority/evidence |
| 053 | `listing_media` | `listing_media` | Listing | **REVIEW** | A·T·E·R·P | What proves offer scope at publish/accept time? |
| 054 | `listing_offers` | `listing_offers` | Listing | **REVIEW** | A·T·E·R·P | What proves offer scope at publish/accept time? |
| 055 | `listing_offer_acceptances` | `listing_offer_acceptances` | Listing | **REVIEW** | A·T·E·R·P | What proves offer scope at publish/accept time? |
| 056 | `assets` | `assets` | Asset | **REVIEW** | I·T·E·M | Asset management ownership economics != physical Property |
| 057 | `asset_ownerships` | `asset_ownerships` | Asset | **REOPEN** | I·T·E·M | Portfolio beneficial/economic rights != property legal title |
| 058 | `asset_valuations` | `asset_valuations` | Asset | **REVIEW** | I·T·E·M | Who owns economic truth and when? |
| 059 | `asset_costs` | `asset_costs` | Asset | **REVIEW** | I·T·E·M | Who owns economic truth and when? |
| 060 | `asset_income_entries` | `asset_income_entries` | Asset | **REVIEW** | I·T·E·M | Who owns economic truth and when? |
| 061 | `asset_leases` | `asset_leases` | Asset | **REVIEW** | I·T·E·M | Who owns economic truth and when? |
| 062 | `asset_documents` | `asset_documents` | Asset | **REVIEW** | I·T·E·M | Who owns economic truth and when? |
| 063 | `deals` | `deals` | Deal/contracts | **REOPEN** | A·S·T·E·M·P | Organization/group context not sole legal counterparties |
| 064 | `deal_properties` | `deal_properties` | Deal/contracts | **REVIEW** | A·S·T·E·M·P | Which legal party/contract and financial obligation? |
| 065 | `deal_invitations` | `deal_invitations` | Deal/contracts | **REVIEW** | A·S·T·E·M·P | Which legal party/contract and financial obligation? |
| 066 | `deal_participations` | `deal_participations` | Deal/contracts | **REOPEN** | A·S·T·E·M·P | Person/Organization capacity and separate contract parties |
| 067 | `deal_customers` | `deal_customers` | Deal/contracts | **REVIEW** | A·S·T·E·M·P | Which legal party/contract and financial obligation? |
| 068 | `deal_partners` | `deal_partners` | Deal/contracts | **REVIEW** | A·S·T·E·M·P | Which legal party/contract and financial obligation? |
| 069 | `deal_admin_assignments` | `deal_admin_assignments` | Deal/contracts | **REVIEW** | A·S·T·E·M·P | Which legal party/contract and financial obligation? |
| 070 | `deal_leaderships` | `deal_leaderships` | Deal/contracts | **REVIEW** | A·S·T·E·M·P | Which legal party/contract and financial obligation? |
| 071 | `deal_commission_terms` | `deal_commission_terms` | Deal/contracts | **REOPEN** | A·S·T·E·M·P | Term is agreement evidence, not automatic payable |
| 072 | `deal_commission_earnings` | `deal_commission_earnings` | Deal/contracts | **REOPEN** | A·S·T·E·M·P | Deal/participation consistency and payout legal eligibility |
| 073 | `deal_investment_commitments` | `deal_investment_commitments` | Deal/contracts | **REVIEW** | A·S·T·E·M·P | Which legal party/contract and financial obligation? |
| 074 | `deal_investments` | `deal_investments` | Deal/contracts | **REVIEW** | A·S·T·E·M·P | Which legal party/contract and financial obligation? |
| 075 | `deal_contracts` | `deal_contracts` | Deal/contracts | **REVIEW** | A·S·T·E·M·P | Parties of contract, not just aggregate Deal ID |
| 076 | `deal_contract_customers` | `deal_contract_customers` | Deal/contracts | **REVIEW** | A·S·T·E·M·P | Which legal party/contract and financial obligation? |
| 077 | `deal_contract_properties` | `deal_contract_properties` | Deal/contracts | **REVIEW** | A·S·T·E·M·P | Which legal party/contract and financial obligation? |
| 078 | `deal_contract_payments` | `deal_contract_payments` | Deal/contracts | **REVIEW** | A·S·T·E·M·P | Payment ≠ provider settlement; evidence and refunds |
| 079 | `deal_documents` | `deal_documents` | Deal/contracts | **REVIEW** | A·S·T·E·M·P | Which legal party/contract and financial obligation? |
| 080 | `auctions` | `auctions` | Auction | **REVIEW** | A·T·E·M·O | Auction notice aggregation != right to run an auction |
| 081 | `auction_properties` | `auction_properties` | Auction | **REVIEW** | A·T·E·M·O | Intelligence display or regulated execution? |
| 082 | `auction_registrations` | `auction_registrations` | Auction | **REVIEW** | A·T·E·M·O | Intelligence display or regulated execution? |
| 083 | `auction_bids` | `auction_bids` | Auction | **REVIEW** | A·T·E·M·O | Bidding business regulated; defer execution until authorization |
| 084 | `auction_results` | `auction_results` | Auction | **REVIEW** | A·T·E·M·O | Intelligence display or regulated execution? |
| 085 | `crm_contacts` | `crm_contacts` | CRM | **REOPEN** | A·S·P·R | Tenant-private; linked Party does not create CRM sharing |
| 086 | `crm_labels` | `crm_labels` | CRM | **REVIEW** | A·S·P·R | Which tenant/purpose is permitted to read/update? |
| 087 | `crm_contact_labels` | `crm_contact_labels` | CRM | **REVIEW** | A·S·P·R | Which tenant/purpose is permitted to read/update? |
| 088 | `crm_pipelines` | `crm_pipelines` | CRM | **REVIEW** | A·S·P·R | Which tenant/purpose is permitted to read/update? |
| 089 | `crm_stages` | `crm_stages` | CRM | **REVIEW** | A·S·P·R | Which tenant/purpose is permitted to read/update? |
| 090 | `crm_opportunities` | `crm_opportunities` | CRM | **REVIEW** | A·S·P·R | Opportunity/lead scope, stage and owner-team assignment |
| 091 | `crm_opportunity_contacts` | `crm_opportunity_contacts` | CRM | **REVIEW** | A·S·P·R | Which tenant/purpose is permitted to read/update? |
| 092 | `crm_opportunity_properties` | `crm_opportunity_properties` | CRM | **REVIEW** | A·S·P·R | Which tenant/purpose is permitted to read/update? |
| 093 | `crm_activities` | `crm_activities` | CRM | **REOPEN** | A·S·P·R | Contact activity privacy, consent and evidence quality |
| 094 | `crm_opportunity_deals` | `crm_opportunity_deals` | CRM | **REVIEW** | A·S·P·R | Which tenant/purpose is permitted to read/update? |
| 095 | `plans` | `plans` | Billing | **REVIEW** | I·T·M·O | Product capacity not role, proof or privilege |
| 096 | `plan_prices` | `plan_prices` | Billing | **REVIEW** | I·T·M·O | Payer, beneficiary, accounting, refund and idempotency? |
| 097 | `plan_entitlements` | `plan_entitlements` | Billing | **REVIEW** | I·T·M·O | Entitlement not organization resource permission |
| 098 | `subscriptions` | `subscriptions` | Billing | **REOPEN** | I·T·M·O | Payer Party vs benefit scope for Team/Group/Community |
| 099 | `invoices` | `invoices` | Billing | **REVIEW** | I·T·M·O | Payer, beneficiary, accounting, refund and idempotency? |
| 100 | `invoice_lines` | `invoice_lines` | Billing | **REVIEW** | I·T·M·O | Payer, beneficiary, accounting, refund and idempotency? |
| 101 | `payments` | `payments` | Billing | **REOPEN** | I·T·M·O | Processor result vs durable payment acceptance and reconciliation |
| 102 | `invoice_payments` | `invoice_payments` | Billing | **REVIEW** | I·T·M·O | Payer, beneficiary, accounting, refund and idempotency? |
| 103 | `payment_attempts` | `payment_attempts` | Billing | **REVIEW** | I·T·M·O | Payer, beneficiary, accounting, refund and idempotency? |
| 104 | `payment_settlements` | `payment_settlements` | Billing | **REOPEN** | I·T·M·O | Reconciliation, reversals, internal funds custody exposure |
| 105 | `bank_accounts` | `bank_accounts` | Billing | **REVIEW** | I·T·M·O | Payer, beneficiary, accounting, refund and idempotency? |
| 106 | `wallets` | `wallets` | Billing | **REOPEN** | I·T·M·O | Real stored-value legal model not yet justified |
| 107 | `wallet_entries` | `wallet_entries` | Billing | **REVIEW** | I·T·M·O | Payer, beneficiary, accounting, refund and idempotency? |
| 108 | `usage_meters` | `usage_meters` | Billing | **REVIEW** | I·T·M·O | Payer, beneficiary, accounting, refund and idempotency? |
| 109 | `usage_events` | `usage_events` | Billing | **REVIEW** | I·T·M·O | Payer, beneficiary, accounting, refund and idempotency? |
| 110 | `quota_allocations` | `quota_allocations` | Billing | **REVIEW** | I·T·M·O | Payer, beneficiary, accounting, refund and idempotency? |
| 111 | `communities` | `communities` | Community | **REVIEW** | A·P·R·O | Content network with moderation not employer-managed team |
| 112 | `community_memberships` | `community_memberships` | Community | **REVIEW** | A·P·R·O | Community participation != operational authority |
| 113 | `community_posts` | `community_posts` | Community | **REVIEW** | A·P·R·O | Who can view/moderate and which reputation signal counts? |
| 114 | `community_post_media` | `community_post_media` | Community | **REVIEW** | A·P·R·O | Who can view/moderate and which reputation signal counts? |
| 115 | `community_comments` | `community_comments` | Community | **REVIEW** | A·P·R·O | Who can view/moderate and which reputation signal counts? |
| 116 | `community_post_reactions` | `community_post_reactions` | Community | **REVIEW** | A·P·R·O | Who can view/moderate and which reputation signal counts? |
| 117 | `community_comment_reactions` | `community_comment_reactions` | Community | **REVIEW** | A·P·R·O | Who can view/moderate and which reputation signal counts? |
| 118 | `friendships` | `friendships` | Community | **REVIEW** | A·P·R·O | Social permission/privacy vs CRM/professional endorsement |
| 119 | `follows` | `follows` | Network/communications | **REVIEW** | A·P·T·O | Follow ≠ reputation nor verified professional endorsement |
| 120 | `conversations` | `conversations` | Network/communications | **REVIEW** | A·P·T·O | Audience privacy, abuse control and retention? |
| 121 | `conversation_participations` | `conversation_participations` | Network/communications | **REVIEW** | A·P·T·O | Audience privacy, abuse control and retention? |
| 122 | `messages` | `messages` | Network/communications | **REVIEW** | A·P·T·O | Private messaging, moderation and lawful retention |
| 123 | `message_attachments` | `message_attachments` | Network/communications | **REVIEW** | A·P·T·O | Audience privacy, abuse control and retention? |
| 124 | `message_reads` | `message_reads` | Network/communications | **REVIEW** | A·P·T·O | Audience privacy, abuse control and retention? |
| 125 | `appointments` | `appointments` | Network/communications | **REVIEW** | A·P·T·O | Audience privacy, abuse control and retention? |
| 126 | `appointment_participants` | `appointment_participants` | Network/communications | **REVIEW** | A·P·T·O | Audience privacy, abuse control and retention? |
| 127 | `appointment_deals` | `appointment_deals` | Network/communications | **REVIEW** | A·P·T·O | Audience privacy, abuse control and retention? |
| 128 | `appointment_properties` | `appointment_properties` | Network/communications | **REVIEW** | A·P·T·O | Audience privacy, abuse control and retention? |
| 129 | `notifications` | `notifications` | Network/communications | **REVIEW** | A·P·T·O | Privacy-safe delivery and per-user choices |
| 130 | `notification_deliveries` | `notification_deliveries` | Network/communications | **REVIEW** | A·P·T·O | Audience privacy, abuse control and retention? |
| 131 | `notification_preferences` | `notification_preferences` | Network/communications | **REVIEW** | A·P·T·O | Audience privacy, abuse control and retention? |
| 132 | `planning_sources` | `planning_sources` | Spatial/Planning | **MODIFY** | I·T·E·D·R | Issuer, license/use right, retrieval time and source status |
| 133 | `planning_datasets` | `planning_datasets` | Spatial/Planning | **MODIFY** | I·T·E·D·R | Snapshot/version/approval/effective and import times |
| 134 | `planning_layers` | `planning_layers` | Spatial/Planning | **REVIEW** | I·T·E·D·R | Official source, CRS, version and permitted usage? |
| 135 | `planning_features` | `planning_features` | Spatial/Planning | **REOPEN** | I·T·E·D·R | Geometry SRID, confidence and source feature linkage |
| 136 | `planning_labels` | `planning_labels` | Spatial/Planning | **REVIEW** | I·T·E·D·R | Official source, CRS, version and permitted usage? |
| 137 | `planning_feature_labels` | `planning_feature_labels` | Spatial/Planning | **REVIEW** | I·T·E·D·R | Official source, CRS, version and permitted usage? |
| 138 | `planning_documents` | `planning_documents` | Spatial/Planning | **REVIEW** | I·T·E·D·R | Official source, CRS, version and permitted usage? |
| 139 | `planning_reports` | `planning_reports` | Spatial/Planning | **MODIFY** | I·T·E·D·R | Report version manifest, disclaimers and reproducibility |
| 140 | `planning_report_properties` | `planning_report_properties` | Spatial/Planning | **REVIEW** | I·T·E·D·R | Official source, CRS, version and permitted usage? |
| 141 | `planning_news` | `planning_news` | Spatial/Planning | **REVIEW** | I·T·E·D·R | Official source, CRS, version and permitted usage? |
| 142 | `planning_news_saves` | `planning_news_saves` | Spatial/Planning | **REVIEW** | I·T·E·D·R | Official source, CRS, version and permitted usage? |
| 143 | `audit_events` | `audit_events` | Integration | **REOPEN** | T·E·P·O | Who/what/when without logging KYC and private CRM secrets |
| 144 | `outbox_events` | `outbox_events` | Integration | **REVIEW** | T·E·P·O | Event delivery/idempotency and confidential payload minimization |
| 145 | `inbox_events` | `inbox_events` | Integration | **REVIEW** | T·E·P·O | Consumer duplicate handling & retry semantics |
| 146 | `idempotency_records` | `idempotency_records` | Integration | **REVIEW** | T·E·P·O | Key scope, request hash, retention and replay contract |

## The 14 new candidate tables — NOT 14 automatically approved migrations

| Table | New assertion | Business value bought | Cost, failure and next proof |
|---|---|---|---|
| `organization_registration_requests` | Account requested registration of a declared legal entity not necessarily yet in canonical Party | Accept enterprise creation without premature legal authority | Matching duplicates, confidential evidence, race, manual review, legal IDs |
| `organization_claim_requests` | Account claims administrative control of an **existing** Organization | Dispute/competing claimant evidence separated from authority | Must not auto-create membership/administrator; approval and revoke process |
| `party_verifications` | An attestation of a specific Party property/capacity by a source | Person KYC, organization existence or qualification need precise meaning | Do not store unnecessary raw biometric/ID; period/issuer/scope/revoke and GDPR-like privacy review |
| `organization_representation_authorizations` | Person has evidenced authority to represent Organization for a bounded purpose | Separates corporate legal representation from workspace admin | Proof may be activity-scoped; authority expiry, conflict and verifier |
| `organization_teams` | Internal operating unit belonging to one Organization | Team-based CRM/lead assignment and functional work | More ACL/cost; avoid copy of Branch; cross-branch cases |
| `organization_team_assignments` | Membership assigned to internal Team during interval | Team-scoped work and offboarding | SAME-org invariant; overlap / assignment history |
| `group_organization_participations` | Organization participates as an explicit party to a collaboration Group | Cross-company supply/project distribution | Group still not Party; no automatic CRM leak; true legal deal parties separately |
| `cadastral_parcels` | Source-qualified parcel geometry/version | Distinguish legal cadastral reference from user listing location | Data access/license, CRS, scale, revision and authority limitations |
| `property_cadastral_links` | Property is associated with a source-qualified parcel by stated basis | Multi-parcel properties and traceable map relationship | Matching confidence, split/merge, validation and temporal re-link |
| `listing_publishing_authorizations` | Publisher Party has documented offer authority on specific Property/Listing | Buyer safety, broker and ownership separation | Strong composite FK/transaction gate, verify grantor's authority, expiration |
| `provider_enrollments` | Party requested activation for a particular provider capacity | Consumer-to-owner-seller/broker/company service onboarding | Different laws, evidence, suspensions and no universal seller=buyer distinction |
| `reputation_policy_versions` | Evaluator uses a versioned calculation policy | Repeatable decisions / fairness / controlled rollouts | Policy correctness, cost of recomputing and legal/privacy review |
| `reputation_assessments` | Party receives evaluation of verifiable evidence for a period and version | Recognition with insufficient-evidence guard | Evidence manifest and abuse/appeals; do not promote model score to legal trust |
| `reputation_benefit_grants` | Beneficiary has bounded non-security benefit based on optional assessment | Incentives, bounded marketing/analytics privileges | Expiry, revocation, budgets, settlement not implied |

## Four renames and corresponding semantic corrections

- `organization_ownerships` → `organization_primary_administrators`: existing software governance link was mislabeled as legal corporate ownership. A one-primary-admin model itself remains hypothesis; membership same-org FK and authority transfer must be proved.
- `property_types` → `property_categories`: separates user-facing taxonomy from legal land-use classification and immutable physical subtype.
- `property_house_details` → `property_houses`; `property_apartment_details` → `property_apartments`: naming refinement only; proof of property category/subtype validity remains open.

## Selected end-to-end impact vectors (arrows)

1. **Enterprise-first entrant**: verified company → invitation/email/IdP → Account/Person resolution → Membership → Team Assignment → scoped CRM → audit → offboarding/recovery.
2. **Organization claim**: Account assertion → claimed legal registry identity → registration/claim request → lawful evidence validation → corporate representation versus platform administrator → role grant → dispute/revoke.
3. **Cross-company group**: ABC/XYZ representatives → Group Organization Participation → shared listing/deal subset → explicit object ACL → separate ABC CRM → isolated export/notification.
4. **Internal hierarchy**: Organization → Branch AND Team → Membership and assignment → resource rule → CRM record allocation → report. Never derive Team from Branch by name.
5. **Private owner seller**: Personal Account → verified capacity → Property and relevant legal/right-of-offer evidence → Listing Publication authorization → audience and discovery → inquiry.
6. **Broker for company**: Person credentials/qualification → Membership Organization → profession-specific rights → specific authorization per Property → human Actor publishes as Organization → complaint/audit.
7. **Property/parcel**: user location point → uncertainty → authoritative source record/CRS → parcel polygon → evidence-scoped Property link → GIS overlay → report with disclaimer.
8. **Government planning**: issuer/permission/license → dataset version → layer geometry → effective time vs ingestion → match with Property → source manifest → report/notification.
9. **Auction intelligence**: official announcement → evidence and time → map/pin/search → user interest; **not** seller-run bidding or payouts without independent approval.
10. **CRM & group conflict**: same contact Party in ABC and XYZ CRM → tenant record ownership → deal collaboration → grant specifically shared data → no cross-tenant note leakage.
11. **Reputation**: proof of Listing/response quality → anti-gaming validation → versioned score → insufficient evidence/restricted state → badge/benefit → visibility controlled by search relevance.
12. **Ranking**: user intent and geographic filter → eligible Listing pool → quality/relevance → provider signal → paid placements labeled separately → outcome metrics/appeals.
13. **Commission**: parties to agreement → licensed activity and deal proof → participation → terms → earned event → payable owner/payee → payment/reconciliation and reversal.
14. **Enterprise SSO**: personal Account authentication → Org access policy → current IdP assurance → membership/role/resource → action; signed-in personal account alone must not bypass tenant rule.
15. **Departure/rejoin**: HR offboard → membership/role/team revoked → sessions/enterprise access bounded → retention of ABC deals and CRM → Person personal data/reputation retained appropriately.
16. **Community moderation**: member posts → community policy → abuse evidence → content actions → appeal → lawful retention and recommendations.
17. **Advertising offer**: commercial plan/entitlement → sponsored promotion grant → budget/expiry and invoice → user-facing paid label → no fabricated verification or organic score.
18. **Contractual data sharing**: two business Parties → purpose, authority and scope → controlled shared dataset → access/event logs → expiration/revocation → no raw KYC/CRM export by default.

## Decisions and gates

**REVIEW ≠ implementation approval.** A table can be named unchanged while having unresolved cross-row invariants. New V0.2 candidates are logical design sketches, not stable contracts. Every new entity must survive one-row semantics, contradiction cases, write owner, authorization, migration, privacy and operational-cost review.

Immediate gate: **ORG-01 and ORG-NET-01**, including enterprise onboarding for person without prior BDSPro Account, two claimants of ABC, membership/role/Team/Group, staff publishing a property, offboarding to XYZ. C01 Party exact subtype and DB-21 claim proof remain open.

### Practical quantitative success metrics to establish later
- fraction of legitimate enterprise invitations activated without unwanted duplicate Person/Account;
- time to provision and revoke confidential access;
- unauthorized cross-tenant access test pass rate (required 100% for targeted test suite, not a promise of absolute security);
- proportion of published listings with valid actionable authority at moment of publication;
- erroneous claims and dispute correction time;
- verified source age/version and map accuracy/conflict incidence;
- valid lead response rates, spam incidents, complaint upheld rates;
- fake review fraud, insufficient-evidence rate, unfair new provider exposure;
- paid benefit budget used vs qualified customer outcome;
- invoice, commission and settlement reconciliation defects.

### Sources, not templates
- GitHub organization Teams: https://docs.github.com/en/organizations/organizing-members-into-teams/about-teams
- Microsoft Entra B2B: https://learn.microsoft.com/en-us/entra/external-id/what-is-b2b
- HubSpot CRM permissions: https://knowledge.hubspot.com/records/assign-access-to-records
- eBay seller levels: https://www.ebay.com/help/selling/selling-tools/seller-levels-performance-standards?id=4080
- OGC API Features: https://www.ogc.org/standards/ogcapi-features/
- Legacy BDSPro TQD: https://github.com/datvtph41107/bdspro-backend/blob/b2a7a8167e9c8af2a47f1fde4f7b2a9d951611fd/tqd-service/README.md

Do not mistake external product comparisons for evidence of BDSPro business volume, legal access or required microservice boundaries.
