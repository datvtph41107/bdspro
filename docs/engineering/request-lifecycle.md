# BDSPro Request Lifecycle

Compared with laravel-drx main: app/Http/Kernel.php, routes/api.php, modules/IPActive, PreLoginMiddleware, AuthenticateUser, SufLoginMiddleware and ManageDevice.

DRX teaches action-specific controls and readable middleware naming. BDSPro deliberately does NOT copy TrustProxies='*', trusted CF-Connecting-IP from arbitrary requests, logging complete HTTP requests, or checking OTP after token issuance.

## Active Gin pipeline

External request -> HTTP header/read/write constraints -> CORS -> Gin direct client IP -> request ID / recovery -> RequestMonitor -> body size limit -> route-specific pipeline -> Controller -> Service -> Store -> response and safe structured evidence.

- Register: PreRegister -> IPActive(register) -> createUser -> auth.Register.
- Password login: SufLogin wraps [PreLogin -> IPActive(login) -> loginUser -> auth.Login -> CreateSession].
- Email verification: PreVerifyEmail -> IPActive(verify_email) -> VerifyEmailTx.
- Refresh/revoke: PreToken -> IPActive(action) -> token controller -> session store.
- Protected organization route: Authenticate(access JWT) -> controller -> SQL membership/owner Authorize and update.

## Names and precise owners

| Name | Responsibility | Restriction |
| --- | --- | --- |
| RequestIngress | Body size, transport timeouts, trusted source | No identity or resource decision |
| PreLogin and Pre* | Parse and validate shape, store typed input | No JWT issuance |
| IPActive | Redis atomic per-action IP and identity quotas | No resource authorization |
| Authenticate | Verify access JWT; set account ID | No automatic owner rights |
| Authorize | SQL predicates within the affected resource query | Never trust user-supplied role |
| Controller | Map valid input to service call and result to HTTP | No duplicate rate policies |
| Auth Service | Verify credentials then create tokens/session | No post-token authentication checks |
| Store | Transactions, constraints, ownership | No HTTP codes |
| SufLogin | Emit final login outcome only | No session mutation |
| RequestMonitor | Denial evidence and repeated-event detection | No auto-ban, no raw payload logs |

## Detection semantics

security.request_denied on 401/403/413/429 includes request ID, safe route pattern, method, status and a short pseudonymous IP fingerprint. No raw URL, passwords, tokens, or raw IP. Redis records the number of denials per source for 10 minutes; at the 12th denial, emit one security.suspicious_activity event. These thresholds are initial operating assumptions, not proof of malicious intent. A monitoring failure does not change authorization decisions; an IPActive Redis failure rejects auth endpoints with HTTP 503.

IP hashes are pseudonymous, not irreversible anonymization. A direct-peer IP may represent a shared NAT address. A reverse proxy must be explicitly configured before relying on forwarded headers.

## Verification

Run Go tests, vet, race detector and Go build in CI with disposable PostgreSQL and Redis. Test invalid/malformed payload, fake forwarded IP, concurrent Redis admission, 401/403/429/413 events, alert threshold, non-leakage, and unchanged SQL authorization.

Google OIDC and Phone OTP belong to separate integration branches and are not activated by this change.
