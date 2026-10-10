# BDSPro — Business → Verified Core → Incremental Implementation
Date: 2026-10-10. This is an execution playbook **not evidence of implemented code**.
Scope repo `datvtph41107/bdspro`. Design `architecture/canonical-database-final`, execution `dev`; do not rewrite unexplained local work, reset repository, blindly merge 160-table DBML or edit production schema merely from docs.

## 1. Go/no-go and two independent tracks
**Product gate (BUSINESS-00)** requires observation and pilot evidence before expensive or customer-facing production expansion: actual source and client workflows, privacy/consent practice, agent voluntary usage, minutes saved, callback failures, stock errors, buyer budget, and credible payment commitment. Proposed study ~5 teams/~15–25 agents; 2–4-week trial are hypotheses; results NOT already available.
**Engineering proof track** may begin now as bounded local PostgreSQL/Go exercises after reconciling live `dev` and recording risk. It is not a production launch. The user requests to progress into hands-on code after checkpoint. Do not substitute endless hypothetical business review for a runnable, test-driven narrow core.
**Do not claim:** entire business model fully validated, 160 DBML tables implemented, current dev head reconciled unless fetched, database parser or Postgres test passed unless executed.

## 2. Stable mindset (every gate)
ACTUAL JOB → WHO USES/PAYS → WHEN ACTING / CAPACITY → AUTHORITY + DATA SCOPE → ONE-ROW FACT → PRE/POST CONDITION → FAILURE/CONCURRENCY/RETRY → PG INVARIANT + MINIMAL SQL → READ UX → OBSERVABLE TEST → COST → DECISION/COMMIT.
- Write Core is authoritative, stable business facts. Read convenience can be UNION/view/query/derived projection, rebuildable or clearly versioned. Avoid per-handler duplicate branches and uncontrolled caches.
- Prefer one backend/process initially; explicit PostgreSQL SQL, migrations, sqlc, pgx/v5/pgxpool from present code; modular boundaries when pressure proves need, not premature microservices.
- One Account can work in different Organization contexts; never derive tenant permissions solely from Person identity or subscription. Organization-first enterprise invitation is legitimate; do not require personal paid plan.
- Constraint fit: simple CHECK/PK/FK/unique where possible; composite/partial indexes/range locks/triggers/deferrable constraints only for a **specific counterexample**. Tenant safety from DB FKs **and** backend resource auth; tests include malicious IDs and concurrent state changes.
- Good security at every stage: password/credential hygiene, TLS, secrets, sensitive data minimality, file validation, logs redaction, backup and restore proof; do not defer tenant safety as a paid feature.
- Separate 'designed', 'parsed', 'migrated', 'tested', 'operated', 'observed in customer usage'. Always report true state.

## 3. Implementation gates (do not merge all)
| Gate | Real problem and minimal scope | Data/API proof | Failure/exit evidence | Deferred |
|---|---|---|---|---|
| **ENG-00** | Reconcile `dev` Git tree + migrations/sqlc/Go package, local PostgreSQL; choose slice seam | actual `git status`, HEAD, schema migrations, sqlc generate/test, query inventory | no overwritten local source; runnable baseline tests; safety rollback | new tables until seam understood |
| **CORE-01 Party** | Create exact-one Person or Org identity (C01) | parties/persons/orgs, transaction API/DDL experiment; stable Party id | concurrent inserts; orphan/dual subtype rejection; deletion/recovery; explicit DB-21 lab output | full social identity graph |
| **CORE-02 Account/Auth** | Secure Account links to Person and session boundary | register/login principal, credential lifecycle minimum, idempotent safe Account create | recovery and revoked session; no auto account merging by email; no secrets in logs | passkey/SSO/SCIM/service clients until need |
| **CORE-03 Organization context** | ABC workspace, An invitation and membership, proper resource authority | org account context, membership, small permission policy; enterprise-first An flow | same-org FK + dynamic permission checks; cross-tenant request tests; offboarding revokes | branch/group/team hierarchy, legal claim engine unless partner requires |
| **BUS-01 Source capture** | House 75m2 S1 4.2bn declared + Confirm | source intake current row, source price report; operation key and tenant; one short Save action | retry, concurrent edits, wrong org, reported!=verified, no public Property | cadastral/eKYC/GIS |
| **BUS-02 Price & stock freshness** | S1 4.2→4.1, S2 independent 4.3 | append meaningful report, version; read latest eligible source status | backdated event, race, two sources coexist, no global listing price overwrite | event sourcing search pipeline |
| **BUS-03 Share preview** | Allowed truthful content from source | derived template render with source version; export gate if policy satisfied | revoked member/media, stale version, unknown fact stays unknown, export != posted | social platform automation/full CMS |
| **BUS-04 Inquiry** | Minh contacts An via Zalo about house after Facebook post | minimal ABC-scoped inquiry + next action, optional Contact link | no Account required for Minh; retry vs genuine new inquiry; no cross-tenant existence leak | pipeline+deal on first message |
| **BUS-05 Work** | An hands follow-up to Bình, reminder due, cancel/complete | assigned membership, due_at, status; atomic state transition and eventual reminder | concurrent transfer/complete; no double notifications; no status if data absent | algorithmic sales scoring |
| **BUS-06 Read UX & pilot** | Today dashboard, stock list, manager report | SQL read model scoped to org, report definitions, minimal metrics | read-your-writes, real capture coverage disclosed, user benefit/payer pilot | separate analytics DB/AI |
| **BUS-07 Appointment/Opportunity** | Only when actual pilot needs visits and qualified demand | external CRM Contact appointment, structured demand, qualifying opportunity | proposed vs confirmed vs attended vs cancelled; owner role recheck | Deals/payments as automatic follow-up |
| **ORG-01 legal/enterprise** | When legal-company claims/marketplace publisher required | organization_registration/claim/representation review; qualification evidence | competing claimants, reviewer revocation, first registrant cannot seize brand | enterprise IAM prematurely |
| **PUB-01 Listing publishing** | Lawful publisher makes Property-facing offer | property identity, listing, price terms, scoped grant, publication episodes | revoked mandate, private source not auto-public; multi-property scope | marketplace nationwide traffic |
| **BILL-01 commercial** | Demonstrated willingness-to-pay and recurring billing | subscriber Party, plan version, seats/rights, invoice/reconcile | no duplicated payment or cross-tenant license | wallet/financial platform |
| **EXP-N** | CMS/ads, Group, Deal/commission, asset rental, community, reputation, GIS, auction | each gets own actor/contract/regulatory/economic gate | legal license, dispute/ops cost, differentiated user value | all untouched until gate |

