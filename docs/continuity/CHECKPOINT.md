# Current Checkpoint

> **Latest investment-gate correction (2026-10-09):** Before authorizing production implementation of ORG-01/ORG-NET-01 or broad 160-table V0.3 concepts, read `docs/database/final/BUSINESS-REALITY-GATE-2026-10-09.md`. Initial BDSPro has NO inherent organic audience; Facebook/Zalo/property portals remain agent acquisition channels. Verify user workflow pain, employee adoption, payers and measurable ROI in short pilot; keep ORM/GIS/Ranking/Group/Enterprise IAM as design candidates only until justified. Core DB lab continues for learning; NO schema/runtime change.


> **Latest remote design-branch milestone — 2026-10-09 V0.3:** `docs/database/final/README.md` now points to `CORE-REVIEW-V0.3.md`, `CORE-VECTOR-REVIEW-V0.3.md` and original 146-row impact mapping. GitHub canonical DBML has **160 candidate tables**, 6 composite relation contracts, and a matched 160-row dictionary. This is a docs/logical DBML change only; `dev` production schema/migrations unchanged. ORG-01/ORG-NET-01 and C01 SQL proofs still pending. Older "146-table" status below is historical. Always reconcile live Git and local work.


**Latest human-review override (2026-10-09; updated for evening enterprise-network discussion):** Read [Trust, Marketplace & Canonical Core recovery checkpoint](checkpoints/2026-10-09-trust-marketplace-core/README.md) **before** treating 2026-10-06 database design closures as current, fully proved business decisions. Deep review reopened Account cardinality, Membership episode semantics, Organization ownership/verification, provider activation, GIS provenance and related assumptions. The 146-table DBML is a review **candidate**, not authorization for production schema deployment. Core review resumes at **ORG-01 Organization Identity / Registration / Claim / Representative Authority + enterprise-first onboarding**, followed by **ORG-NET-01** (internal Team vs independent cross-company Group, Channel, Provider publication authority). Read `checkpoints/2026-10-09-trust-marketplace-core/ORGANIZATION-NETWORK-AND-PROVIDER-ONTOLOGY.md`. Internal Team/Department and Group/Community are distinct; do not make Group automatically an Organization child; C01 exactly-one Party subtype is a target invariant lacking complete PostgreSQL enforcement. DB-20 lab complete; DB-21 proposed, not verified. Keep any locally altered `dev` source unchanged. The older snapshot below is retained for historical traceability.

---

Updated: 2026-10-06

## Source identity

Active repository:

`datvtph41107/bdspro`

Active implementation branch:

`dev`

Stable/default branch:

`main`

Canonical database design branch:

`architecture/canonical-database-final`

Preferred local workspace:

`~/projects/bdspro`

Operational Go module identity:

`github.com/datvtph41107/bdspro`

Historical evidence repository:

`datvtph41107/bdspro-backend`

## Verified Git reality

The database-design branch was created without rewriting history from:

```text
dev:
  2a84561cb29dca7f8b198e61d4ad5a6980c2de5e
  migrate listing to sqlc

main:
  1a49d5bc469110c3b4b8d42a0d5f419bc45857bc

relationship at design start:
  dev ahead of main: 4 commits
  dev behind main:   0 commits
```

The dedicated design branch exists to avoid overwriting or assuming anything
about unpushed local `dev` work.

Live Git/source and completed proof always outrank this document. Never reset,
clean, rebase, amend or overwrite unexplained local work merely to match durable
prose.

## Current phase

```text
Canonical database design
v0 semantic model       CLOSED
v1 logical model        CLOSED
v2 PostgreSQL strategy  CLOSED
final review baseline   ACCEPTED
bulk implementation     NOT STARTED / NOT AUTHORIZED
```

The durable package is:

```text
docs/database/final/README.md
docs/database/final/CANONICAL-DATABASE-ACCEPTANCE.md
docs/database/final/BDSPro-CANONICAL-DATABASE.dbml
docs/database/final/CANONICAL-DATA-DICTIONARY.md
docs/database/final/LEGACY-MIGRATION-MAP.md
docs/database/final/CONTINUATION-PROMPT.md
```

The DBML currently contains:

```text
146 tables
223 parsed foreign references
0 references to undefined tables
```

