# BDSPro Canonical Database — Final Acceptance Report

Status: **DESIGN ACCEPTED FOR REVIEW / NOT AN EXECUTABLE MIGRATION**

Repository authority: `datvtph41107/bdspro`  
Design branch: `architecture/canonical-database-final`  
Source baseline used: `dev@2a84561cb29dca7f8b198e61d4ad5a6980c2de5e`  
Historical evidence repository: `datvtph41107/bdspro-backend` (evidence only)

Canonical diagram source:

- `docs/database/final/BDSPro-CANONICAL-DATABASE.dbml`

This report closes the requested design pass from **v0 → v1 → v2 → final**.
It does **not** replace the currently executed migrations under `migrations/`.
Those remain live implementation truth until a future implementation slice
deliberately migrates toward this design with expand/backfill/prove/switch/contract.

---

## 1. What "final" means here

"Final" means the database has a complete target vocabulary and relational map
for the known BDSPro product surface, with the dangerous legacy abstractions
removed and the cross-domain invariants written down.

It does not mean:

- every table must be created in the next migration;
- every optional product capability must be enabled immediately;
- the schema can never evolve;
- old data may be dropped because a target table does not preserve a legacy name;
- a DBML relationship by itself grants authorization.

Implementation remains pressure-driven. The target is a map, not permission to
jump over migration proof.

The full DBML currently contains **146 tables** across identity/security,
organization/authorization, collaboration groups, geography, property/project,
listing/market, assets, deals, auctions, CRM, billing/usage, community/social,
chat, appointments, notifications, planning/GIS, media/documents and operational
integration evidence.

---

# 2. Authority and evidence model

The design follows this authority order:

1. live `bdspro` source / Git at an exact commit;
2. reproducible proof/tests for that source;
3. durable continuity documents;
4. historical `bdspro-backend` source as business/evolution evidence;
5. external patterns only as comparative evidence.

The old microservice topology is **not** inherited. Old table names are not
requirements. In particular, this design intentionally does not preserve
"database per service", GORM BaseEntity, generic owner pairs, generic status
columns, universal soft delete, or framework-shaped actor vocabulary.

Representative historical evidence used during the design included:

- `user-service/database/migrations/000008_add_organization_directory.up.sql`
- `user-service/internal/usecase/organization/service.go`
- `organization-service/internal/domain/entity/organization_member.go`
- `organization-service/internal/domain/entity/organization_branch.go`
- `organization-service/internal/domain/entity/deal_member.go`
- `organization-service/internal/usecase/deal_usecase.go`
- `organization-service/internal/usecase/deal_invitation_usecase.go`
- `organization-service/internal/usecase/deal_commission_usecase.go`
- `organization-service/internal/usecase/investment_usecase.go`
- `bdspro-service/infra/db/migrations/011_create_property_tables.sql`
- payment, CRM, notification and planning migrations in their historical services.

---

# 3. v0 — semantic truth

v0 was not a table-generation phase. It answered:

```text
WHAT EXISTS?
WHAT DOES ONE ROW ASSERT?
WHAT IS IDENTITY?
WHAT IS ONLY A PROPOSAL?
WHAT IS AN EFFECTIVE RELATION?
WHAT IS CURRENT STATE VS HISTORY?
WHAT IS AUTHORITY VS BUSINESS RELATIONSHIP?
WHICH LEGACY NAMES ARE LYING?
```

## 3.1 Canonical identity backbone

The stable identity model is:

```text
Party
├── Person
└── Organization

Account -> Person
```

A `Party` is an independently identifiable person or organization capable of
business relationships, rights or obligations.

A `Person` is not an Account. A person can exist before registering.

An `Organization` is not a role, profile, team or branch.

An `Account` is a local digital identity for authentication. Authentication
methods, sessions and external identities do not become business identity.

## 3.2 Runtime actor is not a table

Runtime authorization still follows:

