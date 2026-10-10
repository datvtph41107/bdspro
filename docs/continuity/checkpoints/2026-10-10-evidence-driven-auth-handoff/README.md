# BDSPro checkpoint — Concrete reasoning method + AUTH implementation handoff

**Recorded:** 2026-10-10 (Asia/Bangkok)
**Repository:** datvtph41107/bdspro — active implementation branch dev
**Purpose:** a reproducible handoff of the user's clarified reasoning/pair-programming method and the next authentic source work. This is NOT a claim of implemented Auth or validated 164-table schema.

## 1. Critical user-confirmed working style

The user explicitly endorsed the method now specified in:
**docs/continuity/EVIDENCE-DRIVEN-REASONING.md**.

Central requirement:
> Do not pronounce conclusions such as "simpler", "more secure", "fewer joins", or "more maintainable" without a visually reconstructable actor, example data, before/after state, actual mechanism or query, a fair competing design under identical conditions, a counterexample, relevant primary source, and what the evidence still does not prove.

A good explanation makes the user mentally see the data and actions: Minh/Hòa registration, real rows, actual SQL, expected vs observed result. Explicitly distinguish fictitious business scenarios from observed market realities, a DB mechanism from justified business policy, and a valid FK from actual authorization.

The user performs local Git, migration creation, SQL, PostgreSQL, Go/sqlc/pgx, tests and IDE edits themselves. Assistant supports with sufficiently concrete, usable small snippets/commands when useful or requested, asks for minimal decisive evidence, debugs real output, and proactively continues without repeated "shall we proceed?" questions. Avoid full scaffolding, patronizing quizzes, theory-only circles and architecture-first rituals.

**Default format for each significant design choice:** concrete scene -> state and data -> mechanism -> fair alternatives/counterexample -> experiment to run -> output & non-guarantees -> proportionate decision -> next hands-on operation.

## 2. Exact evidence and its limits at checkpoint

### Remotely verified on GitHub dev
- Prior observed HEAD 8f77dfc had Listing migration versions 000001..000005 (10 up/down files) and Listing Go/sqlc queries.
- GitHub later recorded commit **545d405e00052971f21c3953d101ac63c808933d**, message "chore(db): remove legacy listing migrations", involving those 10 paths. Checked remotely that migrations/000001_create_listings.up.sql is absent on dev.
- db/query/listing.sql and sqlc.yaml still exist on dev; sqlc.yaml reads schema from migrations. Their existence implies a potential compile/generation/runtime integration debt, not proof of failures until checked.
- Repo README and go.mod record Go + PostgreSQL + golang-migrate + explicit SQL + sqlc + pgx/v5 approach (Go module at github.com/datvtph41107/bdspro).
- Canonical V0.4 = **164 logical candidate tables**, not executable or business-proven implementation. Existing parties, persons, organizations, accounts, account_emails, password_credentials, sessions, passkeys, external_identities, role and membership definitions are review inputs.
- New method file created on dev in subsequent docs-only commit **3973bcb7b60eb52a0421d0cff4deaa434b5c618d**.

### Last known user-reported LOCAL state and uncertainty
- Earlier terminal output: user was on local dev at 8f77dfc, with 10 deleted Listing migrations in the working tree; later typed git push, which said "Everything up-to-date" BEFORE request to remove migrations.
- User then explicitly instructed removal of all legacy migrations; assistant suggested stage/commit/push; remote now contains the deletion commit.
- **The conversation does not include fresh local git status/HEAD output after that deletion commit.** The remote commit does not prove present local working tree, staged state, additional files or installed DB schema.
- **No user-provided evidence yet of new Auth migration applied, sqlc regenerated, fresh PostgreSQL DB lab result, tested Go registration/login or session endpoints.** Old DB-01..20 historical lab output and unmerged PRs do not substitute for current runtime proof.

### Repository sync after these documentation edits
- Docs were updated directly on remote dev as explicitly requested; user local branch may be behind these docs-only commits. Before any next local edit/merge, user should inspect local git status and refs and synchronize deliberately, preserving uncommitted local changes. Never claim docs already landed in local files or local DB.

## 3. Active engineering goal: Auth/Authorization CORE, not bulk conversion

The user's current intent is to implement Auth Core by concrete business behavior, with alternatives tested against both canonical and live source:

1. Business job and actors: e.g., a visitor reads permitted public Listing data, Minh registers, authenticates and accesses own protected data; other domain contacts need not have platform Account. Do not confuse "buyer"/"broker"/"employee"/"admin" with hard-coded Person subtypes.
2. Boundaries: Party/Person stable business identity; Account access identity; email/phone login identifiers; credential; session; authorization on a specific action and resource. Test whether each distinction is justified; do not blindly deploy all entities.
3. Compare model alternatives with the **same observed behavior**. Example: merged users vs Person + Account; including Hòa known before signup, alternative CRM Contact only, lifecycle/account closing, 2 credentials vs 2 accounts, incorrect account linking.
4. Canonical candidate accounts.person_id is a NOT NULL FK but not UNIQUE; account_emails UNIQUE(account_id,email) is not globally unique. These are *concrete, testable* pressures. They do not by themselves establish correct Account cardinality or login identifier policy.
5. C01 Party exactly one Person XOR Organization at commit is a design invariant, **not enforced merely by PK/FK**, historical tests are not current schema proof. Assess later with a purposeful transaction/concurrency example rather than immediately copying a trigger.
6. Evaluate login method with actual actors, UX, recovery, failure and costs. Email+password, OTP/magic link, OIDC and passkey are candidate alternatives; do not assume self-hosted password or JWT is automatically best. If implementing password, use approved password hashing, no plaintext; session/CSRF/cookie/credential hygiene cannot be postponed for public use.
7. Bounded implementation: one justified schema change -> user creates up/down migration -> PostgreSQL positive/negative/concurrent scenarios -> sqlc query/generate when schema supports it -> Go pgx transaction and error handling -> authorization test -> explicitly qualified commit. No all-at-once 164-table migration.

## 4. Immediate next executable gate

When the user next says "Tiếp tục", do NOT ask if they want to proceed or reopen the entire methodology. Recover actual current local source/DB evidence first only as needed for a safe mutation (especially the deletion commit and existing sqlc Listing dependencies), then **lead directly into a concrete registration/auth behavior**. Explain why its first persisted fact and migration boundary are needed, demonstrate at least one fair alternative with actual row/query consequences, give the single next local command/file/code operation. The user writes/runs it. Review concrete output, including what did NOT get proven.

Avoid inventing a migration sequence or asserting a fresh clean DB without observation. No forced choice form; choose a small sensible experiment and adjust from evidence. The next target is a runnable, user-authored identity/account thin slice, not documentation-only review.

## 5. Recovery reading order

1. docs/continuity/CONTINUATION-PROMPT.md (new top-level current override)
2. docs/continuity/CHECKPOINT.md (new top-level current override)
3. docs/continuity/EVIDENCE-DRIVEN-REASONING.md (central new style standard)
4. docs/continuity/MINDSET.md
5. docs/continuity/PRACTICE-PROTOCOL.md
6. docs/continuity/RESPONSE-PROTOCOL.md
7. docs/continuity/REPOSITORY-WORKING-CONTRACT.md
8. docs/database/final/README.md and relevant slices from DBML, data dictionary, actual Go/sqlc/migrations, business Atlas and official primary references.

**Evidence priority:** current observed local source/database > current remote ref/source > actual tests > current documented decision > historical summaries / conversation memory.

**Correct outcome of a restored conversation:** user sees, understands and personally executes one meaningful next action while keeping the long-range Auth/Authorization Core in view.
