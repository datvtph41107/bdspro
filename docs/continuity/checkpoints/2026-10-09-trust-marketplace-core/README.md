# BDSPro — Trust, Marketplace & Canonical Core: recovery checkpoint

**Checkpoint date:** 2026-10-09 (Asia/Bangkok)  
**Repository:** `datvtph41107/bdspro`  
**Design branch:** `architecture/canonical-database-final`  
**Implementation branch:** `dev` (live Git state must be rechecked)  
**Status:** BUSINESS CONTEXT CAPTURED / CORE UNDER HUMAN REVIEW / NOT PRODUCTION IMPLEMENTED  
**Canonical target:** `docs/database/final/BDSPro-CANONICAL-DATABASE.dbml` (146 tables, candidate under renewed review).  
**Old service source:** `datvtph41107/bdspro-backend` (historical evidence only).

This checkpoint records durable conclusions of the 2026-10-06 through 2026-10-09 deep database and business review. It **updates the review posture**, not the physical schema; former "accepted" target artifacts remain historical design baselines, NOT uncontested current decisions.

## Start here: recovery order

1. Inspect live branch, HEAD and unknown local changes; do not overwrite/reset local `dev`.
2. Read `docs/continuity/MINDSET.md`, `PRACTICE-PROTOCOL.md`, `RESPONSE-PROTOCOL.md`; retain deliberate, user-operated PostgreSQL learning.
3. Read **this README**, then `BUSINESS-OPERATING-MODEL.md`, `DECISIONS-AND-PRESSURES.md`, `DATABASE-REVIEW-AND-LAB.md`, and the **latest** `ORGANIZATION-NETWORK-AND-PROVIDER-ONTOLOGY.md`.
4. Open canonical DBML + data dictionary + acceptance documentation as **existing candidate assertions**. Test assertions against this checkpoint; don't mistake prior "CLOSED" for all-business-proof.
5. State what is proven, provisional, open or rejected. Continue **ORG-01 Identity/Verification/Claim**, while preserving C01 Party exactly-one-subtype and unfinished physical enforcement. No wholesale implementation.

### Navigation

- [Business narrative, actors, operating model, references](BUSINESS-OPERATING-MODEL.md)
- [Mindset, invariants, tradeoffs, decision register](DECISIONS-AND-PRESSURES.md)
- [146-table review map, PostgreSQL lab, immediate next steps](DATABASE-REVIEW-AND-LAB.md)
- [Enterprise-first onboarding; organization/branch/team/group/community/channel; seller/broker authority](ORGANIZATION-NETWORK-AND-PROVIDER-ONTOLOGY.md)
- [Canonical database target](../../../database/final/BDSPro-CANONICAL-DATABASE.dbml)
- [Earlier acceptance report — historical design baseline](../../../database/final/CANONICAL-DATABASE-ACCEPTANCE.md)

## Product mission and north star

BDSPro aspires to be a Vietnamese nationwide **trustworthy real-estate ecosystem** connecting **people, organizations, communities, property supply and demand, collaborative work, financial outcomes and spatial/government information**. This is an ambition, not a claim of government-system access, regulatory authorization or achieved compliance.

Three mutually dependent pillars:

1. **TRUST**: identities, proof, authority, safety, privacy, legal compliance, fair grievance handling, auditable transactions.
2. **DATA**: properties, cadastral parcels, planning/land-use records, source provenance, quality, temporal versions, geospatial correctness and legal limits on reuse.
3. **COMMUNITY & MARKET**: buyers, owners, licensed/authorized professionals, organizations, groups, shared deals, CRM, marketing, discovery, reputation and incentives.

Product intelligence (ranking, recommendation, map search, directory, discovery, reports) must derive from protected business truth; it cannot silently create/override legal or permission facts.

## Mental model

```text
PERSON / ORGANIZATION (durable business identity; Party backbone)
        -> ACCOUNT (how a human authenticates; not a business owner)
        -> CONTEXT (personal, organization, group, community, deal)
        -> AUTHORITY + MEMBERSHIP + ROLE + SECURITY POLICY
        -> BUSINESS ACTION (property/listing/CRM/deal/marketing/etc.)
        -> DURABLE FACTS + EVIDENCE + AUDIT AS APPROPRIATE
        -> READ/SEARCH/MAP/REPUTATION PROJECTIONS
        -> BENEFITS, DISCOVERY, CUSTOMER OUTCOMES
```

A subscription grants a **product capability**; a verification attests **specific evidence**; a reputation level represents **assessed outcomes**; a role grants **scoped action rights**; neither one substitutes for another.

## This checkpoint is not a mandate to create dozens of new tables