```text
request
 -> caller
 -> authentication
 -> Account / service identity
 -> represented Person
 -> selected Organization context
 -> current Membership
 -> no current MembershipBlock
 -> current RoleAssignment
 -> Permission
 -> resource relationships
 -> business state
 -> policy
 -> ALLOW / DENY
```

No generic `actor_id` or universal Actor table is introduced.

## 3.3 Invitation is not participation

This rule is repeated consistently:

```text
OrganizationInvitation != OrganizationMembership
GroupInvitation        != GroupParticipation
DealInvitation         != DealParticipation
```

Acceptance materializes a new effective relationship. A terminal invitation is
not reopened. Rejoin creates a new participation episode.

## 3.4 Authority is not relationship

Examples:

- Organization Ownership is not an Organization Role.
- Branch Manager is not an Organization-wide "Branch Manager role".
- Deal Lead is not the same fact as Deal business context.
- Deal Customer/Partner is not a permission.
- Branch Assignment does not itself grant Deal access.
- Creator is provenance, not permanent authority.

## 3.5 Core negative schema

The following legacy patterns are rejected as canonical business truth:

```text
owner_type + owner_id
generic (type, id)
generic status for unrelated lifecycle dimensions
BaseEntity on every table
soft delete on every table
created_by / updated_by on every table
is_owner shadow booleans
Profile/User IDs used where Person/Membership/Participation is the correct endpoint
Invitation status mutated into Membership/Participation state
role_key mixing OWNER / ADMIN / MEMBER / CUSTOMER / PARTNER
EAV for core facts
JSONB for facts that have stable relational identity
"catalog" as a top-level business concept
```

JSONB remains allowed for genuinely semi-structured GIS attributes, audit
payloads, integration events and document metadata.

---

# 4. v1 — logical relational model

v1 turns the v0 facts into explicit relations.

## 4.1 Identity and authentication

Canonical tables:

```text
parties
persons
person_profiles
organizations
accounts
account_emails
account_phones
password_credentials
external_identities
passkeys
sessions
service_accounts
service_account_keys
```

Important decisions:

- Person and Organization use Party identity where a business relationship truly
  accepts either.
- email/phone authentication identity is separated from Organization descriptive
  contact fields and CRM contact snapshots.
- OAuth/OIDC identity uses `issuer + subject`, not provider email as identity.
- passkeys are authenticators, not Accounts or Persons.
- API/service credentials are non-human principals and do not masquerade as a Person.

## 4.2 Organization and authorization

Canonical tables:

```text
permissions
organization_roles
organization_role_permissions
organization_invitations
organization_memberships
organization_membership_blocks
organization_role_assignments
organization_ownerships
organization_branches
organization_branch_assignments
organization_branch_locations
```

Key semantic closures:

- Membership is one effective Person ↔ Organization participation episode.
- rejoin creates a new Membership;
- Leave and Remove are different intents but both end Membership;
- Block is a temporary authority restriction while Membership stays current;
- RoleAssignment is scoped to Membership;
- exactly one current RoleAssignment is the normal committed state for a current
  Membership;
- Organization Ownership is a singular current governance relation backed by
  live historical source evidence;
- current Organization owner is a Membership, not a Profile/User scalar;
- Branch is an operational subdivision, not a generic bucket for
  `branch|department|store`;
- Branch member relation targets OrganizationMembership;
- Branch Manager targets OrganizationMembership and is only policy input for
  explicitly allowed Branch intents.

`department` and `store` are intentionally not copied from the legacy
`organization_branches.type` enum. The legacy implementation did not prove they
share the same identity/lifecycle/invariants as Branch.

## 4.3 Groups

Groups remain explicit collaboration contexts because historical Deal flows have
real Group behavior distinct from Organization behavior.

Canonical tables:

```text
groups
group_invitations
group_participations
group_leaderships
group_admin_assignments
```

Group is not automatically promoted to Party. If later evidence proves that a
Group independently bears business rights/obligations outside its collaboration
context, that decision must be reopened explicitly.

## 4.4 Property vs Listing vs Asset

These three facts are intentionally separated:

```text
Property = the real-estate subject/factual record
Listing  = a market/publication offer about one or more Properties
Asset    = portfolio/ownership/economic treatment of a Property
```

