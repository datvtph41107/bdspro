# A4 — Provider and runtime acceptance

The authentication source supports email/password, Google OAuth, phone OTP
and verified credential linking. Passing CI validates the **offline** behavior
and disposable PostgreSQL/Redis contracts. It is not evidence of successful
delivery by a live SMS carrier, Zalo Business account, Google OAuth application
or production SMTP relay.

## Test environment

Use a separate non-production database and test identities. Load provider
credentials from the deployment secret store, never commit tokens, OTP values
or account credentials to Git. Configure an externally reachable HTTPS origin
and the exact allowed OAuth redirect URI when testing Google. For development,
Mailpit and the Local phone sender may be used to exercise backend contracts
without delivering to an external account.

## Acceptance sequence

1. **SMTP:** request reset for a verified test account; confirm HTTP 202 and a
   queued `job_outbox` entry without plaintext token. Exercise the configured
   worker and confirm delivery to the intended inbox; use the link once, verify
   refresh sessions are invalidated and the same token cannot be reused.
   Disconnect SMTP, confirm the worker returns an error and Asynq retry policy
   retains the task, then restore SMTP and verify delivery.
2. **SMS:** send OTP to an owned test number using configured HTTPS gateway.
   Verify gateway HTTP acknowledgement and real handset delivery independently.
   Test invalid token, HTTP 429, timeout, and rejected OTP; check no account is
   created for unverified phone and no duplicate phone ownership is possible.
3. **Zalo:** use an approved ZBS template and test phone recipient. Verify the
   provider's `error:0` response and recipient delivery. Reject malformed
   responses, nonzero errors and HTTP failures. Reconcile delayed delivery and
   ambiguous outcomes; provider success alone does not authenticate a user.
4. **Google:** complete the browser-based PKCE/state/nonce flow using a
   registered test OAuth client, confirm subject association and AccountID
   continuity. Test stale state, invalid nonce, incorrect issuer/audience and
   attempts to link a subject already owned by another account.
5. **Recovery:** restart API/worker with pending Outbox rows, disconnect Redis,
   restore it and ensure dispatch resumes. Test duplicate delivery and recovery
   without duplicate AccountID or unintended password changes.
6. **Operations:** perform a real backup/restore drill on a disposable instance,
   test migration rollback of a representative deployment, rotate provider
   credentials and inspect alerts/metrics for queue backlog and terminal jobs.

## Known constraints

- JWT access tokens remain active until expiration after refresh-session revoke.
- At-least-once SMTP delivery can produce more than one usable 15-minute reset
  link after ambiguous provider responses. Reset confirmation clears remaining
  account reset proofs.
- Some email/phone verification flows still send synchronously. A transient
  provider outage can leave a challenge subject to its cooldown period.
- The hardcoded Zalo endpoint requires real credentials/template approval to
  prove end-to-end compatibility. Tests using `httptest` cover only the HTTP
  contract and provider acknowledgement classification.
- None of the above live-provider or disaster-recovery checks is accepted
  until the operator supplies test credentials and records observed results.
