# G1.2c.8b — Organization owner promotion, demotion and transfer

Baseline: BDSPro dev@9102ae568e31c766283a6c6a51f2273a58b8b7db
Reference: [Gitea ErrLastOrgOwner and owner team](https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/services/org/team.go).

Gitea reserves a special owner team, but BDSPro owns the `owner` role in
`organization_members`. Never insert a fabricated Gitea Owner Team.

## API contracts

- PUT /organizations/:id/owners/:person_id: promote current ordinary member
  with a verified account/contact; return 200 and updated membership
- DELETE /organizations/:id/owners/:person_id: demote existing owner to
  ordinary member; reject demoting the sole remaining active owner (409)
- POST /organizations/:id/owners/transfer: body { "person_id": N },
  replace acting owner's role with an eligible member's owner role atomically;
  return 200; the actor keeps ordinary org membership and existing team grants

Only a signed, session-live, currently active owner may perform these.
Owner role is never removed via ordinary member DELETE. The recipient
must be a current org member with a verified email or bound verified phone
account; unassociated Person IDs are not eligible.

## Lock order & race graph

1. In the service-owned PostgreSQL transaction lock the acting session
   FOR UPDATE; session revoke must serialize with role changes.
2. Lock organizations row FOR NO KEY UPDATE. This serializes all role
   changes and ordinary member removal for this organization. Unlike
   FOR UPDATE, it remains compatible with FK KEY SHARE on organization
   rows when teams/members are inserted.
3. Lock the acting owner membership FOR SHARE and resolve its PersonID.
   This is compatible with other owner-read transactions while blocking
   concurrent owner role changes.
4. Promotion checks eligible target Account with FOR KEY SHARE, then
   updates the member role. Demotion counts active owners under the
   per-org lock before updating; count<=1 is a 409.
5. Transfer demotes acting owner FIRST then promotes the recipient.
   This is intentional: team mutations acquire owner membership before
   recipient membership. Reversing the two writes invites deadlock with
   team member insertion. If promotion fails, the donor demotion is
   rolled back. No other transaction can see an intermediate ownerless
   state under READ COMMITTED.
6. Write the concrete action `organization.owner.promoted`,
   `organization.owner.demoted`, or `organization.owner.transferred`
   in the same transaction. Migration v30 extends audit constraints
   without overwriting v29.

Existing `CreateOrganization` remains account-first then session-first
to avoid deadlock with password reset. Owner changes never take an
acting-account lock *after* the session lock.

## Finding from the PostgreSQL race suite

An early design used two independent SQL `EXISTS` predicates to lock owner
and recipient memberships during team user insertion. PostgreSQL is free to
reorder independent predicates. A deliberately staged transfer and team
write reproduced deadlock `40P01`, even though the SQL text listed owner
first. This invalidated that assumed lock order.

`AddTeamUserForOwner` and `CreateTeamInviteForOwner` now use MATERIALIZED
CTEs where `recipient_scope` depends on `owner_scope`. The dependent
recipient row lock cannot begin before the owner authorization CTE has
produced its rows. The transfer likewise demotes donor before promoting
recipient; both paths obey owner → recipient, without globally serializing
all unrelated team writes via an organization lock. Retain the race tests
as a regression barrier. PostgreSQL 18 row locks:
https://www.postgresql.org/docs/18/explicit-locking.html

## Proving the invariants

Require a disposable PostgreSQL 18 with pinned connections, lock evidence
via pg_blocking_pids, and tests of:
- dual concurrent demotions of distinct owners: never zero owners
- self-demotion with two owners vs sole-owner rejection
- transfer to missing, unverified or ended member rolls back donor
- direct team insertion vs transfer: no deadlock, correct order
- ordinary member removal vs transfer in both commit orders: role change
  serializes on the organization row
- acting session revocation before/after role transaction
- audit failure rolls back all role changes
- verified email and verified phone eligible, and owner/outsider denied
- HTTP, canonical migration, SQLC, Gin route inventory, Docker, Asynq CI

The BDSPro database does not constrain the **owner-count** invariant for
out-of-band direct SQL writers. It is a service transaction policy;
direct SQLC demotion can bypass owner count and must be prohibited outside
the lifecycle service. G1.2c.9 adds a narrower database guarantee: a live
owner cannot exist without an Account attached to the same Person, and
account deletion is blocked while its owner membership is active. This is
independent of account login/credential availability and does not prove
that a verified or functioning contact method remains accessible. Do not
claim global owner-count invariance or measured p95/p99 without further
contracts and test evidence.