That replaces the historical Product/Post/Asset semantic overlap.

Canonical Property/Project tables include:

```text
property_types
properties
property_locations
property_land_parcels
property_house_details
property_apartment_details
property_amenities
property_media
property_documents
property_valuations
real_estate_projects
project_buildings
project_properties
project_documents
```

A Listing may reference multiple Properties through `listing_properties`.
This explicitly supports real cases such as adjacent plots, legal pairs, combined
lots and multi-property offerings without duplicating the real-estate truth.

Listing tables:

```text
listings
listing_properties
listing_price_terms
listing_publications
listing_media
listing_offers
listing_offer_acceptances
```

Price is not one timeless mutable scalar. Listing asking price is a temporal
commercial term. Property valuation is a different fact.

Publication is modeled as an episode. "Draft" is not required to be permanent
stored identity once the final implementation reaches this model; it can be
derived from absence of a current publication plus Listing lifecycle.

Asset tables:

```text
assets
asset_ownerships
asset_valuations
asset_costs
asset_income_entries
asset_leases
asset_documents
```

Asset ownership is an explicit qualified relation. It is not a generic
`owner_id/owner_type`.

## 4.5 Deals

The Deal model removes the largest historical semantic blob.

```text
Deal
├── business context
│   ├── Organization
│   └── Group
├── optional Organization Branch scope
├── Properties
├── Invitations
├── Participations -> Person
├── business relationships
│   ├── Customer
│   └── Partner
├── governance
│   ├── Lead
│   └── Admin
├── commercial terms
│   └── Commission
├── investment
│   ├── Commitment
│   └── actual Investment
├── Contracts
└── Documents
```

Canonical tables:

```text
deals
deal_properties
deal_invitations
deal_participations
deal_customers
deal_partners
deal_admin_assignments
deal_leaderships
deal_commission_terms
deal_commission_earnings
deal_investment_commitments
deal_investments
deal_contracts
deal_contract_customers
deal_contract_properties
deal_contract_payments
deal_documents
```

Important corrections:

- legacy `deals.owner_id + owner_type` is rejected;
- Organization/Group context is not "Deal Owner";
- Branch is a secondary operational scope, not another owner type;
- Deal Participation targets Person;
- `MEMBER` is represented by Participation itself;
- Customer and Partner are Deal-scoped business relationships and may overlap;
- Admin is an authority capacity;
- historical "Deal Owner" is modeled as primary Deal governance/leadership, not
  as the same fact as Organization/Group context;
- `is_owner` is rejected;
- mutable commission terms are separated from earned commission;
- investment commitment is separated from actual received/approved investment;
- contract customers must be Deal participants with Customer relationship.

## 4.6 Auctions

Canonical tables:

```text
auctions
auction_properties
auction_registrations
auction_bids
auction_results
```

Auction bidding is kept distinct from ordinary Listing offers. A winning result
references a concrete immutable bid; it is not reconstructed from a mutable
"highest_bid" column.

## 4.7 CRM

CRM is modeled as an Organization-scoped relationship/workflow layer, not a
second identity universe.

Canonical tables:

```text
crm_contacts
crm_labels
crm_contact_labels
crm_pipelines
crm_stages
crm_opportunities
crm_opportunity_contacts
crm_opportunity_properties
crm_activities
crm_opportunity_deals
```

A CRM contact may optionally link to a canonical Party, but may exist before a
Person/Organization is confidently matched.

An Opportunity is pre-Deal workflow. Promotion to Deal is represented explicitly
through `crm_opportunity_deals`; it does not silently mutate an Opportunity row
into a Deal.

## 4.8 Billing, payment and usage

Canonical tables:

```text
plans
plan_prices
plan_entitlements
subscriptions
invoices
invoice_lines
payments
invoice_payments
payment_attempts
payment_settlements
bank_accounts
wallets
wallet_entries
usage_meters
usage_events
quota_allocations
```

Money always means amount + currency.

