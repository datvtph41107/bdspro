# R21 — Authentication persistence ownership

`models/db` owns only the PostgreSQL pool, shared SQLC facade, transaction
execution, migrations and list infrastructure. OTP and phone-first signup
implementation belongs in `models/auth/phone.go` and
`models/auth/phone_signup.go`. No `PhoneOTPRepository`, custom pool or
shadow query executor remains in `models/db`.

Both operations use `db.Store.ExecTx` with generated SQLC queries.

**Security invariant:** when an incorrect OTP increments the failure counter,
the transaction closure intentionally returns nil so the increment is
committed. The model returns `auth.ErrOTPInvalid` *after* the successful commit.
Any error committing the transaction is returned instead, and valid identity
writes remain all-or-nothing.

`TestPhoneChallengeCommitsWrongCodeAndConsumesOnce`,
`TestPhoneChallengeLockoutAfterFiveFailures`,
`TestConcurrentPhoneProofOnlyOneSuccess` and the existing phone-first
registration tests exercise the actual PostgreSQL row locks and constraints.

`services/auth/phone_otp.go` owns delivery, login/session orchestration,
account-control proof and rate-limit semantics. The model owns persistence,
proof consumption and credential mutation.
