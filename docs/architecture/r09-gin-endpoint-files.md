# R09 — Split Gin endpoint source files by responsibility

Reference: Gitea commit `12b20c7`, `routers/api/v1/user/*` and `routers/api/v1/org/{org.go,member.go}`.

This refactor **preserves Gin, all handler signatures, SQLC, API URLs, input types, response behavior and route registration**. It moves existing functions unchanged into focused files within their current Go packages.

- `user/auth.go`: handler constructor, registration, login and `Me`.
- `user/verify_email.go`: verification resend, HTML page and token consumption.
- `user/token.go`: access renewal, revocation and session endpoints.
- `user/password.go`: password reset request, HTML page and confirmation.
- `user/email.go`: email linking endpoints.
- `org/org.go`: organization CRUD, constructor and shared path parser.
- `org/member.go`: membership listing and adding.

No adapter, repository, custom router, new middleware, ORM or new public URL prefix is introduced. Review relies on Go 1.27.3 CI, unchanged 34-route inventory, OpenAPI, PostgreSQL, Docker and HTTP smoke.