Payment business truth is separated from provider attempts and settlement
evidence. Wallet entries are immutable ledger facts. Usage consumption is an
event/ledger, while quota allocation is entitlement for a period.

## 4.9 Community, chat, appointments and notification

Canonical tables include explicit relations rather than generic target pairs:

```text
communities
community_memberships
community_posts
community_post_media
community_comments
community_post_reactions
community_comment_reactions
friendships
follows

conversations
conversation_participations
messages
message_attachments
message_reads

appointments
appointment_participants
appointment_deals
appointment_properties

notifications
notification_deliveries
notification_preferences
```

A chat Message points to the sender's ConversationParticipation, not merely a
global user ID. This preserves the context in which the message was sent.

## 4.10 Planning / GIS

Planning data keeps PostGIS pressure explicit:

```text
planning_sources
planning_datasets
planning_layers
planning_features
planning_labels
planning_feature_labels
planning_documents
planning_reports
planning_report_properties
planning_news
planning_news_saves
```

`planning_features.properties JSONB` is intentional because external GIS
feature attributes are semi-structured and dataset-dependent. Geometry remains a
real PostGIS `geometry` column in the target.

## 4.11 Media and documents

Binary/media identity is shared:

```text
media_objects
documents
document_types
```

But domain relationships stay explicit:

```text
property_media
listing_media
community_post_media
message_attachments

property_documents
project_documents
asset_documents
deal_documents
planning_documents
```

There is no canonical `attachments(owner_type, owner_id)` table.

---

# 5. v2 — PostgreSQL enforcement model

v2 is where logical truth becomes enforceable PostgreSQL behavior.

The DBML can express normal FKs and many unique keys, but several canonical
invariants require partial indexes, CHECKs or transactional validation.

## 5.1 Current-row uniqueness

Representative PostgreSQL enforcement:

```sql
CREATE UNIQUE INDEX uq_accounts_one_open_per_person
ON accounts(person_id)
WHERE closed_at IS NULL;

CREATE UNIQUE INDEX uq_org_memberships_one_current
ON organization_memberships(organization_id, person_id)
WHERE ended_at IS NULL;

CREATE UNIQUE INDEX uq_org_membership_blocks_one_current
ON organization_membership_blocks(membership_id)
WHERE ended_at IS NULL;

CREATE UNIQUE INDEX uq_org_role_assignments_one_current
ON organization_role_assignments(membership_id)
WHERE ended_at IS NULL;

CREATE UNIQUE INDEX uq_group_participations_one_current
ON group_participations(group_id, person_id)
WHERE ended_at IS NULL;

CREATE UNIQUE INDEX uq_deal_participations_one_current
ON deal_participations(deal_id, person_id)
WHERE ended_at IS NULL;

CREATE UNIQUE INDEX uq_listing_price_terms_one_current
ON listing_price_terms(listing_id)
WHERE effective_to IS NULL;

CREATE UNIQUE INDEX uq_listing_publications_one_current
ON listing_publications(listing_id)
WHERE ended_at IS NULL;

CREATE UNIQUE INDEX uq_deal_investment_commitments_one_current
ON deal_investment_commitments(participation_id)
WHERE effective_to IS NULL;
```

## 5.2 Exactly-one / XOR checks

Required checks include:

- OrganizationInvitation target: exactly one of
  `invitee_person_id / invitee_email / invitee_phone`.
- Deal business context: exactly one of `organization_id / group_id`.
- Commission term: exactly one of percent or fixed amount.
- Payment terminal timestamps must not describe incompatible terminal outcomes.
- Investment cannot be both approved and rejected.

Representative shape:

```sql
CHECK (
  ((organization_id IS NOT NULL)::int +
   (group_id IS NOT NULL)::int) = 1
)
```

## 5.3 Money constraints

For business money:

```text
amount NUMERIC(19,4)
currency_code CHAR/VARCHAR(3)
amount >= 0 or > 0 according to the fact
```

Floating-point values are rejected for canonical money.

Percentage commission should be bounded:

```text
0 <= percent_value <= 100
```

