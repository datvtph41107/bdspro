# BDSPro Canonical Database Final Package

This directory is the durable entry point for the completed BDSPro database
design pass.

## Status

```text
v0 semantic model       CLOSED
v1 logical model        CLOSED
v2 PostgreSQL strategy  CLOSED
final review baseline   ACCEPTED
implementation          NOT bulk-applied
```

The target diagram currently contains **146 tables**.

The word "final" means **canonical target baseline for review and incremental
implementation**. It does not mean all tables should be created in one migration
or that future business evidence may never change the model.

## Files

Read in this order:

1. [CANONICAL-DATABASE-ACCEPTANCE.md](./CANONICAL-DATABASE-ACCEPTANCE.md)  
   Why the model exists, v0→v1→v2 reasoning, invariants, rejected legacy
   abstractions, enforcement model and implementation gates.

2. [BDSPro-CANONICAL-DATABASE.dbml](./BDSPro-CANONICAL-DATABASE.dbml)  
   Full logical diagram source for dbdiagram.io.

3. [CANONICAL-DATA-DICTIONARY.md](./CANONICAL-DATA-DICTIONARY.md)  
   One-row assertion and semantic responsibility for every table.

4. [LEGACY-MIGRATION-MAP.md](./LEGACY-MIGRATION-MAP.md)  
   How historical BDSPro concepts are copied, split, merged, reconciled,
   deferred or rejected.

5. [CONTINUATION-PROMPT.md](./CONTINUATION-PROMPT.md)  
   One prompt to recover this exact database milestone in a fresh chat.

## How to view the diagram

Open dbdiagram.io, create a new diagram and paste/import the complete contents of:

```text
docs/database/final/BDSPro-CANONICAL-DATABASE.dbml
```

The DBML is the diagram source of truth. Do not manually edit a rendered diagram
and let it drift from the repository source.

The file uses TableGroups so the visual diagram can be navigated by domain:

- Identity & Security
- Organization & Authorization
- Groups
- Property & Projects
- Listings & Assets
- Deals & Auctions
- CRM
- Billing & Usage
- Community, Chat & Notifications
- Planning & GIS
- Media, Documents & Integration

## Relationship to live implementation

Current live implementation on `dev` remains the implementation authority.

At the baseline used for this design:

```text
dev@2a84561cb29dca7f8b198e61d4ad5a6980c2de5e
```

the application has a deliberately small Listing persistence slice using:

```text
PostgreSQL
migrations
explicit SQL
sqlc
pgx/v5 + pgxpool
```

This final package does **not** overwrite those migrations.

Implementation moves toward the target using:

```text
OLD
 -> EXPAND
 -> BACKFILL
 -> COMPARE / PROVE
 -> SWITCH WRITE AUTHORITY
 -> SWITCH READ AUTHORITY
 -> CONTRACT
```

Recommended gates are documented in the acceptance report.

## Canonical reasoning contract

Every future schema change must remain explainable through:

```text
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
```

Important negative rules:

```text
same name != same fact
same shape != same fact
foreign ID != ownership
API shape != durable relation
proposal != effective relationship
creator != authority
relationship != permission
read convenience != canonical write truth
```

Do not restore without new evidence:

- generic `owner_type + owner_id`;
- generic `type + id` resource relationships;
- universal `status`;
- universal soft delete / BaseEntity;
- `created_by/updated_by` everywhere;
- EAV for stable core facts;
- generic attachments as canonical business authority;
- Profile/User IDs where Person/Membership/Participation is the narrower correct
  endpoint;
- Invitation rows mutated into effective Membership/Participation;
- Property/Listings/Assets collapsed into one object;
- Deal context and Deal Lead collapsed into one "owner".

## Review procedure

A full human review should proceed in this order:

```text
1. Identity / Party / Account
2. Organization authority
3. Branch and Group
4. Property / Project
5. Listing / Asset
6. Deal / Commission / Investment / Contract
7. Auction / CRM
8. Billing / Payment / Usage
9. Community / Chat / Appointment / Notification
10. Planning / GIS
11. Media / Documents / Audit / Eventing
12. Cross-domain invariants and migration order
```

For each table ask:

```text
What does one row assert?
What is its identity?
Who owns the mutable truth?
What starts/ends the fact?
What must be impossible?
Can PostgreSQL enforce it?
Does another table already own this truth?
Is this business truth or projection/audit/integration evidence?
```

## Acceptance boundary

The package is accepted as a design baseline when review does not find:

- two canonical owners for the same mutable fact;
- generic IDs that prevent FK integrity for stable business relations;
- conflated proposal/effective/history facts;
- unqualified money;
- hidden business authority in UI/API fields;
- business identity encoded as authentication identity;
- permission implied solely by a relationship;
- migration steps that destroy unreconciled legacy evidence.

After review, implementation resumes one gate at a time from live `dev`.
