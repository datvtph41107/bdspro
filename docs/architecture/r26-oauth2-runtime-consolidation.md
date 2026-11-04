# R26 — One OAuth2/OIDC runtime

BDSPro authenticates through one configured OIDC issuer using the OAuth2
Authorization Code flow with S256 PKCE and verified ID tokens.

- `models/auth/oauth2_credential.go` owns issuer-subject credential linkage.
- `oauth2_credentials` is the sole writable credential table.
- `services/auth/oauth2_client.go` owns protocol state and token exchange.
- `routers/api/v1/user/oauth2.go` owns the HTTP browser boundary.
- All former provider-specific endpoint handlers, model operations, config,
  SQLC queries and errors were retired; Google can now be configured as an
  ordinary issuer (not a distinct authentication implementation).

## Data safety
- 000013 backfilled old provider bindings.
- 000014 synchronized old and new records during rolling compatibility.
- 000015 renamed the active table to `oauth2_credentials`.
- 000016 **archives** the old table as `oauth2_credentials_archive` and
  turns off write mirroring, without deleting old rows.
- Roll back 000016 before deploying a legacy binary; its down migration
  reconciles the legacy table from the **current** OAuth2 credential source.
  Do not run old and new binaries against the post-cutover schema at once.

## Acceptance
`sqlc-check`, migration upgrade/rollback tests, OIDC browser E2E, same-subject
concurrency, phone reauthentication, Docker and HTTP smoke must succeed.
Do not merge until R24/R25 and the R17–R23 stack are integrated into dev.
