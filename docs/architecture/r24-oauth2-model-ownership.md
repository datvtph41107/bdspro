# R24 — OAuth2/OIDC model ownership (migration slice)

BDSPro acts as an OAuth2 **client** with OpenID Connect identity assertion, not as an OAuth2 authorization server. Gitea's `models/auth/oauth2.go` implements OAuth2 client authorization/grant entities for a different role; copying those entities would be incorrect.

## Completed in this migration slice
- Added `models/auth/oauth2.go`: verified OIDC identity value type keyed by `(issuer, subject)`; email is never an identity key.
- Added `models/auth/oauth2_identity.go`: SQLC-backed Get/Link model operations.
- Added `models/account/oauth2.go`: transactional passwordless account creation plus OIDC binding (rejects unverified email).
- Added SQLC queries for independent identities, schema migration and generated query facade.
- Copied existing Google identity bindings once into the new identity table without removing legacy records.

## Important outstanding work
- Provider-neutral Authorization Code + PKCE, nonce, state handling and callback.
- Cryptographic verification of OIDC ID token against trusted discovery/JWKS, including issuer, audience, expiry, signature and nonce.
- Migration of router, settings, service logic, tests, docs and old Google-specific names. Existing Google runtime remains unchanged on this branch.
- Backfill safety tests and concurrency tests for identity link / duplicate subjects.
- CI `make sqlc-check`, PostgreSQL integration and complete Go 1.27.3 validation.

Do not merge this draft as a completed OAuth2 migration. Never use `OIDCIdentity.Key` on unverified token claims.