## 5.4 Same-context invariants

The following relationships must never cross context:

- Organization RoleAssignment role.organization = membership.organization;
- Organization Ownership membership.organization = ownership.organization;
- Branch manager/assignment Membership belongs to Branch Organization;
- Deal Branch belongs to Deal Organization context;
- Deal Leadership Participation belongs to that Deal;
- Contract Customer Participation belongs to the Contract Deal;
- Investment Participation belongs to the Investment Deal;
- Auction bid Registration belongs to the same Auction;
- CRM Stage belongs to the selected Pipeline and Organization.

Where a natural FK cannot enforce the rule directly, the physical pass may use:

1. composite candidate keys + composite FKs;
2. a narrow trigger for a database invariant;
3. one application transaction that locks the stable coordinator row and re-reads
   authoritative state.

Do not duplicate business truth merely to avoid writing the transaction.

## 5.5 Transaction coordinators

High-value concurrent transitions use the narrowest stable coordinator:

```text
Organization governance mutation -> lock Organization
Branch governance mutation       -> lock Branch
Deal leadership/member mutation  -> lock Deal
Invitation acceptance            -> lock Invitation + validate target/current relation
Auction close                     -> lock Auction
Payment finalization             -> lock Payment/order/ledger boundary as required
```

## 5.6 Immutable records

Prefer append-only/immutable rows for:

- commission earnings;
- actual investments once financially confirmed;
- auction bids;
- wallet entries;
- usage events;
- audit events;
- outbox/inbox event evidence.

Corrections occur through compensating facts, not silent historical rewrites.

---

# 6. Final cross-domain invariants

The final review must preserve these invariants.

## Identity

- A Person is business identity even without an Account.
- An Account belongs to one Person.
- At most one open local Account per Person unless future business evidence
  explicitly earns multiple.
- OAuth/OIDC uniqueness is `issuer + subject`.
- passkey credential ID is globally unique.

## Organization

- A current Membership has no `ended_at`.
- At most one current Membership per Organization+Person.
- A blocked Membership is still current but cannot exercise Organization
  authority.
- Current Membership normally has exactly one current RoleAssignment.
- Active/non-archived Organization has exactly one current Ownership relation.
- Owner Membership belongs to the Organization and must be usable under the
  current governance policy.
- Organization archive terminates current authority relations atomically.
- Branch manager and assignments reference current Memberships of the same
  Organization.
- Closed Branch accepts no new current assignments or new Branch-scoped Deal.

## Group

- At most one current GroupParticipation per Group+Person.
- Current Group Lead is a current Participation of the same Group.
- Group admin is independent from lead identity.

## Property / Listing / Asset

- Property is not Listing.
- Listing may market multiple Properties.
- Property may appear in multiple Listings over its lifetime.
- Listing has at most one current asking-price term.
- Listing has at most one current publication episode.
- Accepted Listing offer must belong to that Listing.
- Asset is an economic/portfolio representation of a Property, not the Property
  identity itself.

## Deal

- Deal has exactly one business context: Organization or Group.
- Branch may exist only for Organization-context Deal and must belong to that
  Organization.
- At most one current DealParticipation per Deal+Person.
- Invitation acceptance creates exactly one originating Participation.
- Customer and Partner can overlap.
- Admin and Customer/Partner are independent axes.
- Open Deal has exactly one current Lead once the leadership feature is enabled.
- Lead is a current Participation in the same Deal.
- Commission term is mutable commercial configuration; CommissionEarning is
  earned financial fact.
- InvestmentCommitment is a promise/term; Investment is actual money/evidence.
- Contract customer must be a Deal Customer participation.
- Contract Property must be a Property in the Deal scope unless a deliberate
  broader rule is later introduced.

## Auction

- Registration is unique per Auction+Party.
- Bid registration and bid Auction must match.
- Winning bid belongs to the Auction.
- Closed result is immutable except explicit legal/admin correction workflow.

## CRM

