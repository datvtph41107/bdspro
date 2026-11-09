# First-party session verification

**User path:** `POST /users` → `POST /users/login` → `GET /users/me` → `POST /tokens/revoke` → previous credentials rejected.

**Route and admission:** [Gin route definitions](../../../../../routers/api/v1/api.go); [JWT and session middleware](../../../../../services/context/session.go); [actor model](../../../../../models/auth/actor_context.go).

**State/authority:** [session domain](../../../../../models/auth/session.go), [SQLC session query](../../../../../db/query/session.sql), [live-session concurrency contract](../../../../../docs/architecture/g1-live-session-authority.md).

**Executable test:** [TestG12LiveSessionRevokeDeniesAccessAndRefreshHTTP](../../../../../routers/g1_session_authority_e2e_test.go). It verifies blocked session persistence and denial of both earlier access and refresh credentials after committed revocation.

**Important limit:** A request admitted *before* revocation can finish afterward; the HTTP gate alone is not a transaction lock. The [in-flight acceptance](../../../../../routers/g1_live_session_acceptance_test.go) covers this nuance separately and is not included in this minimal lane.
