# Refresh-session token storage: migration-safe digest

> **Historical storage-rollout discussion.** G1.2b now resolves the live
> first-party access session on protected HTTP routes. Refresh digest storage
> is unchanged; see [g1-live-session-authority.md](g1-live-session-authority.md).


The API issues a signed access JWT and a signed refresh JWT to clients. The
refresh JWT remains a bearer credential. PostgreSQL `sessions.refresh_token`
previously stored that raw credential.

## New write behavior

`services/auth.IssueSession` now persists the deterministic
`models/auth.RefreshTokenDigest` (`sha256:` followed by 64 lowercase hex
characters). The original JWT is returned to the client **once** and is not
stored in the sessions table. Database schema and SQLC remain unchanged.

- JWT signature, token type and expiry are verified before lookup.
- Renewal checks session ID, account ownership, blocked state, expiry and
  constant-time comparison against the persisted digest.
- Revocation first attempts the digest key, then falls back to raw JWT only for
  rows written before the rollout. This fallback does not accept an unsigned
  or expired token: verification occurs first.
- Password reset and revoke-all still invalidate DB refresh sessions.
- Access JWTs remain stateless and valid until their configured expiry.

## Deployment and sunset policy

The documented default maximum refresh TTL is 24h. **Do not remove the
legacy-row fallback until the longest configured refresh TTL at the actual
deployment has elapsed after every writer was upgraded.** Rolling deployments
must keep the digest-aware reader on all instances before enabling new writers;
otherwise old code may reject newly fingerprinted sessions.

This protects future session table snapshots against immediate bearer replay.
It does not replace defense-in-depth for DB access or revoke existing JWTs.

## Tests

Model tests cover digest format, wrong and legacy proofs, and expiry. Service
tests cover login/renew/revoke and legacy renew/revoke. Disposable PostgreSQL
HTTP acceptance verifies the exact digest stored at rest, not merely a mock.

Security review should later decide whether rotating refresh credentials on
every renewal is required; that is a separate, concurrency-sensitive protocol
change and is not implied by at-rest hashing.
