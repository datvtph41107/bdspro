# BDSPro Canonical Database — Legacy Migration Map

> **Current V0.3 design notice:** This map was established for the original 146-table baseline. The live design branch now has 160 candidate tables (4 renamed, 14 additional): see [CORE-IMPACT-MATRIX-V0.2.md](CORE-IMPACT-MATRIX-V0.2.md), [CORE-REVIEW-V0.3.md](CORE-REVIEW-V0.3.md). No data rewrite is authorized; map old ownership/admin semantics carefully, do not mechanically rename persisted columns.


Status: **FINAL DESIGN MIGRATION CONTRACT**

This document maps the historical `datvtph41107/bdspro-backend` data model into
the canonical target in `BDSPro-CANONICAL-DATABASE.dbml`.

It is deliberately a semantic migration map, not a rename list.

## Migration classifications

| Class | Meaning |
| --- | --- |
| COPY | Same durable fact; normalize and copy after validation |
| SPLIT | One legacy row/column encoded several canonical facts |
| MERGE | Several legacy sources duplicate one canonical fact |
| DERIVE | Target fact is reconstructed from stronger existing evidence |
| RECONCILE | Sources may disagree; compare before choosing authority |
| COMPAT | Keep legacy projection/read temporarily during migration |
| DROP_AFTER_PROOF | Remove only after canonical write/read authority is proven |
| REJECT | Do not reproduce the legacy abstraction in the target |
| DEFER | Preserve evidence but do not invent target semantics yet |

The global migration sequence is:

```text
OLD
 -> EXPAND
 -> BACKFILL
 -> COMPARE / PROVE
 -> SWITCH WRITE AUTHORITY
 -> SWITCH READ AUTHORITY
 -> CONTRACT
```

No destructive contraction is allowed before the preceding proof exists.

---

# 1. Identity / authentication

## Historical sources

Typical historical concepts:

```text
profile / user
auth_method
session / user_session
phone / email
password
OAuth/provider identity
passkey/authenticator
role attached to auth/profile
```

## Canonical mapping

| Legacy | Class | Canonical target | Required reconciliation |
| --- | --- | --- | --- |
| profile/user business identity | SPLIT | `parties` + `persons` + optional `person_profiles` | Separate human identity from presentation and login |
| local login/account | SPLIT | `accounts` | Do not treat email/phone/provider as Person identity |
| auth email | COPY/SPLIT | `account_emails` | Normalize carefully; preserve verification evidence |
| auth phone | COPY/SPLIT | `account_phones` | Normalize carefully; preserve verification evidence |
| password hash | COPY | `password_credentials` | Copy only current supported hash; never plaintext |
| OAuth/OIDC provider identity | COPY | `external_identities` | Canonical unique identity is issuer + subject |
| passkey credential | COPY | `passkeys` | Preserve credential ID/public key/sign counter |
| session | COPY | `sessions` | Migrate only unexpired/revocation-safe sessions if operationally needed |
| API/service principal | SPLIT | `service_accounts` + `service_account_keys` | Never create fake Person rows |
| auth role/role_key | REJECT/RECONCILE | Organization authorization model where appropriate | Authentication identity must not own business authority |

## Proof queries / checks

Before switching:

```text
profiles mapped to exactly one Person
open Accounts do not accidentally duplicate one Person
OAuth issuer+subject collisions = 0
passkey credential-id collisions = 0
verified email/phone evidence preserved
```

---

# 2. Organization directory and governance

Historical authority includes
`user-service/database/migrations/000008_add_organization_directory.up.sql`
and organization-service source.

## Organization row

| Legacy | Class | Canonical |
| --- | --- | --- |
| organizations.id | COPY with identity translation | `parties.id` + `organizations.party_id` |
| name | COPY | `organizations.name` |
| tax_code | COPY/RECONCILE | `organizations.tax_code` |
| phone/email/website/description | COPY | same descriptive fields |
| address/detail_address | DEFER | no canonical Organization address until semantics are proven |
| code | REJECT | no generic Organization code |
| type=business/team/other | REJECT | no mixed ontology flag |
| status active/locked/archived | SPLIT | `archived_at`; temporary lock requires dedicated evidence |
| archived_at | COPY | `organizations.archived_at` |
| verification_status | DEFER | dedicated verification capability if required |
| warning_level/warning_count | REJECT from core | risk/moderation subsystem if later earned |
| owner_profile_id | RECONCILE | `organization_ownerships.membership_id` |
| created_by/updated_by | DROP_AFTER_PROOF from core | audit event if provenance is required |