- Contact/Pipeline/Opportunity are Organization-scoped.
- Stage must belong to Opportunity Pipeline.
- A canonical Person/Organization may be linked to a CRM Contact, but CRM data
  does not replace Party identity.
- Opportunity→Deal promotion is explicit.

## Finance

- Money always includes currency.
- Provider attempt != Payment != Settlement.
- WalletEntry is immutable.
- WalletEntry currency matches Wallet currency.
- UsageEvent is immutable and idempotent by operation key.
- Quota allocation and consumption are distinct facts.

## Chat/social

- Message sender is a ConversationParticipation in the same Conversation.
- Read receipt participant is in the same Conversation.
- Friend/follow self-relations are forbidden.
- Social reaction relations are explicit; no generic `target_type/target_id`
  canonical table.

## Operational evidence

- audit/outbox/inbox generic subject identifiers are non-authoritative.
- They must never be queried as the source of ownership, membership or resource
  identity.

---

# 7. What is intentionally NOT in the final canonical core

The following legacy concepts were reviewed and intentionally not carried forward
as canonical business tables/columns:

| Legacy concept | Final treatment |
| --- | --- |
| Organization `code` | Rejected until a stable immutable human reference is actually required |
| Organization `type=business/team/other` | Rejected; conflates ontology |
| Organization generic `status` | Replaced by explicit lifecycle facts such as `archived_at` |
| Organization `owner_profile_id` | Replaced by `organization_ownerships -> Membership` |
| Membership `status=invited/active/suspended/removed` | Split Invitation / Membership / MembershipBlock / ended_at |
| Organization `warning_level/count` | Not a core Organization fact |
| Organization `verification_status` | Dedicated verification capability required before adding |
| `organization_branches.type=branch/department/store` | Rejected as one polymorphic bucket |
| Branch `user_id` member links | Replaced by Membership-targeted assignment |
| Branch-scoped `role_id` | Not earned |
| Deal `owner_id + owner_type` | Rejected |
| Deal `is_owner` | Rejected duplicate/shadow truth |
| Deal `role_key OWNER/ADMIN/MEMBER/CUSTOMER/PARTNER` | Split participation, business relationship and governance |
| Deal `status INVITED/ACCEPTED/REJECTED/WITHDRAWN` on member row | Split Invitation and Participation |
| Deal `done_investment` on member row | Derived/projection from investment facts |
| Product/Post/Asset as interchangeable records | Replaced by Property / Listing / Asset distinction |
| Generic attachments `type/id` | Replaced by domain-specific joins to media/documents |
| Generic EAV core attributes | Rejected |
| Universal soft delete | Rejected |
| Universal BaseEntity timestamps | Rejected |
| Top-level Catalog | Rejected |

---

# 8. Migration strategy from legacy

Migration is not a rename exercise.

Required strategy:

```text
OLD
 -> EXPAND canonical tables alongside legacy
 -> BACKFILL with classification rules
 -> COMPARE old reads vs canonical projections
 -> PROVE invariants and counts
 -> SWITCH write authority
 -> SWITCH reads
 -> CONTRACT legacy columns/tables only after proof
```

Important reconciliation jobs:

## Organization

- reconcile scalar owner vs owner membership/role before backfill;
- split invited rows from accepted Memberships;
- reconstruct current RoleAssignment;
- map temporary inactive/suspended states into explicit Blocks only when the
  source semantics prove temporary authority restriction;
- preserve archival history;
- do not release historical tax-code uniqueness merely because Organization is
  archived.

## Branch

- preserve Branch rows;
- classify legacy Department/Store rows instead of blindly importing them as
  Branch;
- map branch users to canonical OrganizationMembership;
- repair duplicate branch-member rows;
- reconcile manager IDs to Memberships.

## Deal

- profile actual `owner_type` values before mapping;
- separate Organization/Group context;
- split invitation from effective participation;
- detect role-key collisions caused by legacy dedupe;
- compare `role_key=OWNER` and `is_owner` disagreements;
- backfill Customer/Partner as independent relations;
- derive Lead candidate from the strongest consistent source;
- migrate commission terms separately from historical earned/paid amounts;
- map investment records without trusting `done_investment` as source truth.

