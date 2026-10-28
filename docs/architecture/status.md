# Implementation and acceptance status

This page describes current source responsibilities, not a claim of production
deployment. Use the latest GitHub Actions run for the exact revision and result.

| Responsibility | Current implementation |
| --- | --- |
| Process entrypoint | main.go |
| HTTP registration | internal/core/router/routes.go |
| HTTP middleware | internal/core/middleware |
| HTTP input and output | internal/api/handler and internal/core/response |
| Application operations | internal/api/service |
| Database schema and SQL | db/migration, db/query, internal/db/sqlc |
| Password sign-in | password_credentials and AuthService |
| Google sign-up / sign-in / linking | google_identities and GoogleAuthService |
| Queue execution | internal/worker and native asynq.Handler |
| Durable job dispatch | job_outbox and OutboxDispatcher |
| E-mail business operation | internal/jobs/verify_email.go |

## Important behavior

- Authentication uses one AccountID principal, independent of the credential
  used to sign in. A Google ID token with a new verified email can create a
  passwordless Account + GoogleIdentity in one transaction.
- Google subject, not email, identifies a linked Google account. Existing emails
  are never automatically linked; the user must authenticate and link explicitly.
- Password login queries password_credentials. A Google-only account does not
  have a synthetic password hash.
- The original Google linking flow still requires current-password confirmation.
  Additional credential-management methods need their own reauthentication rules.
- Registration writes a job_outbox row in the same PostgreSQL transaction as
  account creation; Asynq handles retry and task processing. Exactly-once SMTP
  delivery is not guaranteed.
- Redis key prefixes, OTP HMAC domain values and Asynq task types are protocol
  identifiers: do not rename them without a compatibility rollout.
- PostgreSQL authorization predicates remain in SQL; do not replace them with
  only preflight handler checks.

## Verification

GitHub CI checks SQLC code generation, migrations on disposable PostgreSQL,
unit/integration tests, Go race detection, go vet and go build. A successful
CI run is not evidence of a live Google OAuth browser flow, SMTP provider
delivery or production Docker startup.
