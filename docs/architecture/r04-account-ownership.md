# R04 — Account persistence ownership with SQLC

Gitea reference: `go-gitea/gitea@12b20c7` — `models/user`, `models/auth`, and `models/db/context.go`.

## Changes
- `models/account/account.go: Register` owns Party → Person → Account → password credentials → email verification outbox in one transaction.
- `models/account/google.go: CreateGoogleAccount` owns verified Google-first signup and identity linking in one transaction. It never links another account by email.
- `services/auth` still owns application workflow: password normalization and hashing, identity error classification, OAuth rules, and token/session issuance.
- `models/db` continues to own connection pools, SQLC-generated queries and explicit `ExecTx`. Removed the duplicate manual registration/Google signup methods.
- PostgreSQL integration tests have been moved into `models/account`; existing API acceptance tests now call the same model functions.

## Deliberate retention
- `models/db/tx_phone_signup.go` retains a special security transaction: wrong OTP attempts must be committed while an invalid attempt is returned. A generic callback that rolls back on error is **not** equivalent.
- Other credential, email verification and password reset transactions remain unchanged until an operation-level owner and commit/rollback semantics can be demonstrated. This is not a blanket renaming exercise.
- SQLC and versioned migrations are canonical; no GORM and no second transaction engine.
- Gin remains production; HTTP methods, paths, JSON, tokens, status and middleware unchanged.

## Acceptance
Run Go 1.27.3 GitHub Actions using PostgreSQL 18, Redis 7 and disposable `bdspro_test`, checking registration/outbox, Google first-login uniqueness, passwordless accounts, phone OTP state commits, route inventory, OpenAPI, race, build, Docker and smoke. CI must pass on both PR and merge commit before calling it accepted.
