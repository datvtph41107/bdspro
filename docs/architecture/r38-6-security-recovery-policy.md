# R38.6 — Password recovery and delegated OAuth credentials

> **Historical R38.6 behavior.** The original stateless first-party access
> policy is superseded for protected Gin APIs by G1.2b's live session check.
> Recovery still revokes OAuth delegations transactionally.
> See [g1-live-session-authority.md](g1-live-session-authority.md).


## Policy decision

**Ordinary session logout** (`POST /users/me/sessions/revoke_all`) invalidates first-party refresh sessions and browser cookies backed by those sessions. It deliberately does not invalidate independent OAuth grants issued to external applications; this is part of the API contract, not a failure of the current token service.

**Password reset after account recovery** is a higher-risk event. BDSPro now consumes its reset proof, changes the password, deletes first-party sessions, revokes existing OAuth grants and tokens, removes pending authorization codes and invalidates other reset proofs **within one PostgreSQL transaction**. Existing OAuth clients remain registered, including clients owned by another user. Users can authorize them again only through a new authenticated session. Internal stateless access JWTs on ordinary APIs still expire normally (R38.5).

RFC 9700 permits automatic refresh-token revocation upon password change or logout but does not mandate a single policy. BDSPro chooses the stricter option for password recovery, not for ordinary logout. See [RFC 9700](https://www.rfc-editor.org/rfc/rfc9700) and [OWASP Forgot Password](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html).

## Concurrency and consistency

The recovery transaction acquires `accounts FOR UPDATE` before changing credentials or revoking tokens. OAuth operations already take a per-account/client advisory lock; they now also hold `accounts FOR SHARE` before any grant or token transition. This serializes OAuth issuance and refresh with recovery, without introducing global Redis state or preventing concurrent share locks on the same account.

A login request may have validated an old bcrypt hash before recovery. For password-based login, `NewSession` compares this previously verified hash against the database again *after* inserting the session but *before* committing it. The session FK takes `KEY SHARE` on the same account, which conflicts with recovery's `FOR UPDATE`. Therefore, a login that reaches the session transaction only after recovery commits cannot issue credentials based on an old password. The session and connection history roll back together on mismatch. Phone/OIDC login remains controlled by its own proof and is not disabled merely by resetting a password.

The reset transaction advances existing consent epochs, revokes grants and tokens, deletes unused codes, and deletes sessions. If a later OAuth UPDATE fails, the password update, reset-token consumption and session deletion all roll back. The proof is retryable after transient infrastructure recovery.

## Known boundaries

The canonical `accounts` schema has no account-disabled state, so this work does **not** implement account suspension. That requires a separate explicit status transition and authority checks at login, refresh, OAuth access resolution and read paths. Access JWTs on ordinary APIs remain stateless under the existing expiry policy; do not describe password reset as immediate invalidation of every issued JWT. No migrations, triggers or alternative middleware stacks are introduced.

## Acceptance cases

1. Password reset atomically invalidates target sessions, OAuth access/refresh tokens and pending authorization codes.
2. A different account's sessions and authorizations remain intact, and OAuth client registration is not removed.
3. A failed OAuth revoke causes full transaction rollback, including reset proof consumption.
4. Concurrent refresh versus reset never leaves a usable new OAuth token.
5. Session-only revoke preserves independent external delegation.
6. A verified old password cannot create a new session after recovery commits.

PostgreSQL row-lock semantics: https://www.postgresql.org/docs/18/explicit-locking.html
