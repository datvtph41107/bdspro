# R06 — Email verification consumption in models/account

Gitea reference: `models/user` and `models/db/context.go` at `12b20c7c9962e0be75912b290a0c5778799b0ad2`.

- Keep **Gin** as the only HTTP router.
- `services/auth.VerifyEmail` validates public behavior and translates PostgreSQL no-row to the existing `invalid_verification_link` error.
- `models/account.VerifyEmail` owns one SQLC/pgx transaction: consume a one-use email proof and update the matching account.
- Generated queries, migrations, worker retries, SMTP and durable outbox remain unchanged.
- Integration test proves a code can be consumed once, updates the correct account, and cannot be replayed.
- `models/db.GetOrCreateVerifyEmailTx` remains in db temporarily because the verification worker's narrow test contract still consumes that operation. Moving it requires a separate worker/retry equivalence test, not a second live implementation.
- No phone OTP changes, GORM, new routing adapter or new public URL prefix.
