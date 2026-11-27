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

## Sequential backlog

- REQ-00 Canonical ownership: documented here.
- REQ-01 Delivery and mailer: separate provider files, application email composition, dev inbox and channel policy; code implemented; runtime verification tracked separately.
- REQ-02 Utility inventory: consumer map for util/timeutil/optional/validation/secret; remove dead or misplaced code with evidence.
- REQ-03 Time/duration/policy: define TTL/cooldown owner and SQL/response/provider consistency; preserve database clock and guards.
- REQ-04 Auth technical modules: token/password/OTP/browser/Google/runtime ownership.
- REQ-05 Models and SQLC: classify models and locate static SQL.
- REQ-06 Account initialization: compare email/phone/Google flows; share only invariant operations.
- REQ-07 Session and credential lifecycle.
- REQ-08 Organization and Team.
- REQ-09 API DTO and response mapping.
- REQ-10 Router and request context.
- REQ-11 Startup and configuration composition.
- REQ-12 Background/legacy consumer audit.
- REQ-13 Final source consistency review.

For each bounded scope: inspect callers/callees, compare Gitea and current code, document ownership and alternatives, implement, build, start server and exercise relevant real APIs, report limits and commit. No mandatory new automated tests at this development stage; existing tests retained as opt-in. Do not claim full IAM/production acceptance from a health probe.

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
