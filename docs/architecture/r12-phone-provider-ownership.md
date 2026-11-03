# R12 — Notification provider selection is not HTTP routing

Reference: Gitea `routers/init.go`, `services/mailer`, `modules/setting` at commit `12b20c7`.

Prior to R12, `routers/api/server.go` chose development-local, SMS and Zalo transports and validated their configuration. This is notification-provider ownership, not route registration.

`services/notification.NewPhoneCodeSender(cfg)` now owns the **same** decision, validation strings and transport timeouts (8 seconds), while `routers/api.NewServer` consumes the existing `PhoneCodeSender` interface and remains responsible for building Gin phone handlers until HTTP composition is fully migrated.

No change to phone OTP algorithm, Redis keys, persisted challenges, HTTP APIs, Docker configuration or SQLC. The development-only Local sender remains enabled solely in development. Missing production sender still removes phone routes as before.

Factory tests prove local/disabled/SMS/Zalo/invalid configuration behavior. The final integration gate is CI on Go 1.27.3, PostgreSQL, Redis, Gin API route matrix and Docker/HTTP smoke.

Subsequent PR: move HTTP application composition from `routers/api/server.go` to `routers/init.go` without introducing custom Gin wrappers or a second router.
