# OAuth2/OIDC ownership and accepted source conventions (R24–R26)

## Integration evidence

The implementation was merged to `dev` in order through PRs #108–#111,
#113–#118. The final R26 merge commit is
`b86af22351d4d4967ffdbe6b305b9a41f53f7c92`.
R24 introduced the protocol client, R25 standardized credential terminology,
and R26 retired the provider-specific runtime.

BDSPro is an **OAuth2 client** with OIDC authentication, not an OAuth2
authorization server. It does not implement Gitea's OAuth2 application,
grant, or access-token-scope entities merely because those names exist in
`go-gitea/gitea/models/auth`.

## Ownership and naming

- `models/auth/oauth2.go` declares `OIDCClaims` from a **verified** ID token.
  Claims are not the stored credential and do not authenticate themselves.
- `models/auth/oauth2_credential.go` declares `OAuth2Credential` (a named
  alias for generated SQLC persistence), `OAuth2CredentialKey` and
  Get/Link/List operations. The key is exactly `(issuer, subject)`, never email.
- `models/account/oauth2.go` creates Party, Person, Account and the
  corresponding credential in one PostgreSQL transaction. Duplicate email
  does not silently merge accounts.
- `models/db` owns pgx, SQLC, migrations and the transaction executor.
  Handwritten model logic belongs to `models/auth` / `models/account`.
- `modules/setting/oauth2.go` validates trusted issuer, authorization,
  token, JWKS, client credentials and redirect URI configuration.
- `modules/validation` checks reusable email syntax; account linking policy
  remains in models.
- `services/auth/oauth2_client.go` is the OAuth2 Authorization Code/PKCE
  coordinator (previously `OAuth2Flow`); it manages one-use Redis state,
  nonce, code exchange and the trusted OIDC token verifier.
- `routers/api/v1/user/oauth2.go` owns cookies, redirects and HTTP binding;
  `modules/structs` owns HTTP request types, not SQLC query parameters.
- `routers/api/response` translates only established model errors to public
  HTTP codes. Unexpected pgx errors are propagated, never serialized.

The persistent table is `oauth2_credentials`, not `oauth2_identities`.
The old name remains ONLY inside immutable historical migration artifacts
and associated migration test documentation. SQLC source lives at
`db/query/oauth2_credential.sql`; generated output is never hand-maintained.

## Database history and compatibility

1. `000013_oauth2_identities`: originally introduced the table and backfilled
   legacy account links. Immutable migration history must retain that name.
2. `000014_oauth2_legacy_sync`: provided a temporary bidirectional bridge.
3. `000015_oauth2_credentials`: renamed the active table and its indexes.
4. `000016_oauth2_legacy_archive`: archived the old provider-specific table
   as `oauth2_credentials_archive` and removed active dual writes.

The rollback test `tests/integration/migrations/oauth2-credentials-upgrade.sh`
exercises upgrading, rolling back R26 and re-upgrading while retaining account
IDs and external credential bindings. Never deploy the old binary against the
post-000016 schema before reversing that migration.

## Security properties and evidence

- Authorization Code with PKCE S256, random one-use Redis state, browser-bound
  HttpOnly SameSite cookie and signed-token nonce checks.
- Trusted issuer, audience, expiry, issued-at, authorized-party and RS256
  signature verification. Public JWKS keys have a bounded cache and key-ID
  refresh; HTTP redirects are never followed for sensitive token exchange.
- External credentials are indexed by issuer and subject, not email.
- Explicit account reauthentication is mandatory for link; verified email
  alone is not sufficient. Phone binding requires ownership proof.
- PostgreSQL enforces one linked subject per account per issuer.
- CI covers rollback/upgrade, SQLC, tests, race detector, Go vet/build,
  route/OpenAPI contract, Docker and HTTP smoke. A mock-provider browser E2E
  also tests login, replay prevention and PKCE.

## Limits and unresolved architectural choices

- One trusted OIDC issuer can be configured at a time; multi-provider
  registration and OIDC Discovery are not implemented.
- Real-provider credential configuration and production end-to-end smoke
  require deployment-specific secrets and are not claimed as completed.
- BDSPro's explicit `db.Store` + `ExecTx(func(*db.Queries))` is NOT Gitea's
  context-propagated `GetEngine(ctx)` / `WithTx(ctx)`; transaction composition
  across models needs its own comparative implementation and isolation tests.
- Models and modules ownership is standardized for this OAuth2 slice; this
  does not imply every planned BDSPro model domain is implemented.

R14–R16 were parallel experiments; R16's Google-specific implementation is
superseded by R24–R26, not automatically merged as an additional runtime.
