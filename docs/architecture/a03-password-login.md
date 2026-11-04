# A03: password-login account-existence hardening

## Threat and bounded mitigation

A password login used to stop before bcrypt when an email did not identify an
account, or when the account had only phone/OIDC credentials. A request with an
existing password credential performed a costly bcrypt comparison. An attacker
can statistically observe the difference even though the HTTP error is the
same.

The authentication service now performs a fixed bcrypt comparison using an
unlinked dummy hash for both missing-account and missing-password conditions.
The hash has the same bcrypt default cost used for newly generated passwords.
A wrong password for a real password credential still uses its stored hash.
Each failure retains the public `auth.invalid_credentials` contract.

This change does **not** guarantee constant-time password login end-to-end:
database lookups, pre-existing credential costs, request admission, network
latency and infrastructure failures remain variable. In particular, keep
IPActive and account-target admission controls independent of this defense.

No new package or repository interface is introduced. Unit tests verify the
dummy hash format/cost and the same canonical model failure across all three
missing/invalid credential cases. The existing login acceptance suite proves
successful logins still issue sessions.
