# Current Checkpoint

> **CURRENT ACTIVE CHECKPOINT — 2026-10-10, EVIDENCE-DRIVEN AUTH IMPLEMENTATION.**
> [Read the new precise checkpoint](checkpoints/2026-10-10-evidence-driven-auth-handoff/README.md) and the mandatory [concrete reasoning method](EVIDENCE-DRIVEN-REASONING.md) **before following any older next-task statement below**. [Copyable next-chat prompt](checkpoints/2026-10-10-evidence-driven-auth-handoff/CONTINUE-PROMPT.md). The user explicitly confirmed: every material claim must be made visible via an actor, sample before/after rows, operation, mechanism, fair alternative, falsifying case and verifiable evidence; user personally implements code/SQL locally; advance proactively without repeated permission questions.
>
> **Verified GitHub dev:** commit [545d405e](https://github.com/datvtph41107/bdspro/commit/545d405e00052971f21c3953d101ac63c808933d) removed the ten legacy Listing migrations; related Listing sqlc query and sqlc.yaml still exist. Subsequent commits added documentation/method only. **Do not treat older assertions that remote still retains the ten migration files or that local is at 8f77dfc as present facts.** User's exact current LOCAL ref/working tree, PostgreSQL schema, new Auth migrations and generated-code compatibility have **not** been freshly observed in this chat. The canonical 164 tables are logical candidates, not tested code. **Next:** user-led, evidence-driven, narrow AUTH Core slice; inspect current local/DB state only where necessary to safely act, then do real migration/SQL/Go work and test negative cases. No bulk 164-table implementation.

> **NEW-CHAT HANDOFF — Git-00 OBSERVED LOCAL OUTPUT, 2026-10-10:** User ran `git branch --show-current`, `git status --short --branch`, `git branch -vv`, `git fetch origin --prune`, `git rev-list --left-right --count dev...origin/dev`, `git diff --name-status dev...origin/dev`, `git branch -r` on `~/projects/bdspro`. Output: local `dev@a69fd2a`, **10 tracked Listing migrations deleted in working tree and NOT COMMITTED**, `rev-list = 0 9` at the exact moment of the observation (local had zero unique commits, remote nine ahead). The committed upstream diff is docs-only (README + continuity docs + `REPOSITORY-WORKING-CONTRACT.md`). `origin/HEAD -> origin/dev`, `origin/dev` and `origin/main` exist. Local branch `main@1ce48ed` displayed `[origin/main: gone]` in `git branch -vv`, even though `origin/main` had subsequently been recreated; **do not confuse local branch/upstream configuration with remote branch existence**. Separately verified GitHub just before this checkpoint: `dev@63dfb4f`, restored `main@3f90f77`; default branch `dev` at last check. This checkpoint commit itself will put the remote **one further commit ahead**; always re-run Git comparisons rather than reuse `0 9` as current. **NEXT:** user wishes to continue in a NEW CHAT. Read [REPOSITORY-WORKING-CONTRACT.md](REPOSITORY-WORKING-CONTRACT.md) and [PRACTICE-PROTOCOL.md](PRACTICE-PROTOCOL.md), then guide user to **preserve the 10 intentional deletions** (scoped stash or intentional local commit), fast-forward local `dev` to `origin/dev`, restore intended deletions, inspect `git diff` and Git status. Have user type the commands and provide actual output; do not perform Git on user's computer, do not reset/hard clean, do not restore Listing migration files by accident. After the Git sync gate, continue **SQL-01** user-written PK/FK/XOR C01 proof in disposable PostgreSQL. No assistant-authored code/CI or PRs.


> **ACTIVE 2026-10-10 — GIT RESTORATION + REPO WORKING STANDARD:** `main` restored as branch from exact historical commit `3f90f7789`; `dev` remains sole code-writing/integration branch (default at the time of restore). `main` is a **historical baseline**, not a demonstrated stable build/release. Canonical working rules: [REPOSITORY-WORKING-CONTRACT.md](REPOSITORY-WORKING-CONTRACT.md), command practice: [ENGINEERING-WORKFLOW.md](ENGINEERING-WORKFLOW.md), user's code ownership: [PRACTICE-PROTOCOL.md](PRACTICE-PROTOCOL.md). User's local state previously showed 10 deleted Listing migrations and local behind remote; remote tool cannot confirm if already synced. **Next user-run task: Git-00** inspect current local working tree, fetch/rev-list, predict/observe, preserve deletions before fast-forward. **Then SQL-01** user writes PK/FK queries in disposable PostgreSQL, tests bare and dual Party subtype, reads errors, chooses C01 guarantee. No assistant-generated implementation or CI. Historic earlier blocks below (including main deleted, PR#2/#3 green) are evidence only, not current next action.


> **ACTIVE WORKING GATE — 2026-10-10 LOCAL ↔ GITHUB CONTRACT:** Verified GitHub remote has only **default `dev`**; `main` and previous branches were deleted. See [Repository Core Contract](ENGINEERING-WORKFLOW.md) for the single operational standard. User writes and tests SQL/Go/CI themselves, with assistant as reviewer/mentor. Last locally reported working tree had **ten deleted Listing migrations**, and local was behind remote; this **has not yet been checked after subsequent remote documentation updates**. Therefore **next action is user's safe local synchronization and precise Git/DB inspection**, not a new PR or automatic C01 implementation. C01 exercises on unmerged prior PRs are historical proof, NOT the newly reset local database proof. The remote still contains historical Listing migrations/Go runtime until user intentionally commits compatible edits. Future checkpoint entries must identify a real HEAD, observed output, what's proven, what's not, and next task; update on material milestones only. Other notes below are historical.


> **LATEST 2026-10-10 VERIFIED PROOF:** Draft [PR #2](https://github.com/datvtph41107/bdspro/pull/2) source `implementation/core01-party-c01-20261010` has two green PostgreSQL 18 CI runs. Most recent verified [run 38015055276](https://github.com/datvtph41107/bdspro/actions/runs/38015055276): old Listing migrations 000001..000005 + new CORE-01 000006 applied; Go tests and 6 C01 transaction/integration tests green; **000006 down → assert tables removed → 000006 up → tests rerun** green. Real migration/Go tests PROVEN in GitHub CI, not user's local database; code NOT YET MERGED to `dev`. For next "Tiếp tục": review PR and live Git/local status before merge, then CORE-02 Account authentication contract; retain BUSINESS-00 product gate. sqlc regenerate drift is an OPEN check.


> **2026-10-10 LIVE CORE-01 UPDATE — CI VERIFIED:** [Draft PR #2](https://github.com/datvtph41107/bdspro/pull/2), implementation branch `implementation/core01-party-c01-20261010` based on `dev@80f5223`. GitHub Actions [run 38014866038](https://github.com/datvtph41107/bdspro/actions/runs/38014866038) completed SUCCESS with PostgreSQL 18; migrations `000001..000006` applied, `go test ./...` passed, all six C01 PostgreSQL integration tests and validation passed. **Not merged to dev; local/production DB not changed, rollback SQL and sqlc regeneration not yet executed.** Next “Tiếp tục”: inspect PR #2 latest run and review/merge decision, then plan CORE-02 account/auth; do not create duplicate C01 migration. BUSINESS-00 product gate remains open.


> **CURRENT 2026-10-10 CHECKPOINT — BUSINESS→CORE IMPLEMENTATION HANDOFF:** The new [checkpoint README](checkpoints/2026-10-10-business-to-core/README.md) supersedes older proposed next steps; read [Business Atlas V1](../business/BUSINESS-ATLAS-V1-2026-10-10.md), [VS01 contracts](checkpoints/2026-10-10-business-to-core/VS01-CONTRACTS-AND-DB-GAPS.md), [DB V0.4 decisions](checkpoints/2026-10-10-business-to-core/DATABASE-CHANGE-DECISIONS.md) and [implementation gates](checkpoints/2026-10-10-business-to-core/BUSINESS-TO-CORE-IMPLEMENTATION.md). DBML now contains **164 LOGICAL CANDIDATE tables**, not 164 runnable migrations; previous 160/146 counts below are historical. No `dev` source modified, no PostgreSQL proof claimed. **Immediate next work: ENG-00 verify live dev/local Git/Go/Postgres, then CORE-01 C01 Party subtype concurrency lab; proceed to CORE-02 Account and CORE-03 Org context.** BUSINESS-00 PMF gate remains open for product deployment. A fresh "Tiếp tục" means recover this exact checkpoint then proceed, not re-review the whole company abstractly.

---


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
