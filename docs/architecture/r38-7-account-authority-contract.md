# R38.7 — Account authority and credential lifecycle (contract checkpoint)

## Current supported authority

The current account row does **not** have a canonical suspended/disabled state. Do not claim account-wide lockout. Do not silently add one without defining its transition authority, status checks, and recovery path.

| Event | First-party refresh sessions | First-party access JWT | Pending OAuth codes | OAuth opaque access/refresh | OAuth client registration |
| --- | --- | --- | --- | --- | --- |
| Revoke one / all first-party sessions | Targeted sessions invalidated | Valid until expiry | Existing codes retain their own lifecycle | Unaffected | Unaffected |
| Revoke one delegated OAuth authorization | Unaffected | Unaffected | Deleted for account/client | Revoked for account/client | Unaffected |
| Password recovery reset | All target sessions deleted | Valid until expiry | Unused codes deleted for account | All target grants/tokens revoked | Unaffected |
| Account suspension | **Not implemented** | **Not implemented** | **Not implemented** | **Not implemented** | **Not implemented** |

**Policy boundary:** recovery removes existing durable credentials; it does not invalidate already-issued stateless first-party JWTs immediately. Claiming immediate account-wide lockout requires a separate authority mechanism at the relevant access resolution paths. Password recovery also does not disable phone or OIDC login proofs; those credentials have independent lifecycle rules.

## Required invariant after recovery

1. Recovery advances each existing `oauth_grants.version` and marks the grant revoked.
2. A stale consent preview (captured before recovery) must fail after recovery, even when supplied alongside a newly created session.
3. The authorizing session from before recovery must no longer authorize code issuance.
4. A newly authenticated session plus a *fresh* consent preview may approve a new grant; recovery does not permanently ban the client.
5. A registered OAuth client survives another account's recovery.

The added PostgreSQL integration regression tests 2–4. Existing R38.6 PostgreSQL tests cover token, grant and session revocation, unrelated accounts, rollback and races.

## Follow-on design gate (not implemented here)

Before adding an account status field, define: who can suspend, the difference between security lock and administrative suspension, session/token validity during suspension, authentication policy for password/phone/OIDC, status enforcement at refresh and access resolution, unsuspension proof, and audit evidence. Only then decide whether `accounts` needs a persisted status.

Reference implementation: `models/db/oauth_consent.go`, `models/db/oauth_recovery.go`, `models/auth/auth.go`.
