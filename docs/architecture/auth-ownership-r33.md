# Authentication ownership and route-first conventions

Source comparison: Laravel DRX main `767737d3a9530d0ec3ff94876752e40ccb302a15`, Gitea main `4477d47d0715b7881ce2dac21e4394984cad8a1f`.
BDSPro base: dev `f5c3f9bf7d0d161d3cc602c91f1da8a3b407a0f7`.

## Source of truth

* `routers/api/v1/api.go` declares the HTTP contract. The URL and HTTP action express intent; middleware decorates admission and observation, not a second business dispatcher.
* `routers/api/v1/user/auth.go` owns HTTP register/login/verify/recovery/session endpoints, preserving public URLs, throttle policies, error/response formats. `oauth2.go` owns browser redirect/state/PKCE/callback protocol and `profile.go` owns self-service profile HTTP.
* `services/auth/auth.go` owns the single `AuthService` for password, phone OTP and session lifecycle. `services/auth/oauth2.go` handles verified external identity linkage/login using the same `AuthService`; OIDC transport, cryptographic verification, JWKS and state remain in their respective client/token files.
* `models/account/account.go` owns account creation/lookup, `models/account/email.go` owns the account email lifecycle. `models/auth/auth.go` owns persisted password and phone challenges/transactions; `models/auth/session.go` owns session checks/revocation and refresh credential fingerprints. `models/organization` groups organization/team operations by actual lifecycle.
* `models/db` owns SQLC, connection and transaction primitives. `modules/auth` contains cryptographic JWT/password primitives; `modules/ipactive` is technical admission.

## Naming and non-goals

Laravel DRX `issueToken` is an OAuth token endpoint. BDSPro's `/users/login` is native account authentication, not an OAuth authorization server. Do not rename it to `/oauth/token` without implementing the corresponding protocol. `encrypt_pass` in the DRX snapshot is a no-op and must not be replicated. DRX `suf_login` checks OTP and device after its controller; never copy that ordering into BDSPro credential issuance.

Use Go MixedCaps methods and explicit action names `Login`, `VerifyPhoneLogin`, `LoginOIDC`, `ResetPassword`. Within services, `issueSession` is the single point where authenticated methods converge. No extra router DSL, middleware chain loop, plugin registry or unused compatibility packages.

## Security invariants preserved during this structural refactor

* Same public routes, status codes, JSON and token types.
* Same SQLC authorization predicates, PostgreSQL constraints, OTP failed-attempt writes, transaction boundaries, OAuth2 issuer/subject linking and PKCE/state verification.
* Same IPActive middleware and per-credential AuthThrottle for every endpoint.
* Existing access tokens remain stateless; revoking refresh sessions does **not** instantly invalidate access JWTs.
* Phone challenge delivery and OTP/session atomicity remain unchanged; define recovery before changing them. Device approval and rate/traffic benchmarks are **not** claimed as completed.
* PR #139 remains separate: do not merge login-connection-history semantics into a behavior-preserving refactor without explicit reconciliation and end-to-end tests.

## Acceptance

Run formatter, `go test ./...`, route inventory, SQLC reproducibility and PostgreSQL/Redis E2E; verify optional OIDC/Phone channel route counts and correctness. Compare built route manifest to baseline. A moved file without changed public behavior is not a new feature.
