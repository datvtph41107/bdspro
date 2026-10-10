# BDSPro — UNIFIED STATE / WORKING MINDSET
**Recorded:** 2026-10-10 · **Canonical file path on both remote `main` and `dev`** · **Working code branch: `dev` (including user's local `dev`)**.
This file is a **single navigational snapshot and decision log** spanning the known remote branches, **not a literal Git merge of incompatible histories**, not a byte-for-byte transcript, and not proof of the local working tree. A copy of the identical content is kept in `main` and `dev`; local must be synchronized by the user.

## 1. Firm collaboration contract
- **User writes and runs their own SQL, Go and (later) CI.** Assistant poses one scenario/invariant, asks for a prediction, reviews user-authored SQL, examines actual output/SQLSTATE/transaction behavior, explains what did/did not get proved, and offers the next small pressure. Do not prewrite full implementation/test/CI or create alternate code branches on the user's behalf.
- Reasoning loop: actual actor & business job → one-row meaning → authority/context → invariant → user prediction → user writes SQL → run in disposable PostgreSQL → inspect precise output/errors → negative case, concurrency, operational risk → decide KEEP/MODIFY/DEFER → only then write migration/Go → review/commit on local `dev`.
- Technical mechanism proof ≠ business-model correctness. E.g. partial UNIQUE can enforce only one active membership while the choice of episodic memberships remains an open domain decision. FK cannot itself grant authority; tenant scope is not resource permission. No batch 164-table migration.
- User said their **local DB and migrations directory were reset**, but remote GitHub cannot inspect uncommitted local changes. Use explicit `git status` and PostgreSQL results, never assume local/remote identical.
- **Assistant must not author/merge new Go/migrations/CI without an explicit new request.** Documentation and branch-state auditing are permitted by the user's current consolidation request.

## 2. Historical lab evidence (user-run; recorded 2026-10-09)
Historical test environment: PostgreSQL **18.6**, `bdspro_test`, schema `c01_lab` (historical snapshot; user later reset DB). Source: `docs/continuity/checkpoints/2026-10-09-trust-marketplace-core/DATABASE-REVIEW-AND-LAB.md`.
- DB-01..14: Party/Person/Organization PK/FK, Account→Person, membership and time intervals. Two Accounts (IDs 2/3) could point to Person 1 in absence of UNIQUE: **SQL possibility**, not approval of that business cardinality. PK/FK did not enforce total/exclusive Party subtype C01.
- DB-15: inserted third open Membership with `(organization_id=6, person_id=1, ended_at=NULL)`, ID 3; grouped count of open memberships became 2 for (6,1).
- DB-16: partial UNIQUE creation failed because duplicates existed.
- DB-17: cleaned duplicates and created `organization_memberships_one_open_uq` on `(organization_id,person_id) WHERE ended_at IS NULL`.
- DB-18: second open Membership rejected by PostgreSQL.
- DB-19: ended Membership ID 2 and created ID 5 within one transaction using a common `NOW()` boundary.
- DB-20: observed `old_membership_id=2; new_membership_id=5; same_boundary=true; is_overlapping=false` with half-open interval semantics `[joined_at,ended_at)`.
- DB-21 / C02 (organization claim/request ≠ existing authority): **proposal only, NOT executed**.
Never represent this as current local database state after the user reset.

## 3. Remote branch audit (snapshot just before the latest same-file synchronization)
Total **9** GitHub branches:
1. `main` — stable; before this file `1a49d5bc469110c3b4b8d42a0d5f419bc45857bc`.
2. `dev` — authoritative next source branch; after docs-only PR #1 merge `55469205c2d09cb3a0b6405a246001a556de3be5`.
3. `architecture/canonical-database-final` — docs/164 logical candidate tables, head `d8537df6a221526fd8ea1ece1cd4f57a01c32d46`; **its documentation is now merged into dev** through [PR #1](https://github.com/datvtph41107/bdspro/pull/1), commit `55469205`. Avoid merging it repeatedly.
4. `checkpoint/core-human-review-2026-10-08`
5. `checkpoint/core-review-2026-10-08`
6. `review/checkpoint-2026-10-08-canonical-human-pass`
7. `review/checkpoint-2026-10-08-core-29-tables`
   - Branches 4–7 are redundant historical pointers to identical SHA `7d1a0dcc74c3fd22bdd9eb603516d954cb531061`; history retained, no new independent code to merge.
8. `implementation/core01-party-c01-20261010` — [Draft PR #2](https://github.com/datvtph41107/bdspro/pull/2): older migration `000006` against Listing lab history, author-generated Go and CI. PostgreSQL CI green for those historical tests; **do not merge into new hand-coded dev baseline**.
9. `implementation/core-baseline-20261010` — [Draft PR #3](https://github.com/datvtph41107/bdspro/pull/3): alternative clean migration `000001` deleting old Listing migrations on its feature branch, author-generated Go and CI. PostgreSQL CI green for its tests; **do not merge wholesale into user's dev workflow**.
Do not delete branches/rewrite Git history merely because their content was inventoried; archiving/deleting/closing is a separate explicit clean-up decision.

**Main/dev are NOT identical source trees.** Even after this file has identical content on both branches, source synchronization requires reviewing the Listing lab code, remote migration deletions and uncommitted local dev changes. No safe blind merge of all branches into one codebase exists while PR#2 and PR#3 implement contradictory migration sequences.

## 4. Domain / schema canonical understanding
- `docs/business/BUSINESS-ATLAS-V1-2026-10-10.md` now exists in remote `dev` via PR #1.
- `docs/database/final/BDSPro-CANONICAL-DATABASE.dbml` V0.4 = **164 logical candidate tables**, not applied SQL. Data dictionary/acceptance notes adjacent. Four V0.4 candidates: `source_intakes`, `source_price_reports`, `crm_inquiries`, `crm_followup_tasks`.
- Business truth boundaries: Party ≠ Account ≠ Membership ≠ Role/authority; one Party exactly one Person XOR Organization target C01; Source Intake ≠ Property ≠ Listing ≠ Asset; Inquiry ≠ Contact ≠ Opportunity ≠ Activity ≠ Task; paid entitlement ≠ access/representation; Group ≠ Organization; claims and Zalo/Facebook outside observations ≠ verified attribution.
- BUSINESS-00 real adoption/willingness-to-pay gate remains open, regardless of green migration CI. Business Atlas is comprehensive **analytic coverage**, not market proof.
- The old remote `dev` still contains five Listing-lab migrations `000001..000005` and Listing/sqlc runtime. Local user intentionally cleared migrations/database; remote changes are **not** present locally until user reconciles them. Do not run `main.go` under a Party-only schema and call it a complete migration success; Listing query/runtime compatibility is an explicit gate.

## 5. Single code path going forward
**Only `dev` is active for hands-on implementation.** `main` remains stable; move code from dev to main after real, user-executed proof and review, not automatically to make heads equal.
Candidate next user-authored change: `000001_create_party_identity.up/down.sql` in cleaned local dev. The user writes minimal PK/FK; tests bare Party and double subtype; then derives C01 enforcement and decision on Party-kind immutability. No assistant-created CI yet. After CORE-01 and sqlc/Listing compatibility, move to Account → Organization Membership → permissions, then business slices.
**If local workspace contains deleted tracked migrations not committed**, stage/review the deletion intentionally; avoid `git reset --hard`, force-push, `git clean -fd` or blind `git pull`. Commit only user-approved changes.

## 6. Safe synchronization instructions for local dev (do not auto-run)
```bash
git branch --show-current
git status --short --branch
git remote -v
git fetch origin --prune
git log --oneline --decorate --graph --max-count=14 --all
git rev-list --left-right --count dev...origin/dev
git diff --name-status dev...origin/dev
```
Read before `git merge origin/dev` or `git pull`. If local has uncommitted migration deletions, first decide to commit those intentional deletions and preserve the clean local database baseline. Remote `main` and `dev` having the same `UNIFIED-STATE.md` does **not** automatically fetch content into a local working tree. Do not claim a local synchronization without user-run Git output.

## 7. Proof gates / subsequent updates
G0: Inspect user's actual local Git status and schema_version/database name → G1: reconcile local migration reset with remote `dev` doc merge → G2: the user writes minimal `000001` SQL → G3: user PostgreSQL PK/FK/UNIQUE/partial UNIQUE/transaction negative proof → G4: C01 design/immutability with concurrency → G5: sqlc/runtime compatibility and Go → G6: authored CI when relevant → G7: merge user-authored verified dev changes to stable main.

On a new chat "Tiếp tục", open this file on `dev`, read fresh local evidence, and continue nearest unverified gate. Do not equate historical CI on unmerged assistant-authored branches to user's newly reset local database.
