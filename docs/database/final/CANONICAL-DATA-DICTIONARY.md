# BDSPro Canonical Database — Data Dictionary

Status: **FINAL REVIEW BASELINE**

This dictionary gives every table in
`BDSPro-CANONICAL-DATABASE.dbml` one primary business sentence. If a future
column or relationship cannot be explained from that sentence or a documented
invariant, it must not be added casually.

Notation:

- **identity** — what makes one row a distinct durable fact;
- **current** — row represents current relation/state;
- **episode** — row has a meaningful start/end lifecycle;
- **immutable fact** — append/correct by compensating fact rather than rewriting history;
- **reference** — controlled vocabulary/configuration, not an actor.

---

# Identity & Security

| Table | One-row assertion / role |
| --- | --- |
| `parties` | This ID identifies one independent business party capable of holding relationships, rights or obligations. It says nothing yet about whether the party is a Person or Organization. |
| `persons` | Party P is a physical human business identity. PK = Party ID. |
| `person_profiles` | Person P currently has this presentation/display profile. It is descriptive, not authentication or business identity. |
| `organizations` | Party O is an Organization with current descriptive/legal-contact facts and an optional terminal archive time. |
| `accounts` | Account A is a local digital login identity belonging to Person P. `closed_at` terminates that local account. |
| `account_emails` | Account A currently knows email E with verification/primary metadata. Email is not Person identity. |
| `account_phones` | Account A currently knows phone N with verification/primary metadata. Phone is not Person identity. |
| `password_credentials` | Account A currently authenticates with this password hash/version point. The hash is an authenticator secret representation, not profile data. |
| `external_identities` | External issuer I says subject S is linked to Account A. Identity key = issuer + subject. |
| `passkeys` | Passkey credential C is registered to Account A with this public key and counter. |
| `sessions` | Session S authorizes a bounded login session for Account A until expiry/revocation. |
| `service_accounts` | ServiceAccount S is a non-human principal, optionally scoped to Organization O. It does not create a fake Person. |
| `service_account_keys` | Key K is one credential for ServiceAccount S with explicit issue/expiry/revocation lifecycle. |

Key invariants:

- Person can exist without Account.
- At most one open local Account per Person in the accepted baseline.
- external identity uniqueness is `issuer + subject`.
- passkey credential ID and session token hash are globally unique.
- primary email/phone is a per-Account constraint, not global Person identity.

---

# Organization & Authorization

| Table | One-row assertion / role |
| --- | --- |
| `permissions` | Code C names one stable system authorization primitive. It is not a final allow/deny decision. |
| `organization_roles` | Role R is a reusable authority bundle owned by Organization O. |
| `organization_role_permissions` | Role R contains Permission P. PK is the pair. |
| `organization_invitations` | Organization O issued one proposal to one target to join with intended Role R during this invitation lifecycle. |
| `organization_memberships` | Person P effectively participates in Organization O during this Membership episode. |
| `organization_membership_blocks` | Membership M is temporarily prevented from exercising Organization authority during this block episode. Membership itself remains current until ended separately. |
| `organization_role_assignments` | Membership M holds Organization Role R during this role-assignment episode. |
| `organization_ownerships` | Organization O is currently governed/owned by Membership M. This is the singular current ownership truth. |
| `organization_branches` | Branch B is one operational subdivision of Organization O, optionally with one current manager Membership and a terminal close time. |
| `organization_branch_assignments` | Organization Membership M is currently assigned to Branch B. This relation alone does not grant arbitrary resource permission. |
| `organization_branch_locations` | Branch B currently has this physical/geographic location description. Structural Branch identity and physical location remain separate facts. |

Important boundaries:

- Invitation ≠ Membership.
- Membership ≠ Role.
- Role ≠ Ownership.
- MembershipBlock ≠ Membership termination.
- Branch Manager ≠ Organization-wide role.
- Branch Assignment is policy input, not automatic permission.

---

# Groups

| Table | One-row assertion / role |
| --- | --- |
| `groups` | Group G is a collaboration context with current descriptive data and optional end time. It is not automatically a Party. |
| `group_invitations` | Group G issued one proposal for Person P to join. |
| `group_participations` | Person P effectively participates in Group G during this episode. |
| `group_leaderships` | Group G currently has Participation P as its singular lead/governance holder. |
| `group_admin_assignments` | Participation P currently has Group-admin capacity in Group G. |

A Person may leave/rejoin by ending one Participation and creating another.
Leadership and admin capacity are independent.

---

# Location / Media / Document Reference

| Table | One-row assertion / role |
| --- | --- |
| `administrative_areas` | Area A is one versioned administrative/geographic unit with optional parent. This supports hierarchy changes without hard-coding province/district/ward columns as ontology. |
| `amenities` | Amenity A is one controlled property-facility concept. |
| `document_types` | Code C classifies a business document kind. |
| `media_objects` | Media M is one stored binary/object identified by storage key and technical metadata. |
| `documents` | Document D is one business document backed by Media M and classified by DocumentType. |