- User wants deep business reality, real counterexamples, operational/performance costs and independent evidence **before implementation**, except narrow lab practice.
- Preserve current live listing code/migrations; no all-at-once 146-table migration.
- Legacy services including `tqd-service` are case studies, not deployment mandates.
- Prefer a robust write core; projections/read models can be rebuilt from authorized authoritative data.
- The user writes SQL/code personally, predicts, runs and sends focused results; do not dump complete paste-ready solutions by default.

## One-prompt continuation (copy into new conversation)

```text
Continue BDSPro's deep canonical database/core design review from the 2026-10-09 durable Trust/Marketplace checkpoint, NOT a fresh redesign and NOT the older 2026-10-06 acceptance as unquestionable truth.

GitHub repo: datvtph41107/bdspro
Design branch: architecture/canonical-database-final
First read exactly:
docs/continuity/checkpoints/2026-10-09-trust-marketplace-core/README.md
docs/continuity/checkpoints/2026-10-09-trust-marketplace-core/BUSINESS-OPERATING-MODEL.md
docs/continuity/checkpoints/2026-10-09-trust-marketplace-core/DECISIONS-AND-PRESSURES.md
docs/continuity/checkpoints/2026-10-09-trust-marketplace-core/DATABASE-REVIEW-AND-LAB.md
docs/continuity/checkpoints/2026-10-09-trust-marketplace-core/ORGANIZATION-NETWORK-AND-PROVIDER-ONTOLOGY.md
docs/continuity/MINDSET.md
docs/continuity/PRACTICE-PROTOCOL.md
docs/database/final/BDSPro-CANONICAL-DATABASE.dbml

Distinguish proven PostgreSQL lab evidence from business assumptions and from production authorization. Canonical DBML has 146 candidate tables; no bulk schema implementation is approved. Focus on Party/Person/Organization, Account and organization identity; revisit and challenge legacy choices including multi-account Person, membership episodes and organization ownership. Reconcile local Git/proof before mutations. Continue ORG-01: enterprise registration vs claim of an existing legal entity, enterprise-first onboarding of a person with NO prior BDSPro account, verify existence/authority/qualification separately. Then test ORG-NET-01: internal team access vs cross-company Group and provider authority before choosing database entities, one step at a time. Keep core trust, verified seller/provider, privacy, CRM, deal commission, planning provenance, recognition and ranking in view as long-term business pressures. User writes/tests SQL; preserve previous lab.
```

## Non-negotiable recovery distinctions

- **Proof:** C01 lab results up through DB-20; concrete DBML facts; previous repositories and official mechanisms where inspected.
- **Provisional:** whether same Person can have multiple Accounts; membership as one relationship vs episodes; verification storage model; managed enterprise identity; seller-level policy; GIS pipeline topology.
- **Hypothesis:** ranking weights 60/40, lookback 180 days, levels 80/92 and associated benefits; no historical BDSPro metrics validated them.
- **Not authorized:** copying production schema into 146 migrations, deploying managed accounts, asserting VNeID access, public display of private KYC/CRM, paying commissions from scores, inheriting `tqd-service` wholesale.

## Snapshot of active lab

PostgreSQL 18.6; DB `bdspro_test`, schema `c01_lab`. Party 1 = Person, Party 6 = Organization. Accounts 2 and 3 both point to Person 1 (experiment, not design endorsement). Membership IDs 1, 2 ended; ID 5 current for (Organization 6, Person 1). Partial UNIQUE `organization_memberships_one_open_uq` on `(organization_id, person_id) WHERE ended_at IS NULL`. DB-20 result membership 2 -> 5: same time boundary = true, overlapping `[)` ranges = false. Do not confuse SQL constraint proof with business-need proof.

**Next review:** ORG-01: (A) new legal-entity registration, (B) claim existing organization by competing requesters, (C) transfer/offboard prior administrator, plus (D) organization's stronger authentication policy. Earlier DB-21 draft `organization_claim_requests.organization_id NOT NULL` tests **existing-organization claim only**, not new registration. Correct semantic design before writing canonical DDL. Use C02 "declaration != authority".

## 2026-10-09 evening refinement

Enterprise-first onboarding is a real first-class B2B entry path even when an employee never started as a consumer. Internal Team/Department, operational Branch, collaborative Group, interest Community and communication Channel are **different business meanings**; Group is not intrinsically a child of an Organization, and Class is not an entity without an independent learning/cohort lifecycle. Provider/seller/broker/publisher represent distinct capacities and scoped proofs, not Person subtypes. See [Organization/Network/Provider ontology](ORGANIZATION-NETWORK-AND-PROVIDER-ONTOLOGY.md). This addition does not authorize extra DBML tables.
