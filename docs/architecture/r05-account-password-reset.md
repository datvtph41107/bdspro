# R05 — Account credential reset ownership

Reference: Gitea `models/user` and `models/db/context.go` at `12b20c7c9962e0be75912b290a0c5778799b0ad2`.

BDSPro preserves **Gin, SQLC, pgx, existing SQL migrations and public HTTP contracts**.

## Operation

`services/auth.ResetPassword` performs HTTP-independent password validation and hashing, then calls `models/account.ResetPassword`.

`models/account.ResetPassword` owns one transaction:
1. consume the single-use password reset proof;
2. upsert the password credential;
3. delete other outstanding reset proofs for that account;
4. delete refresh sessions.

`models/db` retains only the transaction executor and generated SQLC queries. This change introduces no new database queries, migration, generic repository or ORM.

## Tests

The existing PostgreSQL integration test `TestPasswordResetOnceAndCredentialCreation` checks changed password and token replay, and additionally checks that a refresh session is removed. Do not replace it with a mock-only test.

## Exclusions

Phone OTP challenge and phone-first signup deliberately retain their commit-on-wrong-code semantics in `models/db` until those operations can be moved with a dedicated transaction-level failure-injection proof. No changes to Google OAuth, JWT access-token expiry or Redis policies.