A Document is not just a URL. A MediaObject is not automatically a legal/business
document.

---

# Property

| Table | One-row assertion / role |
| --- | --- |
| `property_types` | Code C names one real-estate type in a hierarchy. |
| `properties` | Property P is one durable real-estate subject/factual identity. It is not a Listing and not an Asset. |
| `property_locations` | Property P currently has this geographic/address placement. |
| `property_land_parcels` | Row L is one legal/physical land parcel component of Property P. Multiple rows allow one Property representation to cover paired/adjacent parcels when business truth requires it. |
| `property_house_details` | Property P has these current house/building-specific physical facts. |
| `property_apartment_details` | Property P has these current apartment/unit-specific facts. |
| `property_amenities` | Property P is associated with Amenity A. |
| `property_media` | Media M describes Property P with an ordering/cover role. |
| `property_documents` | Document D is evidence/documentation about Property P. |
| `property_valuations` | At time T, Property P was valued at Amount+Currency by an optional Party, with note/evidence context. This is historical valuation, not asking price. |

Subtype-detail tables are explicit because their columns have stable semantics;
the design rejects generic EAV for these core property facts.

---

# Projects

| Table | One-row assertion / role |
| --- | --- |
| `real_estate_projects` | Project R is one real-estate development/project, optionally developed by Organization O. |
| `project_buildings` | Building B is one building/tower within Project R. |
| `project_properties` | Property P belongs to Project R and may be located in Building B. |
| `project_documents` | Document D belongs to/evidences Project R. |

A Developer is reused as Organization when it is truly an independent business
organization; no duplicate Developer identity is required.

---

# Listings / Market Exposure

| Table | One-row assertion / role |
| --- | --- |
| `listings` | Listing L is one market-facing offer/exposure created by Publisher Party P, with its own lifecycle. It is not the Property itself. |
| `listing_properties` | Listing L markets/includes Property P. Multiple Properties per Listing are allowed deliberately. |
| `listing_price_terms` | During interval [from,to), Listing L asks Amount+Currency. One current term is expected. |
| `listing_publications` | Publication episode E makes Listing L published during its publication interval. |
| `listing_media` | Media M is presentation media for Listing L. |
| `listing_offers` | Party P made one monetary Offer O against Listing L at time T, optionally expiring/withdrawing later. |
| `listing_offer_acceptances` | Listing L accepted exactly Offer O at time T. This is singular current acceptance truth for the accepted-market-offer model. |

Price changes create/close price-term episodes instead of rewriting historical
asking price. Publication history is distinct from Listing identity.

---

# Assets / Portfolio

| Table | One-row assertion / role |
| --- | --- |
| `assets` | Asset A is the portfolio/economic representation of Property P. It is not Property identity. |
| `asset_ownerships` | Party P holds an ownership share in Asset A during this ownership episode. |
| `asset_valuations` | Asset A had portfolio value Amount+Currency at time T. |
| `asset_costs` | Cost C is one incurred monetary expense for Asset A. Immutable accounting/business fact after confirmation. |
| `asset_income_entries` | Income I is one received monetary income fact for Asset A. |
| `asset_leases` | Asset A is leased to Tenant Party P for the specified interval and rent term. |
| `asset_documents` | Document D documents Asset A. |

Cached ROI/total-income/total-cost values are projections over durable entries,
not independent mutable truth.

---

# Deals

| Table | One-row assertion / role |
| --- | --- |
| `deals` | Deal D is one business transaction/work context, contained by exactly one Organization or Group, optionally scoped to one Organization Branch. |
| `deal_properties` | Property P is in scope of Deal D. |
| `deal_invitations` | Deal D issued one proposal to Person P to participate, with optional message and terminal outcome. |
| `deal_participations` | Person P effectively participates in Deal D during this participation episode. |
| `deal_customers` | DealParticipation P currently acts in Customer relationship for its Deal. |
| `deal_partners` | DealParticipation P currently acts in Partner relationship for its Deal. |
| `deal_admin_assignments` | DealParticipation P currently holds Deal-admin authority. |
| `deal_leaderships` | Deal D currently has Participation P as its primary governance/lead holder. |
| `deal_commission_terms` | Participation P currently has this negotiated commission term: percentage or fixed money. |
| `deal_commission_earnings` | Participation P earned immutable Commission amount A in Deal D at time T, optionally later settled. |
| `deal_investment_commitments` | Participation P promises/commits Amount+Currency during this effective interval. |
| `deal_investments` | Participation P actually submitted/paid one Investment amount into Deal D with approval/rejection evidence. |
| `deal_contracts` | Contract C records one contractual transaction fact within Deal D, including agreed amount and lifecycle milestones. |
| `deal_contract_customers` | Customer Participation P is a party/customer to Contract C. |
| `deal_contract_properties` | Property P is covered by Contract C. |
| `deal_contract_payments` | Payment fact X applies Amount+Currency from Payer Party to Contract C at time T and may link to canonical Payment. |
| `deal_documents` | Document D belongs to/evidences Deal D. |