Tax-code backfill must not preserve the legacy partial uniqueness rule that
released the identifier after archive. A legal identifier must be reconciled
across current + archived history.

## Organization invitation / membership

Legacy `organization_members` conflates:

```text
invitation
effective participation
role
temporary suspension
terminal removal
```

Mapping:

| Legacy state/fact | Class | Canonical |
| --- | --- | --- |
| invited row | SPLIT | `organization_invitations` |
| accepted/current row | SPLIT | `organization_memberships` |
| role_key/role_id | SPLIT | `organization_role_assignments` |
| suspended/inactive temporary authority | SPLIT/RECONCILE | `organization_membership_blocks` only when temporary-block semantics are provable |
| removed / leave | SPLIT | `organization_memberships.ended_at` |
| joined_at | RECONCILE | effective Membership start, not invitation issuance |
| removed_at | RECONCILE | Membership terminal time |
| old soft-delete | REJECT as business truth | only migration evidence |

Required classification algorithm:

```text
if row never became effective:
    classify as Invitation evidence
else:
    materialize Membership episode

if a temporary restriction exists independently of membership termination:
    materialize MembershipBlock episode

translate current effective role into one RoleAssignment
```

Never create Membership merely because an admin typed a target ID into an
invitation.

## Organization roles/permissions

| Legacy | Class | Canonical |
| --- | --- | --- |
| roles | RECONCILE/COPY | `organization_roles` |
| role_permissions | COPY | `organization_role_permissions` |
| permissions | RECONCILE/COPY | `permissions` |
| role_profiles / role membership | SPLIT | `organization_role_assignments` |
| is_default, allow_assign, generic domain_type | DEFER/REJECT | add only after separate business pressure |
| owner role | REJECT as ownership source | `organization_ownerships` is authority |

Current Membership should have one current RoleAssignment in normal committed
state.

## Organization ownership

Historical sources may disagree:

```text
organizations.owner_profile_id
organization-service OwnerId
organization_members role_key='owner'
admin/owner authorization fallback
```

Migration classification = **RECONCILE**.

For every non-archived Organization:

1. resolve the legacy scalar owner;
2. resolve the matching current Membership;
3. compare owner-role evidence;
4. flag disagreement;
5. create exactly one `organization_ownerships` row only after resolution.

Do not backfill multiple owners merely because several rows were incorrectly
marked owner. Live historical source uses singular owner semantics.

---

# 3. Organization Branch / Department / Store

Historical `organization_branches` uses:

```text
type = branch | department | store
address
phone/email
manager_id
is_active
soft delete
members
deals
```

This is not imported as one polymorphic canonical entity.

| Legacy | Class | Canonical |
| --- | --- | --- |
| type=branch rows with proven operational branch behavior | COPY/RECONCILE | `organization_branches` |
| type=department | DEFER | preserve source; do not call it Branch automatically |
| type=store | DEFER | classify physical-site vs business-unit semantics first |
| branch member user_id | SPLIT | `organization_branch_assignments.membership_id` |
| manager_id profile/user | SPLIT | `organization_branches.manager_membership_id` |
| address | SPLIT/DEFER | `organization_branch_locations` only when it is genuinely branch location |
| role_id on branch member | REJECT/DEFER | scoped authority not proven |
| is_active | DEFER | reversible disable semantics not proven |
| is_deleted/deleted_at | RECONCILE | terminal `closed_at` only when closure is business truth |

Critical repair:

```text
legacy UserID
 -> Person
 -> current OrganizationMembership
 -> BranchAssignment / Branch manager
```

The legacy path allowed identity confusion and duplicate member rows. Backfill
must deduplicate pairwise assignment facts and flag users without a valid same-org
Membership.

---

# 4. Groups

Historical:

```text
group
group_member
leader/admin/member behavior
Deal context by Group
```

Mapping:

