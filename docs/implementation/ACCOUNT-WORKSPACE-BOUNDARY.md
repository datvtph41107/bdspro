# CH10–CH12 — Authenticated Account → Active Workspaces

## Product contract

One email/password login authenticates an **Account** and its attached **Person**.
It does not authenticate an Organization as a second, separate login.

`GET /auth/workspaces` with the existing cookie or `Authorization: Bearer`
returns `{"workspaces":[{"organization_id":1,"organization_name":"...","membership_id":2,"joined_at":"..."}]}`.
No membership returns HTTP 200 with `{"workspaces":[]}`. Invalid/expired
sessions return 401; backend lookup failure returns 503. The caller cannot
provide an account or person ID; these come from authenticated session state.

A workspace is listed only if:
- account is open, membership belongs to the account's person, joined and not ended;
- organization is not archived, membership is not currently blocked;
- at least one current role assignment with a permission exists.

There is no workspace switching token and no tenant identity cached in JWT.
UI selection merely chooses the Organization ID in subsequent routes. **Every
action** continues to verify live tenant authorization and resource ownership
inside the responsible domain SQL. The workspace list is a navigational
snapshot, not a permission grant. It may be stale by the time a user clicks it.
It does not list pending invitations or unapproved registration requests.

## Why not SMS MFA in this PR

Existing `/auth/phone/start` and `/auth/phone/verify` require an already
authenticated account and only establish phone possession. Using them after
password entry, while a full session already exists, would not provide MFA.
A safe optional SMS second-factor login needs pending authentication,
verified explicit enrollment, time-limited/attempt-limited challenges,
device/session promotion only after confirmation, generic error behavior,
and account recovery. It must not be bootstrapped by the existence of a
verified phone. SMS is not a phishing-resistant authenticator.

## Gate

No schema migrations, no roles in access JWT, no AWS. GitHub CI must pass
SQLC regeneration consistency, Go/race/integration tests and Docker smoke.
Relevant security counterexamples: cross-person/cross-tenant, blocked/ended
membership, ended role, no permissions, archived organization, revoked session.
