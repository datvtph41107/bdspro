# R24 — OAuth2/OIDC model ownership (draft foundation)

The model now defines a provider-independent **OpenID Connect** external identity key: `(issuer, subject)`. Email is profile information and cannot be an account-linking key.

OAuth 2.0 is an authorization protocol; OIDC is the identity layer required for sign-in using an external identity provider. BDSPro is acting as an OAuth2/OIDC *client*; Gitea's `models/auth/oauth2.go` includes authorization-server entities, which are not the same role.

## Not yet implemented

- OIDC Discovery and trusted provider configuration
- Authorization Code + PKCE state, nonce, callback and token exchange in provider-independent code
- ID token validation against trusted issuer, audience and JWKS, with caching and rotation
- SQL migration and generated SQLC for external identities keyed by issuer and subject
- Account linking, first-login creation, concurrency and unlink rules
- Gin route/API error updates, Redis state compatibility and migration from `google_identities`
- Full Go 1.27.3 CI and integration tests

**Do not merge this draft as a complete OAuth2 implementation.** Existing authentication remains unchanged, deliberately. `OIDCIdentity.Key` validates key shape only; it does not verify a token.
