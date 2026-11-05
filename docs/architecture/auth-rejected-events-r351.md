# R35.1 — Rejected authentication proofs

**Stacked dependency:** #140 R33 canonical Auth ownership → #141 R34 atomic successful session/history → #142 R35 connection retention → R35.1 PR #143. Apply migration 000020 only after 000019. No merge of old PR #84 migration 000013.

## What is recorded

`authentication_failures` is an append-only security-evidence table **only for rejected authentication proofs**. Its `account_id` may be null for unknown identities, and a known account FK cascades on account deletion. `method` and `reason` are restricted by PostgreSQL. Successful login/session creation is never duplicated here: `models/auth.NewSession` continues to atomically insert a session plus `account_connection_histories`.

The actual password/OTP/refresh token is never persisted. The attempted email/phone/issuer-subject is represented only as HMAC-SHA256 of a method-scoped identifier. Emails use BDSPro's lowercase lookup normalization, E.164 phones preserve the normalized phone value, and OIDC issuer/subject remain case-sensitive (no case folding). Domain separator `auth-rejected-proof:v1` prevents cross-protocol comparison. This R35.1 implementation uses the existing signing secret (minimum 32 bytes) rather than introducing another secret before lifecycle requirements are agreed; **rotating that key breaks correlations across the rotation**, and the signing secret must never be exported with the audit rows. If the key is absent/short, no event is written.

Observed client IP comes from Gin trusted-proxy resolution, not the JSON body, and is normalized by `net.ParseIP`; unknown/invalid input yields an empty value. Raw IP is personal data and requires access controls, retention, protected database backups and privacy review. Password/OTP fields and request body must not enter structured logs.

## Owners and failure behavior

- `services/auth.AuthService` observes the **actual rejected proof decision**; it invokes `models/auth.RecordAuthenticationFailure` only for expected invalid-credential results.
- `models/auth` owns method-scoped HMAC and nullable account mapping; `models/db/sqlc` owns typed SQL operations.
- For a **failed login**, writing this secondary event is best-effort. A database write failure is logged with a stable event and error *type*, not error text, IP or credential. An expected 401 stays 401. A database failure during credential lookup still uses the ordinary 5xx path and is not mislabelled as a bad password.
- **Successful session issuance is different**: session + successful-login history must commit atomically before credentials are returned, per R34.
- IPActive and `AuthThrottle` reject some requests before the AuthService. Their existing structured logs / admission records are intentionally **not** copied to PostgreSQL on every 429: database writes per rejected flood request would amplify attacks. Formal pre-auth abuse aggregation/metrics belongs to operational acceptance, not this feature.
- Rejected OAuth2 *service identity decisions* may be written here; cryptographic/front-channel OIDC protocol failures rejected before `LoginOIDC` do not enter this table. They need independent security logging policy.

## Cleanup and operation

The single `services/worker.AuthenticationRetention` lifecycle now deletes both successful connections and rejected proofs independently at startup and hourly, with the same 90-day cutoff, up to 1,000 per batch and 20 batches per table per sweep. Failure in one table's sweep does not starve the other; both are retried on the next hour. SQLC uses `FOR UPDATE SKIP LOCKED` to bound work under multiple workers. If the worker is stopped, database rows do not expire automatically. Backup and restored snapshot retention remain separate operational responsibilities.

## Acceptance boundaries

Required: sqlc reproducibility and migration rollback, DB18 integration test for null account, known-account link, method/reason check, domain-scoped digest, retention preserving recent events, service tests for public error parity and audit-write fault injection, race/build and Docker smoke. No device registration, JWT instant revocation, HTTP activity API or full cross-surface audit registry is introduced. The older #84 rejected-password history cannot be merged as a parallel implementation without explicit reconciliation.