| Legacy | Class | Canonical |
| --- | --- | --- |
| group | COPY/RECONCILE | `groups` |
| group member | SPLIT | `group_participations` |
| group invite/pending membership | SPLIT | `group_invitations` |
| leader/owner scalar or special role | RECONCILE | `group_leaderships` |
| admin role | SPLIT | `group_admin_assignments` |
| member status | SPLIT | participation episode / invitation outcome / end |
| generic soft delete | RECONCILE | `groups.ended_at` only if business ended |

Group is not promoted to Party during this migration.

---

# 5. Product / Property

Historical BDSPro used `product` as a broad real-estate record with location,
price, owner and type data.

Canonical target says:

```text
Property = real-estate subject/factual identity
Listing  = market exposure/offer
Asset    = portfolio/economic treatment
```

## Product → Property

| Legacy | Class | Canonical |
| --- | --- | --- |
| product identity | RECONCILE | `properties` |
| property_type_id | COPY/normalize | `property_types` |
| province/district/ward/address/lat/lng | SPLIT | `property_locations` + `administrative_areas` |
| land info | SPLIT | `property_land_parcels` |
| house info | SPLIT | `property_house_details` |
| apartment info | SPLIT | `property_apartment_details` |
| amenities | COPY | `property_amenities` |
| product media | COPY | `property_media` + `media_objects` |
| product legal docs | SPLIT | `property_documents` + `documents` |
| mutable current price | RECONCILE | valuation and/or ListingPriceTerm depending semantics |
| owner_id + owner_type | REJECT | ownership belongs to Asset or another explicit relation |
| generic visibility | DEFER | policy/publication concern, not Property identity |
| product history | RECONCILE | audit/history evidence, not duplicate current fact |

### Price classification

Legacy `product.price` must be classified:

- asking/marketing price → ListingPriceTerm;
- appraised/estimated value → PropertyValuation;
- acquisition/current portfolio value → AssetValuation;
- transaction/contract amount → DealContract;
- unknown → preserve in migration staging until classified.

Do not force all old price values into one canonical column.

---

# 6. Post → Listing

Historical Post is market/publication behavior.

| Legacy | Class | Canonical |
| --- | --- | --- |
| post identity | RECONCILE | `listings` |
| product_id | SPLIT | `listing_properties` |
| title/content | COPY | `listings.title/description` |
| post_price | SPLIT | `listing_price_terms` |
| active/expired/hidden state | SPLIT | `listing_publications` episodes + terminal Listing lifecycle |
| expired_at | COPY | current publication `expires_at` |
| media | COPY | `listing_media` |
| owner_id + owner_type | REJECT | `publisher_party_id` |
| num_view/click counters | DERIVE/projection | analytics/read model, not Listing truth |
| VIP/package flags | DEFER | promotion/subscription capability |

Historical one-Post→one-Product is widened deliberately to explicit
many-to-many `listing_properties` because BDSPro business cases include
multi-lot/multi-property offerings.

---

# 7. Asset

Historical Asset mixed a Property reference, generic owner, value and operational
financial counters.

| Legacy | Class | Canonical |
| --- | --- | --- |
| asset row | COPY/RECONCILE | `assets` |
| product_id | RECONCILE | `assets.property_id` |
| owner_id + owner_type | SPLIT | `asset_ownerships.owner_party_id` |
| acquisition/current value | SPLIT | `asset_valuations` and/or Deal contract evidence |
| asset_cost | COPY | `asset_costs` |
| asset income | COPY | `asset_income_entries` |
| rental/exploitation | RECONCILE | `asset_leases` or later explicit exploitation fact |
| legal documents | COPY | `asset_documents` |
| cached total cost/income/ROI | DERIVE | projections from immutable entries |

Owner history is not a boolean `is_owner`; ownership is a temporal relation.

---

# 8. Projects / Developer

| Legacy | Class | Canonical |
| --- | --- | --- |
| project | COPY/RECONCILE | `real_estate_projects` |
| developer entity | RECONCILE | canonical Organization when it is an independent business organization |
| project build/tower | COPY | `project_buildings` |
| inventory/unit | RECONCILE | Property + `project_properties` |
| project docs | COPY | `project_documents` |