Critical axes remain independent:

```text
Participation existence
Customer / Partner business relationship
Admin authority
Lead governance
Commission term
Investment commitment
actual investment
```

The design therefore has no single `role_key` capable of overwriting these
independent truths.

---

# Auctions

| Table | One-row assertion / role |
| --- | --- |
| `auctions` | Auction A is one time-bounded auction process run by Seller Party P. |
| `auction_properties` | Property P is offered in Auction A. |
| `auction_registrations` | Bidder Party P registered once to participate in Auction A. |
| `auction_bids` | Registration R placed immutable monetary Bid B in Auction A at time T. |
| `auction_results` | Auction A closed with Winning Bid B at time T. |

The winning result references a bid; no mutable highest-bid scalar becomes
historical authority.

---

# CRM

| Table | One-row assertion / role |
| --- | --- |
| `crm_contacts` | Organization O maintains Contact C as a CRM relationship record, optionally linked to a known canonical Party. |
| `crm_labels` | Label L is an Organization-scoped CRM classification. |
| `crm_contact_labels` | Contact C has Label L. |
| `crm_pipelines` | Pipeline P is one Organization-scoped sales/customer workflow. |
| `crm_stages` | Stage S is one ordered stage inside Pipeline P. |
| `crm_opportunities` | Opportunity O is one active/closed business opportunity moving through a Pipeline/Stage. |
| `crm_opportunity_contacts` | CRM Contact C participates in Opportunity O. |
| `crm_opportunity_properties` | Property P is relevant to Opportunity O. |
| `crm_activities` | Activity A records one CRM interaction/event in Organization context, optionally against Contact/Opportunity. |
| `crm_opportunity_deals` | Opportunity O has been explicitly promoted/linked to canonical Deal D. |

CRM Contact is allowed to exist without a canonical Party match. This prevents
inventing identity from phone-book data.

---

# Subscription / Billing / Payment / Usage

| Table | One-row assertion / role |
| --- | --- |
| `plans` | Plan P defines one sellable subscription package identity. |
| `plan_prices` | During interval [from,to), Plan P costs Amount+Currency for BillingPeriod B. |
| `plan_entitlements` | Plan P grants entitlement code E with value V. |
| `subscriptions` | Party P subscribes to Plan X during this subscription lifecycle. |
| `invoices` | Invoice I states an amount-due document/lifecycle for Subscriber Party P, optionally tied to Subscription S. |
| `invoice_lines` | Line L contributes Quantity × UnitAmount to Invoice I. |
| `payments` | Payment P is one business money-transfer operation from Payer Party to Payee Party with explicit terminal lifecycle. |
| `invoice_payments` | Payment P settles/applies to Invoice I. |
| `payment_attempts` | Attempt A is one provider execution attempt for Payment P with idempotency/provider reference evidence. |
| `payment_settlements` | Settlement S is provider/bank evidence that Payment P settled Amount+Currency at time T. |
| `bank_accounts` | Party P owns/uses one linked bank account represented securely. |
| `wallets` | Party P has one wallet ledger account for Currency C. |
| `wallet_entries` | Entry E is one immutable debit/credit ledger fact in Wallet W. |
| `usage_meters` | Meter M names one durable usage dimension. |
| `usage_events` | Party P consumed Quantity Q of Meter M for unique OperationKey K at time T. |
| `quota_allocations` | Party P is allocated Limit L for Meter M during Period [start,end). |

Provider attempt, Payment, Settlement and WalletEntry are separate facts.
A mutable wallet balance may be projected but is not a substitute for ledger
history.

---

# Community / Social

| Table | One-row assertion / role |
| --- | --- |
| `communities` | Community C is one social/community space. |
| `community_memberships` | Person P participates in Community C during this membership episode. |
| `community_posts` | Party P authored one social Post in optional Community C at time T. This is not a real-estate Listing. |
| `community_post_media` | Media M is attached to CommunityPost P. |
| `community_comments` | Person P authored Comment C on Post X, optionally replying to another Comment. |
| `community_post_reactions` | Person P currently records Reaction R to Post X. |
| `community_comment_reactions` | Person P currently records Reaction R to Comment C. |
| `friendships` | Persons A and B have one friendship request/accept/end lifecycle, originally requested by one party. |
| `follows` | Person A currently follows Person B from time T. |

No generic social `target_type + target_id` table is used for canonical reactions.

---

# Chat

