# Organization and invitation verification

**User path:** authenticated, eligible account creates `POST /orgs`; owner manages `/orgs/:id/members` and `/orgs/:id/teams`. Only a verified authorized actor may issue a team invitation. Replay must fail.

**Route and request authority:** [Gin route definitions](../../../../routers/api/v1/api.go), [org actions](../../../../routers/api/v1/org/org.go) and [resource assignment](../../../../routers/api/v1/organization_context.go).

**Business and persistence:** [organization service](../../../../services/organization/org.go), [team service](../../../../services/organization/team.go), [organization queries](../../../../db/query/organization.sql), [team queries](../../../../db/query/team.sql).

**Executable tests:**
- [Organization access HTTP](../../../../routers/organization_access_e2e_test.go): owner/member/outsider results, persisted membership and revoked visibility.
- [Team invitation HTTP](../../../../routers/team_invite_e2e_test.go): create team, issue one invitation, inspect job outbox, reject unauthorized redemption and replay.

**Important limit:** Existing acceptance tests do not prove every concurrency schedule, a strictly database-enforced last-owner count, or the runtime of an external email provider. Review [owner integrity](../../../../docs/architecture/g1-owner-account-integrity.md) and [doctor limits](../../../../docs/architecture/g1-owner-coverage-doctor.md) before making these claims.