Do not keep a second Developer identity if the same legal organization is already
a canonical Organization.

---

# 9. Deal core

Historical Deal has the highest reconciliation risk.

## Deal business context

Historical evidence:

```text
deals.owner_id + owner_type
deal_of_organization
deal_of_group
deal_of_branch
deal_of_organization.branch_id
```

Mapping:

| Legacy | Class | Canonical |
| --- | --- | --- |
| owner_type=ORGANIZATION | RECONCILE | `deals.organization_id` |
| owner_type=GROUP | RECONCILE | `deals.group_id` |
| owner_type=USER/MEMBER | RECONCILE/DEFER | not a canonical Deal context unless real data proves a separate mode |
| deal_of_organization | RECONCILE then DROP_AFTER_PROOF | validation source only |
| deal_of_group | RECONCILE then DROP_AFTER_PROOF | validation source only |
| deal_of_branch | COPY/RECONCILE | `deals.branch_id` when one current Branch is proven |
| duplicate branch in deal_of_organization | RECONCILE | must agree with canonical Branch |

Canonical Deal must have exactly one of Organization or Group context.
Branch is optional and legal only under Organization context.

## DealMember blob

Historical `deal_members` includes:

```text
deal_id
member_id
role_id / role_key
amount_commit
commission_value/type
note
is_unilateral
done_investment
status INVITED/ACCEPTED/REJECTED/WITHDRAWN
message
invited_at/responded_at/withdrawn_at
inviter_id
is_owner
```

Canonical split:

| Legacy field/state | Class | Canonical |
| --- | --- | --- |
| INVITED proposal | SPLIT | `deal_invitations` |
| ACCEPTED relation | SPLIT | `deal_participations` |
| REJECTED | SPLIT | Invitation outcome DECLINED |
| WITHDRAWN after acceptance | SPLIT | Participation `ended_at` |
| member_id/profile ID | RECONCILE | `deal_participations.person_id` |
| role MEMBER | REJECT | Participation itself means member |
| role CUSTOMER | SPLIT | `deal_customers` |
| role PARTNER | SPLIT | `deal_partners` |
| role ADMIN | SPLIT | `deal_admin_assignments` |
| role OWNER | RECONCILE | `deal_leaderships` candidate |
| is_owner | REJECT | broken shadow truth |
| amount_commit | SPLIT | `deal_investment_commitments` |
| commission_value/type | SPLIT | `deal_commission_terms` |
| done_investment | REJECT as truth | derive from investments/commitment policy |
| invitation message/time | SPLIT | `deal_invitations` |
| inviter_id | audit provenance | `audit_events` if required |
| is_unilateral | RECONCILE | termination cause/event if business rules need it |
| color_id | REJECT from core | UI/presentation concern |

### Critical role collision repair

Historical creation code may add the same profile as Member + Customer + Partner +
Admin and then de-duplicate by MemberID, silently dropping later labels.

Therefore backfill cannot trust one final `role_key` as complete history.

Reconcile from:

- deal create requests/history;
- CRM/contract evidence;
- commission data;
- investment data;
- audit/history tables;
- current role key only as one signal.

## Deal leadership

Compare:

```text
role_key = OWNER
is_owner = true
creator
group leader / organization authority
```

Do not assume they agree.

The migration must produce an exception list for:

- zero owner candidates;
- multiple owner candidates;
- role owner but `is_owner=false`;
- `is_owner=true` but non-owner role.

A Deal leadership row is created only after the chosen business rule is proved.

## Deal commission

| Legacy | Class | Canonical |
| --- | --- | --- |
| member commission value/type | SPLIT | `deal_commission_terms` |
| final commission calculation | DERIVE/RECONCILE | `deal_commission_earnings` |
| commission paid state | RECONCILE | earning `settled_at` + Payment evidence |
| only_owner_get_commission | RECONCILE | policy/config input; not participant identity |

Do not overwrite earned commission when terms change later.

## Deal investment

