# R24: model and module ownership for OAuth2/OIDC

## Scope

BDSPro is an **OAuth2 client** implementing the Authorization Code flow with PKCE and obtaining identity assertions via **OpenID Connect**. It does not implement Gitea's OAuth2 **authorization-server** entities (OAuth2Application, authorization grants, access token scopes) merely because those names exist in `go-gitea/gitea/models/auth`.

## What exists on this branch

- `models/auth/oauth2.go`: OIDC identity value, scoped by the exact `(issuer, subject)` tuple.
- `models/auth/oauth2_identity.go`: persistent identity lookup and explicit linking.
- `models/account/oauth2.go`: atomic Party/Person/Account/identity creation, verified email validation, rollback on constraint conflicts.
- `db/migration/000013_oauth2_identities.*`: additive migration with backfill of legacy identities; does **not** discard old data.
- `db/query/oauth2_identity.sql` and generated SQLC code: one query implementation; application models do not hand-edit SQLC output.
- `modules/setting/oauth2.go`: trusted issuer and endpoint configuration.
- `services/auth/oauth2_flow.go`: PKCE challenge, one-use Redis state, nonce and code exchange.
- `services/auth/oauth2_token.go`: signed OIDC ID-token verification.
- `services/auth/oauth2_auth.go`: account linkage/login and session workflow.
- `routers/api/v1/user/oauth2.go`: HTTP start, callback, explicit link, phone reauthentication.
- `modules/structs/auth.go`: request DTO, not an input for SQLC.
- `routers/api/response/errors.go`: public HTTP error shape; raw database and provider errors are not serialized.

## Model vs module conventions

- The model package owns persistent behavior and invariants, not arbitrary helper registries. Model operations may call SQLC, and by default return underlying query errors unchanged.
- `modules/setting` validates technical provider configuration. `modules/validation` checks email syntax. Provider trust and account linking are not owned by utility packages.
- `models/db` alone owns SQLC, pgx and transaction mechanics. No `modules/errors` framework.
- `db.Store` currently includes SQLC `Querier`; `ExecTx` passes transaction-bound `*db.Queries`. This is **not** equivalent to Gitea `db.GetEngine(ctx)`/ `db.WithTx(ctx)`. A transaction-propagation refactor requires proof of nested operations and isolation before being applied globally.
- HTTP DTOs, model options and generated SQL params have separate owners; use a distinct type only where it protects a real semantic boundary.
- `services/auth` orchestrates external transport and account/session operations; it must not be a generic mapper of every PostgreSQL error.

## Invariants and tests

- The identity key is `(issuer, subject)`, never email alone. Issuer is validated against configured trusted issuer.
- The OAuth2 code is exchanged only with the configured HTTPS token endpoint.
- State is random, browser-bound by HttpOnly SameSite cookie and consumed with Redis GETDEL exactly once.
- PKCE uses S256. A fresh nonce is verified in the signed ID token.
- ID tokens verify RS256 signature, key ID, audience, exact issuer, expiry, issue time if present, `azp` for multi-audience and nonce.
- PostgreSQL uniqueness prevents attaching the same external identity to another account.
- Link requires a verified local reauthentication proof; the callback alone does not authorize linking by matching email.
- Passwordless registration and identity insert share a transaction; account creation failure cannot leave an unlinked account.

## Remaining before integration into dev

1. Legacy Google-named source, endpoints, settings and schema are still present for compatibility and must be retired in a separate reversible change. **Historical migration files must never be rewritten**.
2. The earlier R16 flow move is a parallel unmerged PR; reconcile rather than merge duplicate mechanisms.
3. Complete real-provider smoke, migration rollback/upgrade edge cases, JWKS caching/rotation and provider compatibility review.
4. Compare Gitea context-bound transaction against BDSPro explicit executor on a real cross-model workflow before changing all model signatures.
5. Merge stacked R17–R23 into dev in order, with full CI after each integration, before R24; R14–R16 may overlap and require reconciliation.

Do not treat this draft as completion of the requested removal of Google-specific behavior.
