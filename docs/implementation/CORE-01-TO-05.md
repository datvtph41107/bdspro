# CORE-01 → CORE-05: Architectural decision record, implementation and acceptance

**Repository:** datvtph41107/bdspro, single implementation branch dev
**Decision baseline:** canonical DBML V0.4, 164 logical candidate tables, NOT a blanket physical schema or production approval.
**Implementation stack:** Go 1.27.1, pgx/v5, PostgreSQL 18, sqlc, versioned golang-migrate SQL.

## A. Why these modules exist together

One user registers (Account), proves email ownership, logs in (Session), is reviewed as a representative of an Organization, becomes initial workspace administrator, invites another Person, and the authorized Person captures a source observation and its reported price history. A complete path is necessary for database invariants and HTTP access checks to have a real actor; building Source first using client-supplied account_id would hide the authorization boundary.

## B. One-row truths and actor scenarios

1. Party = durable business identity, Person and Organization mutually exclusive subtypes. Account = digital login identity of Person; Profile = presentation, NOT login identity.
2. account_emails = contact methods and verification evidence; a username is not a verified Person. password_credentials = one current password hash; sessions = revocable session token hash; no plaintext password/token in DB.
3. Membership = an episode of participation, not legal employment or ownership; Role = org-scoped permission bundle; RoleAssignment and temporary MembershipBlock are independent facts.
4. RegistrationRequest = a declaration asking to register NEW legal entity; ClaimRequest = a request for administrative control of EXISTING Organization. Neither creates immediate admin rights. Reviewer authority is platform-scoped, not inherited from any tenant role or commercial tier.
5. Invitation = proposal; Acceptance creates Membership and RoleAssignment atomically after matching verified AccountEmail.
6. SourceIntake = one company-scoped incoming observation of potential real-estate stock, possibly inaccurate. SourcePriceReport = one report at a point in source/history time. Neither automatically creates Property, Listing, publication or legal ownership.

## C. Key invariants, mechanism and proof pressure

| Invariant | Enforcing mechanism | Residual |
| --- | --- | --- |
| C01 one Party exactly one Person or Organization | parties.kind + subtype composite FKs + DEFERRABLE INITIALLY DEFERRED constraint triggers | Verify concurrent moves/deletes; other migrations referencing Party must preserve it |
| Account belongs to Person, max one open per Person | FK accounts.person_id; partial UNIQUE WHERE closed_at IS NULL | Account merge/recovery policy out of scope |
| Email case-insensitive key uniquely maps login | UNIQUE lower(btrim(email)), primary-email partial UNIQUE | Real mail local-part case semantics and unverified-address squat require policy |
| Password never stored raw | Salted Argon2id PHC hash; separate credential table | Password recovery/rate limiting/abuse controls not implemented |
| Session bounded and revocable | Secure random token, SHA-256 stored hash, expiry and revocation, Account closure checked on reads | Reverse-proxy TLS, browser CSRF and session rotation require end-to-end review |
| Tenant-scoped Role and Membership | Composite organization_id/member/role FK, active Membership, Blocks; permission checks | Concurrent permission revocation and data writes require serial consistency policy |
| Reviewer authority separate from tenant admin | platform_organization_reviewers granted by privileged operator SQL; review row locks | A DB superuser bypasses ordinary application boundary; admin audit and verified evidence are operational obligations |
| Initial workspace admin differs from legal ownership | Organization primary administrator FK to same-org Membership | Representation and licensing need separate scoped legal proof |
| Claim cannot steal current admin | Row-level lock on Organization plus primary-admin uniqueness | Legal disputes/transfer remain separate workflow |
| Invitation acceptance cannot replay | Conditional UPDATE pending → accepted, transaction creates Membership and Role | Multiple pending invites/re-invitations and pre-existing member require product policy |
| Report prices preserved | Append-only application writes, no mutation API; recorded/statement time separated | DB superuser can still UPDATE, immutable policies/audit need privilege gates |
| Price never linked to wrong tenant | Composite FK (organization_id,intake_id) | FK doesn't authorize actor: every read/write must check membership, Role and Blocks |
| Retry doesn't double-create | Unique scoped request key + normalized payload SHA-256, replay read | Hash canonicalization changes need versioning; rollback/retention policy not implemented |

## D. Comparison with alternatives and impacts