| Legacy | Class | Canonical |
| --- | --- | --- |
| member amount_commit | SPLIT | `deal_investment_commitments` |
| investment row | COPY/RECONCILE | `deal_investments` |
| investment member ID | RECONCILE | Participation |
| invest_type Customer/Partner/Member | RECONCILE | relationship evidence, not identity |
| done_investment boolean | DERIVE | compare approved actual investments to current commitment |
| proof image/doc | COPY | `documents` reference |
| pending/approved/rejected | SPLIT into explicit terminal evidence | investment timestamps/state constraints |

## Deal contract / transaction

Historical customer IDs in contract/payment flows resolve to profile IDs.

Mapping:

```text
legacy customer profile
 -> Person
 -> DealParticipation
 -> DealCustomer
 -> DealContractCustomer
```

Contract product/property references map to `deal_contract_properties`.

Payment/money movement maps to `deal_contract_payments` and canonical `payments`.
Do not let transaction-service generic `from_type/to_type` become canonical
business ownership.

---

# 10. Auctions

If historical auction/order matching exists through generic transaction/deal
tables, classify by actual business action.

Canonical:

```text
auction
 -> registrations
 -> immutable bids
 -> one result referencing winning bid
```

Do not backfill a bid from a mutable "current price" unless immutable evidence
exists. Ambiguous auction history must remain migration evidence rather than
invented precision.

---

# 11. CRM

Historical CRM contains contacts/basic_profile/friend/pipeline/stage and other
duplicated profile-shaped data.

Mapping:

| Legacy | Class | Canonical |
| --- | --- | --- |
| CRM contact/basic profile | RECONCILE | `crm_contacts` |
| known canonical user/profile | LINK | `crm_contacts.linked_party_id` |
| labels | COPY | `crm_labels` + join |
| pipeline | COPY | `crm_pipelines` |
| stage | COPY | `crm_stages` |
| opportunity/lead workflow | RECONCILE | `crm_opportunities` |
| activity/history | COPY/normalize | `crm_activities` |
| converted opportunity/deal | RECONCILE | `crm_opportunity_deals` |

Never manufacture a Person merely because CRM has a phone/name. A CRM contact can
remain unlinked until identity is known.

---

# 12. Subscription / payment / wallet

Historical payment source contains commerce orders/settlements/payment attempts,
banks, wallets and provider eventing.

Canonical semantic mapping must classify business purpose:

| Legacy | Class | Canonical |
| --- | --- | --- |
| plan/package | RECONCILE | `plans`, `plan_prices`, `plan_entitlements` |
| active package/member subscription | RECONCILE | `subscriptions` |
| bill/order representing amount due | RECONCILE | `invoices` + `invoice_lines` |
| payment business operation | COPY/RECONCILE | `payments` |
| provider payment attempt | COPY | `payment_attempts` |
| provider settlement | COPY | `payment_settlements` |
| bank | reference/config | bank code; do not duplicate bank entity unless business needs it |
| linked bank account | COPY with encryption migration | `bank_accounts` |
| wallet | COPY/RECONCILE | `wallets` |
| wallet mutation/history | COPY as immutable facts | `wallet_entries` |
| provider inbox/outbox | operational | `inbox_events` / `outbox_events` |

Never derive wallet balance by trusting a mutable legacy cached balance without
reconciling ledger entries.

---

# 13. Usage / quota

Historical TQD quota tables used generic subject_type/profile/organization pairs.

Canonical target uses Party because both Person and Organization are already
valid business parties:

```text
usage_meters
usage_events(subject_party_id)
quota_allocations(subject_party_id)
```

Backfill rules:

- profile subject → corresponding Person Party;
- organization subject → Organization Party;
- preserve operation/idempotency key;
- usage events are immutable;
- quota allocation is not consumption;
- runtime reservation/cache data is not durable usage truth.

---

# 14. Community / social

Historical post/newsfeed/friend/tag/reaction data is classified into:

```text
communities
community_memberships
community_posts
community_comments
post/comment reactions
friendships
follows
```

Rules:

- author/member endpoints normalize to Person/Party as appropriate;
- a social Post is not a real-estate Listing;
- generic reaction target type/id is not retained as canonical truth;
- self-friend/self-follow rows are migration errors;
- friendship duplicate orientation (A,B) vs (B,A) must be canonicalized.

---

# 15. Chat

Historical:

```text
conversations
participants(user_id)
messages(sender_id)
read_receipts(user_id)
```

