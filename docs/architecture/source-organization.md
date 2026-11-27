# Source organization and ownership

Baseline: `4f63b56`; Gitea v1.24.7 (`99053ce4fa2b45f1bca5837418c0c57f793ca824`) is a design reference, not a mandatory copy. BDSPro uses SQLC and PostgreSQL, not XORM.

## Ownership rules

| Scope | Owns | Does not own |
| --- | --- | --- |
| cmd | CLI, initialization and shutdown composition | domain use cases |
| routers | HTTP routing, binding, admission, response mapping | DB transaction orchestration, provider protocol |
| services | business operations, invariants, transaction lifetime, delivery policy | arbitrary HTTP provider wire formats |
| models | domain types, data operations and persistence semantics | HTTP response writing, external delivery |
| db/query | static SQL, named SQLC query contracts | business workflows |
| modules | cohesive technical capabilities (settings, log, web, transport, crypto, time) | application-specific use-case orchestration |
| services/context | shared HTTP adapter (explicit exception to business services) | domain transaction orchestration |

Dependencies normally flow from routers to services/models/modules; services can use models/modules and other services with explicit contracts; modules must not depend on upper layers. Direct model access for simple reads is acceptable. Do not add wrappers for every SQLC query. Existing exceptions are audit items, not reasons for a sweeping rewrite. Settings may depend on technical types where necessary; avoid cycles.

File names describe one cohesive responsibility or reason to change. Split provider implementations by protocol; no one-function-per-file rule. Prefer standard library operations over trivial helpers. Utilities must have a stable, reusable semantic contract; policy belongs to its domain. Keep explicit Go error checks, preserve errors.Is/As and transaction ordering, keep secrets out of logs. Use named constants for real policy/operational limits, not every literal. Do not generate new interfaces or packages without a consumer. No arbitrary line-count limit. Keep existing API/security contracts while refactoring.

