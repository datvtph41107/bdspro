# R08 — Verification proof issuance belongs to the account model

Gitea reference: `models/user` and `models/db` at `12b20c7`.

- Move the locked-account, reusable verification token operation from `SQLStore` into `models/account.GetOrCreateVerifyEmail`.
- `services/jobs.SendVerificationEmail` remains responsible for secret generation, URL rendering, SMTP delivery and retries. It now depends on `GetAccountByID` and the shared explicit transaction method, not an account-specific method on `db.Store`.
- A DBTX fake verifies the job still reuses its persisted proof when SMTP fails. A disposable PostgreSQL integration test verifies reused code, successful verification, and refusal to issue a code after verification.
- The SQLC query text, migrations, outbox task schema and Gin routes do not change.
- Phone OTP remains under the existing special commit-on-failure transaction contract.