## Property / Listing / Asset

- classify historical Product rows into Property truth;
- treat Post as Listing/publication evidence, not Property identity;
- preserve price history where recoverable;
- map Asset ownership/economic records separately from Property;
- use explicit many-to-many Listing↔Property where a legacy listing represented
  multiple real-estate subjects.

---

# 9. Implementation order

The final diagram is broad, but implementation remains incremental.

Recommended source/migration order:

```text
Gate 1 — Identity backbone
  parties -> persons -> organizations -> accounts/auth

Gate 2 — Organization authority
  invitations -> memberships -> blocks -> roles -> ownership -> branches

Gate 3 — Property truth
  property types -> properties -> location/legal/detail/media/documents

Gate 4 — Listing
  listing_properties -> price terms -> publication -> media/offers

Gate 5 — Deal
  context -> invitation -> participation -> customer/partner -> lead/admin
  -> commission -> investment -> contract

Gate 6 — Asset/project/auction/CRM

Gate 7 — Billing/payment/usage

Gate 8 — Community/chat/appointments/notifications

Gate 9 — Planning/PostGIS

Gate 10 — audit/outbox/idempotency integration hardening
```

Each gate must have:

```text
migration
 -> sqlc queries
 -> application transaction
 -> integration proof
 -> concurrency/failure proof where required
 -> only then next gate
```

Do not create all 146 tables in one production migration.

---

# 10. Current live Listing source vs final target

The live `dev` branch currently has a deliberately small learning schema:

```text
listings
listing_publications
description
price
status DRAFT/PUBLISHED
```

That code remains valid learning/runtime evidence.

The final target intentionally evolves it:

```text
listings
 -> listing_properties
 -> listing_price_terms
 -> listing_publications as temporal publication episodes
 -> listing_media
 -> listing_offers
```

A future implementation must migrate this step-by-step. This acceptance artifact
does not authorize rewriting the current code/migrations wholesale.

---

# 11. Final acceptance checklist

The design is accepted when review confirms all of the following:

- [x] Person/Organization business identity is independent from Account.
- [x] Invitation and effective participation are separate in Organization, Group and Deal.
- [x] Organization Ownership has one canonical owner relation.
- [x] Branch member/manager endpoints use OrganizationMembership.
- [x] Branch is not a `branch|department|store` polymorphic bucket.
- [x] Deal context is not represented by generic owner type/id.
- [x] Deal Customer/Partner/Admin/Lead are not one role enum.
- [x] Commission terms are distinct from earned commission.
- [x] Investment commitment is distinct from actual investment.
- [x] Property, Listing and Asset have separate identities.
- [x] Listing supports multiple Properties.
- [x] Prices are domain-qualified and temporal where change matters.
- [x] Money has currency.
- [x] Media/Document objects are reusable while domain links stay explicit.
- [x] CRM does not create a second identity universe.
- [x] Payments separate business payment, provider attempt and settlement.
- [x] Operational generic references are non-authoritative only.
- [x] No universal soft delete/BaseEntity/status/owner abstraction remains.
- [x] PostGIS remains explicit for planning spatial data.
- [x] Migration follows expand/backfill/compare/prove/switch/contract.
- [x] Current live Listing implementation is preserved rather than overwritten by the design artifact.

---

# 12. Acceptance result

**v0: CLOSED for the canonical fact model represented by this report.**  
**v1: CLOSED as a complete logical target map for the known BDSPro product surface.**  
**v2: CLOSED as the PostgreSQL enforcement strategy and migration contract.**  
**FINAL: ACCEPTED AS THE CANONICAL DATABASE REVIEW BASELINE.**

The next phase is not "design more tables by default". The next phase is a full
human review of the DBML/report, then implementation gate-by-gate from live
source.

Any future schema change must identify:

```text
new business fact or changed invariant
 -> why the current canonical model cannot represent it safely
 -> smallest schema change
 -> migration/proof plan
```

That is the completion criterion for this database design initiative.