Canonical:

```text
conversation_participations(person_id)
messages(sender_participation_id)
message_reads(participation_id)
```

Migration must prove:

- sender was a participant in the same conversation at send time or classify an
  historical exception;
- read participant belongs to conversation;
- duplicate current participation is collapsed only when it is not a genuine
  leave/rejoin episode.

---

# 16. Appointments

Historical appointments/schedules normalize participants to Person and explicit
domain links:

```text
appointments
appointment_participants
appointment_deals
appointment_properties
```

Do not create a generic appointment_target(type,id) table.

---

# 17. Notifications

Historical notification `OwnerOf/type/id` patterns are delivery addressing,
not business ownership.

Canonical:

```text
notifications.recipient_person_id
notification_deliveries
notification_preferences
```

Historical event payloads may remain JSON integration evidence. They do not
become resource authority.

---

# 18. Planning / TQD / GIS

Historical `qh_*`, layer, label, report and planning-news data maps by business
fact:

```text
source             -> planning_sources
dataset/version    -> planning_datasets
layer              -> planning_layers
spatial feature    -> planning_features
label              -> planning_labels
feature-label      -> planning_feature_labels
source docs         -> planning_documents
generated report   -> planning_reports
report-property    -> planning_report_properties
planning news      -> planning_news
saved news         -> planning_news_saves
```

PostGIS geometry is preserved as geometry, not serialized into application JSON.

External feature attribute bags may remain JSONB because their schema legitimately
varies by dataset/layer.

---

# 19. Media / documents

Historical product/post/asset/deal/project/document tables often duplicate file
URLs and document metadata.

Migration:

```text
physical stored object
 -> media_objects

business document
 -> documents -> media_object

domain relationship
 -> explicit domain join
```

Examples:

- product image → property_media;
- post image → listing_media;
- asset legal file → asset_documents;
- deal contract/proof → deal_documents or investment proof;
- planning source file → planning_documents.

Do not introduce `attachments(owner_type, owner_id)`.

Deduplication by checksum is optional and must not merge distinct document
business facts merely because bytes happen to be identical.

---

# 20. Audit / history / eventing

Historical business-history tables, generic logs and event queues must be
classified before import:

- current canonical state → backfill actual domain tables;
- immutable provenance/security evidence → `audit_events`;
- integration publication state → `outbox_events`;
- consumer idempotency → `inbox_events`;
- request idempotency → `idempotency_records`;
- read-model/search analytics → rebuild rather than treating as canonical truth.

Generic `subject_kind + subject_id` is allowed only here because these rows are
evidence about another fact, never the source of that fact.

---

# 21. Global reconciliation reports required before cutover

Every migration gate should produce machine-checkable reports.

Minimum reports:

```text
identity:
  unmapped profiles
  duplicate Persons
  Account collisions

organization:
  owner disagreements
  memberships without Person
  duplicate current memberships
  role-assignment conflicts
  branch users without same-org Membership

property/listing/asset:
  products with ambiguous price semantics
  posts with missing property
  asset ownership conflicts
  invalid geography references

deal:
  unknown owner_type values
  context disagreement among owner fields/link tables
  invitation/participation contradictions
  duplicated/lost participant capacities
  zero/multi lead candidates
  commission term inconsistencies
  investment vs done_investment disagreement
  contract customer not participating

finance:
  payment terminal-state conflicts
  settlement without payment
  wallet ledger/balance mismatch

cross-domain:
  orphan FKs
  impossible temporal intervals
  currency mismatches
```

Every exception must end in one of:

```text
resolved deterministically
resolved by business/admin review
quarantined with explicit reason
preserved in compatibility storage pending decision
```

"Guess a value so the migration passes" is forbidden.

---

# 22. Cutover contract

A legacy table/column can be contracted only when all are true:

1. canonical backfill completed;
2. validation report has no unexplained exceptions;
3. dual/read comparison is green for an agreed period or proof set;
4. canonical write path owns new mutations;
5. canonical read path owns application behavior;
6. rollback/recovery path is known;
7. old source is no longer an authority;
8. the deletion is a separate migration/change.

This preserves the central BDSPro rule:

> **Migration changes authority, not merely storage shape.**
