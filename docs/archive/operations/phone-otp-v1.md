> **Historical V1 design — not the current authentication contract.** Phone-only registration and OTP login are now implemented. Use [the current authentication guide](auth-three-methods.md) and [architecture contract](../architecture/authentication.md). This document is retained only to explain the earlier bind-first design.

# Phone OTP V1 — development-only authentication

## Identity contract

- Phone OTP is a new way to prove possession, not a new account system.
- Verified phone is linked to exactly one existing `accounts.id`.
- Phone binding requires an access JWT plus current BDSPro password.
- Successful phone login uses `auth.IssueSession` to issue the same BDSPro access JWT, refresh JWT and sessions as Google/password login.
- Organization member/owner authorization remains in SQL and is never determined by phone or IP.

## Routes (only active in ENVIRONMENT=development)

| Route | Admission | Action |
| --- | --- | --- |
| POST /users/me/phone/start | Authenticate + IPActiveSource + PrePhoneBind + IPActive(phone) | Check password, create bind challenge |
| POST /users/me/phone/verify | Authenticate + IPActiveSource + PrePhoneVerify + IPActive(phone) | Consume challenge and bind phone |
| POST /users/phone/login/start | IPActiveSource + PrePhoneLoginStart + IPActive(phone) | Create challenge for verified number |
| POST /users/phone/login/verify | IPActiveSource + PrePhoneVerify + IPActive(phone) | Consume and issue BDSPro session |

Use E.164 format, for example `+84901234567`. Codes are six digits, generated with crypto/rand, HMAC-protected using a key derived from the server's configured symmetric secret and bound to purpose, phone, account and code.

## Development delivery

The temporary `phone_otp_challenges.dev_code` column exists **only so developers can inspect an OTP in the development database**. No code is returned by HTTP or logged. Do not use production user phone numbers with this sender. API routes are not registered in other environments. A production rollout must integrate an out-of-band provider and remove or disable plaintext code storage before enabling.

To read a code in your disposable dev database:

```sql
SELECT phone_e164, purpose, dev_code, expires_at, failed_attempts
FROM phone_otp_challenges
WHERE phone_e164 = '+84901234567';
```

## Enforcement

- Five-minute OTP validity; resend cooldown 60 seconds.
- Up to five failed attempts, blocked for fifteen minutes.
- A new code within the same attempt window does not erase failure history.
- A DB row lock serializes attempts; consuming an OTP removes it in the same transaction as binding.
- IPActiveSource limits even malformed requests; IPActive adds normalized-phone limits.
- Unknown phone and cooldown have the same 202 response on login/start to discourage enumeration.
- No automatic account creation by number; no implicit role elevation; no SMS credentials committed.

These controls do not eliminate SIM swap or phishing risks. SMS OTP should not be treated as strong proof for high-risk operations.

## Verification

Run migrations 000001 through 000005 on a disposable test database, then `go test -count=1 ./...`, `go vet ./...`, and the race detector. Verify successful bind/login, replay rejection, failed attempts, lockout, throttling, unknown-phone neutral responses, missing JWT on bind, and production route absence.

**Integration requirement:** This feature branch is stacked on PR #41, which contains Google OAuth, SMTP and DRX-style request pipeline. Do not cherry-pick individual OTP files onto an older dev version or overwrite local WSL changes.