| Table | One-row assertion / role |
| --- | --- |
| `conversations` | Conversation C is one chat thread/space. |
| `conversation_participations` | Person P participates in Conversation C during this episode, with optional mute setting. |
| `messages` | Participation P sent Message M inside its Conversation at time T; reply/recalled lifecycle is explicit. |
| `message_attachments` | Media M is attached to Message X. |
| `message_reads` | ConversationParticipation P read Message M at time T. |

Sender and reader target the context identity (ConversationParticipation), not
just a global Person/User ID.

---

# Appointments

| Table | One-row assertion / role |
| --- | --- |
| `appointments` | Organizer Person P scheduled Appointment A for one time interval/location, optionally later canceled. |
| `appointment_participants` | Person P is invited/associated with Appointment A and may record a response. |
| `appointment_deals` | Appointment A concerns Deal D. |
| `appointment_properties` | Appointment A concerns Property P. |

Deal/Property links stay explicit instead of a generic appointment target.

---

# Notifications

| Table | One-row assertion / role |
| --- | --- |
| `notifications` | Person P should be informed of Topic T with this notification content; read state belongs to this recipient notification. |
| `notification_deliveries` | Delivery D is one attempt to deliver Notification N through Channel C. |
| `notification_preferences` | Person P currently enables/disables Channel C for Topic T. |

Notification addressing is not resource ownership.

---

# Planning / GIS

| Table | One-row assertion / role |
| --- | --- |
| `planning_sources` | Source S identifies one origin/provider of planning data. |
| `planning_datasets` | Dataset D is one effective planning dataset/version sourced from S. |
| `planning_layers` | Layer L is one named spatial layer within Dataset D. |
| `planning_features` | Feature F is one spatial geometry with layer-specific semi-structured properties. |
| `planning_labels` | Label L is one classification vocabulary item for a PlanningLayer. |
| `planning_feature_labels` | PlanningFeature F carries Label L. |
| `planning_documents` | Document X evidences/supports PlanningDataset D. |
| `planning_reports` | Person P requested one planning analysis/report job, with explicit completion/failure evidence and optional result document. |
| `planning_report_properties` | PlanningReport R analyzes/concerns Property P. |
| `planning_news` | News item N is one planning/news source publication. |
| `planning_news_saves` | Person P saved PlanningNews N at time T. |

PostGIS geometry is canonical spatial truth. JSONB is limited to externally
variable feature attributes, not used to hide stable relational facts.

---

# Operational Evidence / Integration

| Table | One-row assertion / role |
| --- | --- |
| `audit_events` | At time T, Actor A performed/caused action X concerning an identified subject, with evidence payload. This row does not become the subject's canonical business state. |
| `outbox_events` | Domain/integration Event E is durably queued for publication from a committed business transaction. |
| `inbox_events` | Consumer C has received/processed integration Event E, preventing duplicate side effects. |
| `idempotency_records` | Within Scope S, request key K has canonical request hash H for an idempotency window. |

Generic subject/aggregate identifiers are deliberately allowed only here because
these tables are operational evidence, not relationship/ownership authority.

---

# Cross-table identity rules

The most important endpoint rule is:

```text
use Person
  when the fact concerns the human independent of a specific participation

use OrganizationMembership
  when the fact exists specifically because the Person participates in an Organization

use GroupParticipation
  when the fact exists specifically inside a Group

use DealParticipation
  when the fact exists specifically inside a Deal

use ConversationParticipation
  when the fact exists specifically inside a Conversation

use Party
  only when both Person and Organization are genuinely valid business participants
```

This prevents the legacy "everything points to user/profile/member_id" failure.

---

# Lifecycle vocabulary rules

Use explicit lifecycle columns only when the lifecycle is part of the business
fact:

```text
joined_at / ended_at        participation episode
blocked_at / ended_at       temporary block episode
assigned_at / ended_at      role assignment episode
published_at / ended_at     publication episode
effective_from / effective_to commercial term
opened_at / closed_at       business process
created_at / completed_at / failed_at operational job/payment
```

Do not add `status` merely because an API wants a status string. A status column
is valid only when one coherent mutually-exclusive lifecycle truly exists.

---

# Money rules

Every canonical money fact uses:

```text
amount
currency_code
```

Never store business money in floating-point.

Different money facts must not be merged merely because they contain the same
number:

```text
Property valuation
Listing asking price
Asset valuation
Investment commitment
actual Investment
Contract agreed amount
Payment
Commission earning
Cost
Income
Bid
Offer
```

Each has different authority and lifecycle.

---

# Final dictionary invariant

For every future schema proposal, complete this sentence first:

> "One row in TABLE means ..."

If that sentence requires "or", "sometimes", an opaque `type`, or an unrelated
status branch, the table is probably mixing facts and must be re-reasoned before
implementation.
