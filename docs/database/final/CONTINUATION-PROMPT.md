# Continuation Prompt — Canonical Database Finalization

> **2026-10-10 V0.4 OVERRIDE:** Latest logical DBML has **164 candidates**, not earlier 160/146. First read [2026-10-10 Business-to-Core recovery](../../continuity/checkpoints/2026-10-10-business-to-core/README.md), [Business Atlas V1](../../business/BUSINESS-ATLAS-V1-2026-10-10.md) and [V0.4 DB decisions](../../continuity/checkpoints/2026-10-10-business-to-core/DATABASE-CHANGE-DECISIONS.md). The new four tables are `source_intakes`, `source_price_reports`, `crm_inquiries`, `crm_followup_tasks`, all **review candidates**. ENG-00→CORE-01 is the next engineering gate; BUSINESS-00 field adoption and WTP still open. Older ORG-01 and 146-table handoff notes below are historical and must not override active state.

---


> **2026-10-09 V0.3 active handoff:** Read [CORE-REVIEW-V0.3.md](CORE-REVIEW-V0.3.md), [CORE-VECTOR-REVIEW-V0.3.md](CORE-VECTOR-REVIEW-V0.3.md) and [CORE-IMPACT-MATRIX-V0.2.md](CORE-IMPACT-MATRIX-V0.2.md) BEFORE the older prompt below. Actual design DBML is now **160 candidate tables** (146 original mapped, four renames and 14 additions). Do not mistake acceptance for implemented source. Continue ORG-01/ORG-NET-01, not bulk migration.


> **2026-10-09 active review notice:** This older finalization handoff is preserved as historical design context. Before accepting its `v0 → v1 → v2` closed assertions, first read `docs/continuity/checkpoints/2026-10-09-trust-marketplace-core/README.md` (and its three linked documents). Business review REOPENED cardinality, Organization authority, Membership semantics, provider trust and data-provenance choices. Continue ORG-01 rather than implementing the 146-table schema. The prompt below records the earlier milestone, not the latest work gate.

Paste the following prompt into a fresh ChatGPT conversation:

```text
Continue BDSPro from the canonical database finalization milestone, not from conversational memory.

Repository: datvtph41107/bdspro
Stable branch: main
Active implementation branch: dev
Database design branch: architecture/canonical-database-final

First reconcile live Git/source. Then read, in this order:

1. docs/continuity/CONTINUATION-PROMPT.md
2. docs/database/final/README.md
3. docs/database/final/CANONICAL-DATABASE-ACCEPTANCE.md
4. docs/database/final/BDSPro-CANONICAL-DATABASE.dbml
5. docs/database/final/CANONICAL-DATA-DICTIONARY.md
6. docs/database/final/LEGACY-MIGRATION-MAP.md
7. docs/continuity/CHECKPOINT.md
8. docs/continuity/DECISIONS.md
9. docs/continuity/HISTORY.md

Treat immutable live Git/source and completed proof as higher authority than prose.

The canonical DB package is the accepted target review baseline from the completed
v0 -> v1 -> v2 -> final database-design pass. It is NOT permission to create all
146 tables at once and it does NOT replace current live migrations/source on dev.

Preserve the migration rule:

OLD
 -> EXPAND
 -> BACKFILL
 -> COMPARE / PROVE
 -> SWITCH WRITE AUTHORITY
 -> SWITCH READ AUTHORITY
 -> CONTRACT

Preserve the reasoning chain:

BUSINESS INTENT
 -> ACTOR / CAPACITY / CONTEXT
 -> PRECONDITION / AUTHORITY
 -> BUSINESS TRANSITION
 -> DURABLE FACTS
 -> INVARIANTS
 -> TRANSACTION BOUNDARY
 -> EFFECTS / EVENTS / PROJECTIONS
 -> RELATIONAL MODEL
 -> PHYSICAL ENFORCEMENT

Do not restore generic owner_id+owner_type, generic type+id relationships, generic
status, universal soft delete/BaseEntity, EAV for stable core facts, or generic
attachments as canonical business authority.

Do not conflate:
- Account with Person;
- Invitation with Membership/Participation;
- Role with Ownership;
- Branch Assignment with Permission;
- Property with Listing or Asset;
- Deal Organization/Group context with Deal Lead;
- Deal Participation with Customer/Partner/Admin/Lead;
- commission terms with earned commission;
- investment commitment with actual investment;
- Payment with provider attempt/settlement;
- audit/event rows with canonical business state.

Start by performing one full review pass of the canonical database package,
domain by domain. Record any proposed correction as:
CURRENT FACT -> COUNTEREXAMPLE/PRESSURE -> BROKEN INVARIANT -> SMALLEST CHANGE.

Only after review is accepted, choose the first implementation gate from the
actual live dev source. Do not rewrite the database wholesale. Preserve unknown
local work and never reset, clean, rebase or amend as a recovery shortcut.
```

## Expected recovery result

A successful recovery should be able to state:

- the live branch/SHA relationship;
- that the final DBML contains 146 tables;
- why the target DBML is separate from live migrations;
- the accepted identity/organization/property/listing/deal boundaries;
- the first implementation gate after review;
- any contradiction found between live source and this package.