That structural check proves internal table-reference completeness only. It does
not prove that dbdiagram/PostgreSQL accepts every future physical constraint or
that all cross-row business invariants are expressible in DBML.

## Relationship to live implementation

The design package is a **target architecture artifact**, not an executable
migration.

Current live implementation on `dev` remains deliberately small:

```text
PostgreSQL
 -> versioned migrations 000001..000005
 -> explicit SQL
 -> sqlc
 -> pgx/v5 / pgxpool
 -> Listing create/edit/publish/readiness
```

Do not replace those migrations or current Listing code wholesale with the
146-table target.

Implementation must proceed through:

```text
OLD
 -> EXPAND
 -> BACKFILL
 -> COMPARE / PROVE
 -> SWITCH WRITE AUTHORITY
 -> SWITCH READ AUTHORITY
 -> CONTRACT
```

## Canonical design closures

### Identity

```text
Party
├── Person
└── Organization

Account -> Person
```

Person/Organization are business identity. Account/authenticators/sessions are
digital-security facts.

### Organization

Accepted separation:

```text
Invitation
Membership
MembershipBlock
RoleAssignment
Ownership
BranchAssignment
BranchManager capacity
```

These are not one status/role/member blob.

Organization Ownership is a singular current relation to OrganizationMembership.
Branch assignments/managers target OrganizationMembership, not User/Profile.

### Property / Listing / Asset

```text
Property = real-estate subject/factual identity
Listing  = market exposure/offer about Property
Asset    = portfolio/economic treatment of Property
```

Listing may reference multiple Properties.

Price facts are qualified by domain and lifecycle; one generic mutable `price`
is not the final model.

### Deal

Accepted semantic skeleton:

```text
Deal
├── exactly one business context
│   ├── Organization
│   └── Group
├── optional Organization Branch scope
├── Properties
├── Invitations
├── Participations -> Person
├── Customer / Partner relationships
├── Admin authority
├── Lead governance
├── Commission terms
├── Commission earnings
├── Investment commitments
├── actual Investments
├── Contracts
└── Documents
```

Rejected:

```text
owner_type + owner_id
is_owner shadow truth
single role_key mixing OWNER/ADMIN/MEMBER/CUSTOMER/PARTNER
Invitation row mutated into accepted/withdrawn participation
done_investment as canonical participant truth
```

### Cross-domain

Money is amount + currency.

Media/document storage may be shared, but business links remain explicit.

Generic `subject_kind + subject_id` is allowed only in operational evidence
(audit/outbox/inbox), never as canonical ownership/participation truth.

Planning/GIS retains PostGIS geometry.

## Current authority for database work

Read in this order:

1. live Git/source;
2. `docs/continuity/CONTINUATION-PROMPT.md`;
3. `docs/database/final/README.md`;
4. `docs/database/final/CANONICAL-DATABASE-ACCEPTANCE.md`;
5. `docs/database/final/BDSPro-CANONICAL-DATABASE.dbml`;
6. `docs/database/final/CANONICAL-DATA-DICTIONARY.md`;
7. `docs/database/final/LEGACY-MIGRATION-MAP.md`;
8. this checkpoint and the decision register.

Historical `bdspro-backend` remains evidence only.

## Current unresolved pressure

The target database is now broad enough that the next responsible action is **not
to add more tables by default**.

The next pressure is:

```text
full human review of the canonical target
 -> identify any fact/invariant/name that still fails under real business cases
 -> amend the target only with explicit counterexample/evidence
 -> accept review
 -> select the first implementation gate from live dev
```

The first implementation gate after review is Identity backbone unless review
finds a reason to change the order:

```text
parties
 -> persons
 -> organizations
 -> accounts/auth
```

Current Listing code/migrations are preserved and must be reconciled with the
Property/Listing target through incremental migrations rather than a reset.

## Working reasoning contract

For every future database change use:

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

For every table first state:

> One row means ...

If the answer requires an unrelated `type`, generic owner pair, overloaded
status, or multiple incompatible identities, return to semantic modeling.

## Next authorized gate

1. Review the final database package domain by domain.
2. Record corrections, if any, against concrete counterexamples.
3. When review is accepted, reconcile live local `dev` Git state.
4. Select one implementation gate.
5. Implement through migrations + SQL/sqlc + transaction + proof.
6. Never create all target tables at once merely because the final DBML exists.
