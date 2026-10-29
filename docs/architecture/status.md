# Implementation and acceptance status

This page describes current source responsibilities, not a claim of production
deployment. Use the latest GitHub Actions run for the exact revision and result.

| Responsibility | Current implementation |
| --- | --- |
| Process entrypoint | main.go |
| HTTP registration | internal/api/routes.go |
| HTTP middleware | internal/api/middleware |
| HTTP input and output | internal/api/handler and internal/api/response |
| Application operations | internal/service |
| Database schema and SQL | db/migration, db/query, internal/database |
| Password sign-in | password_credentials and AuthService |
| Google sign-up / sign-in / linking | google_identities and GoogleAuthService |
| Queue execution | internal/worker and native asynq.Handler |
| Durable job dispatch | job_outbox and OutboxDispatcher |
| E-mail business operation | internal/jobs/verify_email.go |
| Phone verification | internal/service/phone_otp.go and internal/notification |
| Primary phone ownership | phone_credentials, bound only after OTP; change is separate |

## Important behavior

- Authentication uses one AccountID principal, independent of the credential
  used to sign in. A Google ID token with a new verified email can create a
  passwordless Account + GoogleIdentity in one transaction.
- Google subject, not email, identifies a linked Google account. Existing emails
  are never automatically linked; the user must authenticate and link explicitly.
- Password login queries password_credentials. A Google-only account does not
  have a synthetic password hash.
- Phone-only registration and login are available when phone delivery is configured. Phone binding on an existing account still requires password confirmation or linked-Google reauthentication.
- Email verification is confirmed by POST after a non-consuming GET preview; resend uses the shared job_outbox. Verified email users, including Google accounts, may set/reset a password through a single-use email proof.
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
