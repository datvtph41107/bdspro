# Organization API — canonical `/orgs`

`/organizations` has been intentionally retired; this is a breaking API path change. The Account–Person–Organization permissions, SQLC queries, PostgreSQL schema and JSON fields remain unchanged.

| Method | Endpoint | Access |
|---|---|---|
| GET | `/orgs` | verified_contact |
| POST | `/orgs` | verified_contact |
| GET | `/orgs/:id` | verified_contact |
| PATCH | `/orgs/:id` | verified_contact |
| GET | `/orgs/:id/members` | verified_contact |
| POST | `/orgs/:id/members` | verified_contact |
| DELETE | `/orgs/:id/members/:person_id` | authenticated_verified_contact_owner |
| DELETE | `/orgs/:id/owners/:person_id` | authenticated_verified_contact_owner |
| PUT | `/orgs/:id/owners/:person_id` | authenticated_verified_contact_owner |
| POST | `/orgs/:id/owners/transfer` | authenticated_verified_contact_owner |
| GET | `/orgs/:id/teams` | verified_contact |
| POST | `/orgs/:id/teams` | verified_contact |

All Organization and Team routes remain protected by existing authenticated-session and verified-contact middleware. Owner rules are ultimately enforced by the existing transactional SQLC predicates; changing the route path does not replace them.