## 4. First executable work order — ENG-00 → CORE-01
1. Reconcile actual local `dev` branch: path, working tree, `git status --short`, `git log -1`, migrations `000001..000005`, SQL/sqlc queries, Go tests. Treat baseline summary as **historical**. Do not reset or checkout over local modifications. Remote repo connector cannot see local uncommitted files.
2. Pick disposable lab schema `c01_lab` in `bdspro_test` if still available and appropriately isolated (prior notes mention PostgreSQL 18.6, DB-20 finished, DB-21 **not executed**; verify actual server). Do not run against production.
3. State exact test: one Party must own exactly one subtype after transaction commit (Person XOR Organization), including concurrent writers; PK/FKs in two subtype tables alone do not ensure totality. Draw transaction boundary / constraint trigger alternative / pending Party creation strategy, choose trade-off, then write **small SQL test** and inspect failure behavior.
4. Retain exact observed proof (SQL, version, session isolation, execution result, adversarial cases, rollback), not just 'C01 done'. Review Account cardinality and personal-vs-tenant scope as next contracts.
5. Only after CORE-01 proof, move to CORE-02/03; choose how to connect existing Listing `dev` migrations by expand/backfill/compare/switch/contract rather than drop/create all.
6. For actual Go implementation respect `docs/continuity/PRACTICE-PROTOCOL.md`: user normally writes smallest code after understanding requirement, assistant supplies scenarios and escalation. If user explicitly requests full code, may implement with tests, never make unverified claims.

## 5. Database change discipline
At design branch: DBML is logical candidate, not SQL. A table addition needs one-row assertion, source actor/owner/custody, target FK/effective time, 3 negative cases, read model, measurement value, costs and reviewer state. Any V0.4 candidate addition must synchronize DBML/table group/data dictionary/count/readme/gap log and reconcile all 146 historical traceability positions. Keep 160 original table identities stable. Do not rename existing prod migrations wholesale or mark table names as implemented.
At `dev`: new versioned migrations only after `git status`/source verification. For schema migration use OLD→EXPAND→BACKFILL→COMPARE/PROVE→SWITCH WRITE→SWITCH READ→CONTRACT. Account for rollback, lock duration, tenant privacy and data retention.

## 6. Acceptance matrix per implementation gate
**Functional:** positive scenario and negative counterexample, API result and one-row meaning.
**Relational:** PK/FK/CHECK/unique/composite integrity; simultaneous transactions where needed; no nondeterministic last-write-wins.
**Security:** unauthorized/cross-org IDs, revoked memberships, sensitive media/PII, auditing without overcollection.
**Operations:** errors observable without secrets, retries idempotent, restore drill, reasonable performance and cost, manual support fallback.
**User/product:** measured capture frequency, time cost, stock freshness, successful callback/visit, manager report replaces—not doubles—work; buyer WTP and renewal evidence.
**Change control:** actual branch/commit/migration/proof/date recorded; if absent status is OPEN.

## 7. Spend gates / stop conditions
No paid external integration, automated social posting, bulk data ingestion, AI ranking or cross-company CRM before reliable rights and observed lift. Missing willingness to pay, data capture only under coercion, poor quality stock, product-dependent ad traffic, privacy incident or unmanageable onboarding are stop/simplify signals; do not justify sunk cost by expanding scope.

## 8. Prior review still open
C01 exactly-one subtype not SQL verified; organization claim/representative validity; employee-first without consumer Account; membership episode; actor/resource/role/permanent identity split; source vs property identity; price report vs listing term; Contact vs Inquiry vs Opportunity; due-work vs historical Activity; appointments with external contacts; original 160 candidate domain families; money/payee/legal licensing; CMS/SEO/ads; legal privacy/GIS quality. These are deliberate open items, not forgotten decisions.

## 9. Continuation behavior
On message **"Tiếp tục"** in a new chat: look up `docs/continuity/checkpoints/2026-10-10-business-to-core/README.md` from GitHub, read this implementation plan and freshest code/checkpoint, identify nearest unverified gate; if coding intent active, start ENG-00/CORE-01 **without repeating conceptual BUSINESS-03 from scratch**. State actual facts versus proposals, never claim 'automatic memory' guarantees if repository not accessible. Update checkpoint after proof. Reference `docs/continuity/MINDSET.md`, `PRACTICE-PROTOCOL.md`.