References: [Gitea mailer](https://github.com/go-gitea/gitea/tree/99053ce4fa2b45f1bca5837418c0c57f793ca824/services/mailer), [Gitea team model](https://github.com/go-gitea/gitea/blob/99053ce4fa2b45f1bca5837418c0c57f793ca824/models/organization/team_user.go), [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).

## Canonical engineering backlog (authoritative)

This section is the single status ledger for the branch. Historical decisions below are evidence, not a second checklist. Update this section in the same commit as each scope; link exact CI evidence when available. A passed smoke run is not full product/security acceptance.

### Definition of done (every scope)

- [ ] Pin branch HEAD and prior CI result; inventory file paths, public and private symbols, imports, callers/callees, configuration, SQL/migrations, generated bindings, docs and runtime consumers as applicable.
- [ ] Record current behavior, required business/security invariants, unit/format and canonical owner; distinguish authoritative policy from projections, caches and independent checks at trust boundaries.
- [ ] Decide KEEP / INLINE / DELETE / MOVE / MERGE / REWRITE / DEFER per candidate with concrete evidence and alternatives. Use standard Go operations directly; do not create speculative wrappers, interfaces, packages, or duplicate policies. Preserve useful security/transaction boundaries even if code looks similar.
- [ ] Implement one cohesive scope, remove obsolete consumers and imports, update generated artifacts only from source, update docs and record explicit contract changes (none by default). Do not merge unrelated scopes.
- [ ] Check gofmt, diff and build; run real local/disposable runtime probes for relevant success, invalid input and failure paths. Use focused automated tests when timing, concurrency, redaction, boundary semantics or a difficult-to-reach invariant cannot be demonstrated reliably by an ordinary API call. Keep existing opt-in suites; distinguish actual tests from skipped tests.
- [ ] Review error propagation, data privacy, auth guards, SQL lock/transaction ordering and compatibility where applicable. Record commit, exact CI URL, checked cases, skipped cases and rollback via revert. Do not call a scope verified until the relevant evidence exists.

### Sequential work

| ID | Scope / concrete work | Acceptance evidence | State |
| --- | --- | --- | --- |
| REQ-00 | Canonical ownership: define layer, package, file and policy owners; maintain this ledger; audit exceptions | Ownership register + per-scope decisions; final cross-check REQ-13 | In progress (rules recorded) |
| REQ-01 | Delivery/mail: provider contracts, channel selection, transport and templates; delete redundant adapters; verify provider timeout/error/privacy | Disposable Mailpit email/phone probes; real provider acceptance separately | Implemented; scoped smoke passed |
| REQ-02 | Utility: inventory util/timeutil/optional/validation/secret; delete dead helpers, competing validation, unused generics | Build and scoped smoke after fix 221b6c1; retained semantic contracts | Scoped verified |
| REQ-02.1 | Function design: direct stdlib, cohesive files, identity and actor guard ownership, remove forwarding | CI [45965376546](https://github.com/datvtph41107/bdspro/actions/runs/45965376546); broader function audit REQ-13 | Scoped verified |
| REQ-02.2 | Errors: error category, safe client code/message, diagnostic operation, transport mapping; remove redundant types and helpers | CI [45966107081](https://github.com/datvtph41107/bdspro/actions/runs/45966107081); expanded privacy tests opt-in | Scoped verified |
| REQ-02.3 | HTTP input: remove unused Form/Query wrappers; canonical query/path handling; one validated page for SQL and Link/JSON | Build, focused pagination contract and smoke; full org workflow separately | Implemented; new CI pending |
| REQ-03 | Time and policy: inventory JWT/session/OTP/recovery/invite TTL, cooldown, Redis window, SQL interval, mail/provider/API values; assign owner and units; remove independently maintained duplicates | Compare configured and persisted expiry, response, resend boundaries; preserve DB-clock guards and in-flight records | Todo |
| REQ-04 | Auth primitives: password hashing, JWT, OTP/MAC, proofs, browser, refresh, Google, randomness, initialization; eliminate wrapper/format duplication, document key/algorithm ownership | Wrong/expired/replayed proof, password policy/capacity, origin, token type and signature; no live credentials | Todo |
| REQ-05 | Persistence: migrations -> SQLC queries -> generated Go; inventory inline SQL, query aliases, transaction and DB error contracts, pagination model | Fresh migration, SQLC generation/diff, transaction rollback, constraints; no manual generated edits | Todo |
| REQ-06 | Account creation: compare email/phone/Google registration, identity normalization, verified contact, person/account initialization, uniqueness and terms | Real local creation paths, duplicate identities, rollback and verified-contact invariants | Todo |
| REQ-07 | Session/credential lifecycle: issue/refresh/revoke, reauth, unlink, password change, contact mutation; one guard and authority per invariant | Revocation, refresh replay, credential removal, concurrent mutation/lock order; session authority preserved | Todo |
| REQ-08 | Org/team: membership/ownership, permissions, invitations, naming/validation, audit and transactional boundaries | Owner/member/nonmember matrix, last-owner guard, invitation replay and membership concurrency | Todo |
| REQ-09 | API DTO/response: request binding, PATCH absent/empty/null, public errors, pagination, JSON shape, Swagger contract | Real API status/body/header compatibility and invalid fields, no internal data leak | Todo |
| REQ-10 | Router/context: routes, middleware order, adapters, request data, binding and response status, client IP; remove unjustified abstractions after consumer inventory | Route inventory, request order, cancellation, one response, trusted proxy and body limits | Todo |
| REQ-11 | Startup/settings/logging: config defaults/env/INI, resource composition, log sinks, startup failures, shutdown, CLI | Local config check, invalid config, port conflict, SIGINT/SIGTERM, Docker/binary, redacted logs | Todo |
| REQ-12 | Legacy/background: job/outbox/cron/worker and dependencies, commands, docs, feature flags; delete only after verifying operational consumers and migration compatibility | No stale routes/config/deps, build, startup, migrations; explicitly account for existing persisted jobs | Todo |
| REQ-13 | Final sweep: every production Go package/file/exported and private function, tests, config keys, migrations, queries, workflows, docs and deps; reconcile canonical register, ownership exceptions and stale references | Complete inventory with disposition, no unexplained duplicates, build/runtime acceptance, documented residual risks | Todo |

### Canonical ownership register (current / target to verify)

| Concern | Authority / boundary | Consumers or projections | Status |
| --- | --- | --- | --- |
| Basic string/number/time/error operations | Go standard library | All layers | Rule; final sweep pending |
| Email/username identity validation | services/auth/identity.go | Registration, contact, Google, login (lookup normalization is distinct from validation) | Partial; REQ-06 |
| Password hash/policy and credential proof formats | modules/auth | Auth workflows, persistence of hashes/proofs | REQ-04 |
| Actor mutation guard | services/auth/access.go, DB locks executed within transaction | Credential/profile/session/org mutations | Partial; REQ-07/08 |
| Public error category/code/safe message | Domain declarations + modules/util/error.go | services/context/error.go HTTP mapping, frontend | REQ-02.2 scoped |
| HTTP pagination | routers/api/v1/utils.GetListOptions; models/db.ListOptions represents validated page/size and computes offset | SQL, Link, JSON; pass same values | REQ-02.3 |
| Path positive ID / UUID semantics | net/http PathValue; positive ID contract in services/context; UUID parse at endpoint | Route handlers | REQ-02.3 |
| Runtime durations / defaults | Owner to determine per policy, not a universal time package | DB deadlines, response, mail, providers, Redis | REQ-03 |
| Stored data and constraints | DB migrations, SQLC source queries; generated output is a projection | Models/services | REQ-05 |
| Authorization and lifecycle | Domain owner + database constraints/locks; HTTP early guards are independent boundary checks | Routers/services/models | REQ-07/08 |
| Configuration | modules/setting load/validate; runtime composition in cmd | Technical modules and services | REQ-11 |
| Logging | modules/log, static operation metadata | Runtime sinks, HTTP error correlation | REQ-11 |

No global policy manager or generic canonicalization package is prescribed. One authority per semantic rule does not mean one function for all operations. Duplicate expressions are acceptable when semantics differ; duplicate independently editable business policy is not.

## REQ-01 decisions

- modules/mail owns SMTP and message encoding; services/mailer owns registration, password, credential notice and team invitation text. Existing error identity retained via mailer.ErrUnavailable.
- Provider adapters are private implementations in services/phonedelivery. No independent module consumer was found; remove modules/phonedelivery and its three type aliases.
- services/phonedelivery owns immutable config snapshot, default/allowed channel selection, concurrency budget, result logging, development Mailpit guard. Mailpit sender uses modules/mail directly. Provider receipt is private; SendCode returns only error, because all business callers discarded the receipt.
- PrepareWebRoutes constructs the single service before static route registration (no network or DB I/O). OTPHandler method values capture it for only five dependent endpoints. Delete services/phonedelivery/context.go; keep context.Context for cancellation, timeout and log correlation. No global service or general DI framework.
- Retain early router channel checks before admission, auth checks before challenge writes and final sender validation. Removing them requires checking ordering and callers separately.
- Preserve provider URLs, request/response fields, timeouts, redirect refusal, no retry/fallback, existing OTP wording/TTL, error responses and logging privacy. TTL ownership addressed by REQ-03, not this change. No schema changes.
- Existing SMTP integration test relocated to mailer; message test remains with module. No additional mandatory Go test gate. The existing CI runtime step starts disposable Mailpit and calls real email/phone start and verify APIs, reads OTP only from its local inbox, and never prints codes or proof tokens. This does not verify live provider delivery or all mailer consumers.

## REQ-01 follow-up: evidence for each surviving production file

Baseline: [f89619c](https://github.com/datvtph41107/bdspro/tree/f89619c1f0cf5f177b30be2558ebc010ce79b2a1). Its build, migration, HTTP/SMTP/phone Mailpit probes and SQLC check passed in [run 45962286517](https://github.com/datvtph41107/bdspro/actions/runs/45962286517). The next revision needs its own runtime result.

| File | Concrete callers / reason to exist |
| --- | --- |
| services/phonedelivery/service.go | PrepareWebRoutes -> New; owns shared client, enabled sender map, slots; no external I/O during creation |
| channel.go | OTPHandler.PhoneChannels/PhoneStart/PhoneResend/StartPhoneLink, auth.StartPhone/ResendPhone/StartContactLink, Service.SendCode; default and enabled-channel policy, copied public options |
| delivery.go | auth.StartPhone, ResendPhone, StartContactLink -> SendCode; canonical OTP validation, capacity, provider dispatch, safe logs |
| sender.go | sender map and four implementations share message/receipt/interface; HTTP handlers use Failure/ErrUnavailable; receipts not exported |
| http.go | Twilio/Zalo/Telegram use exchange; Zalo/Telegram use jsonRequest; response cap, safe errors, no leaked upstream text |
| twilio.go | New chooses SMS -> twilioSender.Send, form and BasicAuth |
| zalo.go | New chooses Zalo -> zaloSender.Send, approved template and provider error status |
| telegram.go | New chooses Telegram -> telegramSender.Send, Gateway verification protocol |
| mailpit.go | New chooses development -> mailpitSender.Send; local-only guard and OTP inbox via mail.SendPlain |

Deleted: independent modules/phonedelivery package, exported transport constructors and aliases, request dependency context, redundant NormalRoutes forwarding function. Authentication continues to own code generation, MAC, DB and verification; no business flow or API/schema change. Private Mailpit adapter relies on SendCode validation of canonical phone/code and New validation of channel, still checks development/local SMTP at its own I/O boundary. Earlier router checks before admission and auth checks before challenge mutation remain, since removing them changes rejection ordering. No cross-repo full line audit claim: remaining scopes REQ-02–13 require their own consumer map and acceptance.

## REQ-02: utility ownership audit

Baseline [6b1fbc3](https://github.com/datvtph41107/bdspro/tree/6b1fbc37e1de3c631bd7cc46944568c6fcdccf85). Prior runtime [45963214397](https://github.com/datvtph41107/bdspro/actions/runs/45963214397) passed build, migration, HTTP/OTP Mailpit probes and SQLC check. This revision requires its own build/runtime result.

Method: enumerated all 11 Go files in the five utility packages, then checked imports and named call sites across the other 226 non-generated Go source/test files at the pinned tree. Checked README, architecture/operations docs, workflow and Makefile for references. This is a utility-specific reference audit, not an exhaustive semantic review of every Go function.

| Source at baseline | Actual responsibility / consumer | Decision |
| --- | --- | --- |
| modules/util/random.go | RandomInt, RandomString, RandomPersonName, RandomOrganizationName, RandomEmail; no Go consumer outside file | Delete; math/rand/v2 must not become credential randomness |
| modules/secret/derive.go and random.go | ValidDerivationKey, DeriveToken, RandomHex; no imports of this package in audited Go source | Delete; auth's purpose-bound HMAC, crypto/rand and proof formats remain unchanged |
| modules/validation/validation.go | PasswordLength (8 runes, 72 bytes), EmailAddress; no imports. Actual modules/auth/password.go validates UTF-8, 15–128 runes and <=512 bytes; services/auth/password.go owns actual email normalization | Delete stale competing policy, not migrate it into live auth |
| modules/util/optional.go | OptionalArg only used by APIErrorNotFound and Base.FormString | Delete generic variadic fallback; use explicit local defaults, preserve first argument semantics |
| modules/timeutil/expiry.go | Expired only used by Payload.Valid | Inline !time.Now().Before(payload.ExpiredAt); exact deadline still expired |
| modules/timeutil/timestamp.go | TimeStamp and TimeStampNow used only by ipactive.Event and redis.checkKey; no evidence of need for conversion/addition methods | Use int64 Unix seconds, preserve field and units |
| modules/timeutil/duration.go | SecondsCeil used by ipactive Redis expiry conversion and policy validation | Keep rounding for Redis; validate Window > 0 directly. No float conversion or overflow-prone addition |
| modules/optional/option.go | PATCH profile/settings/org/team; absent vs empty vs explicit null are distinct | Keep JSON contract, Has/Value, Some/None (existing test construction); remove unused FromPtr and ValueOrDefault |
| modules/util/error.go | Shared domain/public error mapping across services, routers and context | Keep; errors.Is/As and safe error semantics are meaningful |
| modules/util/operation.go | Auth operation annotation and log diagnostic traversal | Keep; do not replace with raw error text |

Gitea comparison: [TimeStamp implementation](https://github.com/go-gitea/gitea/blob/99053ce4fa2b45f1bca5837418c0c57f793ca824/modules/timeutil/timestamp.go) and [Follow.CreatedUnix](https://github.com/go-gitea/gitea/blob/99053ce4fa2b45f1bca5837418c0c57f793ca824/models/user/follow.go#L18) show a persisted timestamp type with real model consumers. BDSPro uses PostgreSQL timestamptz and generated time types; its only TimeStamp consumer is an admission event, so the copied abstraction is not justified. [Gitea optional](https://github.com/go-gitea/gitea/blob/99053ce4fa2b45f1bca5837418c0c57f793ca824/modules/optional/option.go) is a reference, not an API surface BDSPro must reproduce.

No schema, password policy, OTP/MAC, JSON PATCH null behavior or API status contract changes. No replacement generic secret/random package. REQ-03 will address actual time policy duplication across SQL, response, mail and providers; REQ-04 will review auth-specific crypto and token primitives separately.

## REQ-02.1: function boundaries and reading order

- Use strings/strconv/time/errors directly for ordinary transformations. Do not create aliases for standard library expressions. A reusable identity validator is not an alias: it owns input canonicalization, format limits and the public rejection contract. Keep login identifier normalization separate from email validation because their accepted inputs and rejection paths differ.
- Group file declarations and operations by responsibility. Prefer the main workflow followed by its private implementation details; shared policy stays with its existing semantic owner. Do not split files by number of functions or by mechanical operation (string, error, conversion).
- Use early returns for errors. Use a condition switch when alternatives of the same result require distinct handling; retain errors.Is/As and specific-before-general order. Do not replace error checks with generic wrappers, and never expose raw internal errors to clients. Keep typed public error mapping separate from transport writing.
- Moved NormalizeEmail and ErrEmailInvalid from password.go to identity.go, next to username validation, without changing its exported signature or accepted input. Moved the actual actor mutation lock into access.go as LockActorForMutation and removed the forwarding function/private duplicate name. Updated callers within services/auth and reused the same guard in RevokeSession. Account lock, active account check, session lock and error identity are preserved. No schema, SQL, credential formats or API changes.
- Baseline 9c3afe2 failed build because services/context/api.go retained an unused util import; corrected in 221b6c1. Runtime acceptance for this refactor requires a fresh CI run. See [Go review comments: function boundaries](https://go.dev/wiki/CodeReviewComments#line-length) and [error flow](https://go.dev/wiki/CodeReviewComments#indent-error-flow).

## REQ-02.2: error design and HTTP boundary

Baseline [b303081](https://github.com/datvtph41107/bdspro/tree/b3030813d7f65ab74892b52db5003c49889f1430), acceptance [45965376546](https://github.com/datvtph41107/bdspro/actions/runs/45965376546) passed. Reference: [Go error wrapping](https://go.dev/blog/go1.13-errors), [Gitea team errors](https://github.com/go-gitea/gitea/blob/99053ce4fa2b45f1bca5837418c0c57f793ca824/models/organization/team.go#L29), [Gitea HTTP errors](https://github.com/go-gitea/gitea/blob/99053ce4fa2b45f1bca5837418c0c57f793ca824/services/context/api.go#L113).

- Declare domain errors with their owner, use errors.Is/As and ordinary early returns. A public error retains category, internal message, stable client code and safe client message; these are distinct contracts. Do not add one type or predicate per error when a sentinel is enough.
- Removed WithKind, WithCause and APIErrorNotFound: no callers found in default-branch code search or the 55 inspected IAM model/service/router/context files at the baseline. This was a bounded reference audit; compile/test must confirm other paths. Flattened publicError: no embedded kindError, same Error/Unwrap behavior and constructor signature.
- Grouped API error declarations, classification and response methods in existing services/context/error.go; API context construction stays in api.go. Inlined PublicError classification in its only observed caller APIErrorAuto. Shared redacted 500 writer between automatic and forced internal paths; retain both public entry points and their existing distinct event names. No arbitrary err.Error() in HTTP response/log. The explicit nil fallback remains only on the forced internal path.
- In writeAuthError, invalid credentials reuse the existing canonical sentinel response instead of duplicating the code/message; Bearer header, generic unauthenticated and password capacity priority unchanged. Do not generalize this auth-specific switch to all domains.
- Keep WithOperation: structured static operation metadata is consumed by log diagnostics, not equivalent to fmt.Errorf text. Keep WithPublic/ErrorForClient: stable safe error metadata and errors.Is/As. Existing diagnostic/privacy tests must continue passing. No schema, security policy or API contract change intended; fresh CI required.

## REQ-02.3: request input and pagination

At baseline cc0b946, GetListOptions already validates page/limit and SQL offset, but SetLinkHeader rereads the raw page through FormInt and silently defaults. Pass the validated page and size from both list handlers instead. Remove base_form.go (five helpers, only observed surviving caller FormInt), DefaultQuery/QueryDefault and PathParam forwarding; positive path ID validation remains a real contract. UUID consumers use net/http PathValue and preserve canonical non-nil UUID checks. No changes to database pagination, HTTP response shape, URL generation or validation precedence intended. Targeted regression check deliberately supplies a raw page distinct from validated page, because ordinary valid list requests cannot distinguish the two data paths. Scoped audit of API handlers/context and default-branch code search, not exhaustive; new CI must verify build and focused test. Full authenticated org list runtime remains pending REQ-08/09.
