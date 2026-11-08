# G1.2c.8a — Ordinary organization membership removal

Source baseline BDSPro dev@53d0048041e37687b65c4f9fb0d4899037e7b5a7.

Gitea reference:
- https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/services/org/team.go
- https://github.com/go-gitea/gitea/blob/edebfaa12516a140d7ee3fcbaf25a6a1cc2cdb76/routers/api/v1/org/member.go

## Domain difference

Gitea has a reserved owner team and an ErrLastOrgOwner condition.
BDSPro does **not** have a Gitea-style owner team. Owners are active
`organization_members(role='owner')` and ordinary `team_members` are
independent team grants. Copying Gitea's owner team structure would
misrepresent BDSPro's schema.

The new DELETE /organizations/:id/members/:person_id endpoint revokes
only a regular member. The model enforces `target.role='member'` in the
owner-scoped SQLC UPDATE, making every owner (including the last one)
unremovable from this endpoint. A separate role-management contract is
required for promotion, demotion and ownership transfer; it must
serialize owner-count changes and protect the last owner.

## Lifecycle and invariants

Router: parse path and signed actor, return 204 or public error.
Service: one commit boundary contains acting-session FOR UPDATE, scoped
membership close, dependent team-link deletion and audit.
Model/SQLC: owner predicate uses `FOR SHARE OF owner`; UPDATE locks
recipient membership. Team mutations and invitation issuance use
`FOR SHARE` of recipient membership. Either they commit before removal
and their team grants are cleared, or they fail after removal.
The row is soft-ended, while team_member records are physically
deleted: rejoining does not restore past team privileges.

Invitation redemption and worker delivery require
`invitation.created_at >= active_membership.joined_at`. A token
issued in an earlier membership generation can never be reused after
leaving and rejoining. `RevokeExpiredTeamInvites` also closes
previous-generation open invitations on next issuance, releasing the
unique-open-invite slot. The removal deliberately does not lock invite
rows after holding membership locks: acceptance locks invitation then
takes recipient membership FOR SHARE; reversing those locks risks a
cycle. Issuance and acceptance remain transaction-bound.

Audit `organization.member.removed` is committed or rolled back with
the member and team changes. Migration 29 only extends the audit
constraint and raises canonical-version acceptance. Older migrations
are immutable. HTTP contracts and the audit route inventory are
updated explicitly; no generic middleware or new event bus.

## Acceptance

Test PostgreSQL on bdspro_test: positive removal; denied owner,
self/last-owner, missing or repeated removal, foreign actor,
expired/blocked acting session; team grants gone after rejoin;
old invitation invalid after rejoin; new invitation possible;
audit failure rolls back; concurrent revocation with team writes
and token acceptance; correct HTTP status; SQLC generation, Go race,
migration, Asynq/Docker/HTTP smoke CI.

Costs: membership UPDATE, team grant DELETE, and audit insert under
one transaction and session lock. No p95/p99 claim.

Owner promotion/transfer remains deliberately outside this slice.