### Identity subtype enforcement
- Mere FK Party→Person/Organization ensures child points to parent but **not** exactly one subtype. Deferred PostgreSQL constraint trigger lets a transaction create Party before child while enforcing final commit state.
- Costs: trigger logic, migration/test maintenance, consideration of subtype-key updates. In exchange, arbitrary Go endpoints cannot leave a bare Party committed.

### Account and presentation
- Password, profile/avatar, email-verified state are kept outside durable Party/Person roots. This reduces mixing presentation and auth lifecycles but adds joins and separate migrations.
- One login plus per-Organization memberships avoids duplicating human credentials for every enterprise; workspace switching must still prove current membership/role each request.

### Organization onboarding
- First claimant does not automatically become admin; Registration vs Claim are distinct. For new legal entity, reviewer inserts registry identifier under jurisdiction/code uniqueness. For existing Organization with primary admin, Claim approval intentionally refuses takeover. Reviewer must be a platform-privileged account independently provisioned.
- Costs: review operations, evidence handling, waiting/pending flows, potential dispute process; alternatives such as automatic first-claim lower friction but expose brands to hijack.

### Source vs Property/Listing
- Alternative A: INSERT directly into listings; wrong because Zalo report does not prove property, title or right to publish.
- Alternative B: one mutable latest price; shorter SQL, but destroys prior and contradictory source assertions.
- Chosen: SourceIntake and append-only PriceReport with atomic initial-price insertion, extra FK/index/storage. The recorded_at timestamp is not proof that source statement_at is newest.
- Metadata/price stored as strings+NUMERIC, avoids float64 money rounding. Idempotency stored per scoped request (extra bytes/indexes), but network retry does not clone source or reported price.

### Repository and engineering
- Removed obsolete Listing prototype runtime and generated code after previous migrations were removed from dev; Git history retains them. Full DB migration baseline is now 000001..000005, not 164 physical tables.
- Core SQL uses explicit pgx transactions; Source data path uses sqlc generated Go. This trades SQLC consistency for focused integration speed. Revisiting sqlc for Auth/Organization is reasonable after the behavior is validated; do not assume equivalent contracts automatically.
- New dev commit sequence is organized by business units; each commit and CI run is evidence only for its exact tree.

## E. Security and operations TODO; not hidden acceptance claims

- Client account_id never trusted. Sessions are random and hashed in DB; no cookies readable by JavaScript; Secure cookie in non-development.
- SMTP token issuance after registration commit is not yet durable outbox/retry; if delivery fails, Account can remain unverified and require operator assistance. Do not claim delivery reliability.
- HTTP Origin / Sec-Fetch-Site policy and SameSite=Lax are *not a complete CSRF architecture*. Before public launch add CSRF tokens and formal same-origin browser scenario tests.
- Need rate limiting and anti-enumeration timing review at ingress for login, register, token verify, review/invite endpoints.
- Must provision least-privilege DB roles; current Docker root local setup is NOT production operational separation.
- Business legal evidence, platform reviewer approval trail, tax registry matching and personal data handling depend on jurisdiction/process. A supplied text reference by reviewer is not legal verification by itself.
- Audit events, test data retention, data deletion, media rights, source privacy, production secrets management and observability need separate gates.
- PostgreSQL default isolation permits revoke-vs-write races: for strict immediate revocation choose row locks/authorization revision and test serial ordering.

## F. Acceptance matrix

- Source-feature baseline on separate branch: GitHub CI previously passed 11 isolated PostgreSQL 18 tests. These were *fixture tests*, not main application acceptance.
- dev 000001..000005: must be tested as one SQL migration chain, including DOWN/UP on disposable CI DB.
- Register → Verify → Login → Workspace approval → Invitation → Source/Price: integrated main tests must PASS under fresh PostgreSQL DB.
- Negative tests: bare Party commit, duplicate email rollback, unauthorized reviewer, wrong-org role/price FK, duplicate invitation acceptance, org takeover, block/revoke/read denial, idempotency concurrency.
- External/local filesystem and database synchronization can only be confirmed with user local git/psql outputs; remote CI cannot claim it.
- Production release/real market validation not part of this engineering acceptance.

## G. Next product gate

When dev CI is green: reconcile user's local uncommitted work non-destructively, then run fresh DB migrations and the complete server locally, record git commit hash + migration version + request evidence. Proceed to CORE-06 CRM Inquiry/Follow-up only after CORE-05 module passes this end-to-end pilot gate.
