# DRX-to-Go parity: Account access history (R32)

References: Laravel DRX main 767737d3a9530d0ec3ff94876752e40ccb302a15;
BDSPro dev f5c3f9bf7d0d161d3cc602c91f1da8a3b407a0f7.

## Domain translation

| Laravel DRX | BDSPro | Owner |
| --- | --- | --- |
| User | Account | models/account |
| User profile | Person / PersonProfile | models/person and models/db |
| Shared domain identity | Party | models/db |
| UserConnectionHistory | AccountConnectionHistory | models/account/account.go |
| UserDeviceRegister | AccountDeviceRegister (future protocol) | models/account/device_register.go (not created) |
| UserSession | Existing Session | models/auth/session.go |
| UserService::getUserAccessHistories | AuthService.GetConnections | services/auth/auth.go |
| DeviceService::storeConnection | auth.NewSession | models/auth/session.go |
| GET connections | GET /users/me/connections | routers/api/v1/user/auth.go |
| IPActive::checkTimeActiveLatest | Existing Redis IPActive.Check | modules/ipactive |

The HTTP /users/me namespace is an existing contract. Account owns auth;
Person owns human profile; Party represents cross-domain identity, not auth.
Do not silently rename public paths as part of a persistence migration.

## Implemented invariants

* One committed refresh session insertion creates exactly one login history
  event in the same PostgreSQL transaction. Tokens are not released on failure.
* Events contain account, session correlation ID, action, IP, UA and timestamp;
  no bearer secrets.
* Session cleanup does not erase history: session_id deliberately has no
  cascading foreign key because sessions may be deleted on password reset.
* SQL queries scope reads by the trusted account_id; clients cannot override
  it via query parameters. List is capped at the latest 50 events.
* This is a successful-login history, not all HTTP requests, online presence,
  or a substitute for IPActive admission.
* Raw IP and User-Agent require restricted access, logs and exports.
* **Production blocker:** define retention, periodic deletion, and DSAR /
  backup procedures. This tranche has no automatic history expiration.

## Why not copy DeviceService verbatim?

DRX identifies devices from User-Agent-derived kind/client/OS. These values
are spoofable and may collide; its post-login middleware can check devices
after Passport created credentials. Do not represent this as device proof.
Next gated tranche must specify an opaque installation secret stored hashed,
new-device verification before release of usable credentials, session/device
revocation semantics, and failure recovery. No unused speculative device table.

## Acceptance

* Disposable PostgreSQL: successful atomic write, account isolation,
  second-write rejection rolling back session, history surviving cleanup.
* HTTP: bearer required, verified account scoping, safe driver errors.
* SQLC v1.30.0 / dbalias reproducibility; OpenAPI / route inventory.
* Access JWT revocation remains stateless until normal expiration; full
  immediate revocation is a separate protocol change.

## Integration ownership (R34)

Merged on top of PR #140 AuthService convergence in an independent review branch. History read is owned in the existing `models/account/account.go` / `services/auth/auth.go` / `routers/api/v1/user/auth.go`, not a one-function file per layer. Transactional session-history insertion is owned by `models/auth.NewSession`, which uses `db.WithTxResult` and transaction-scoped SQLC queries. `models/db` owns only the transaction mechanism and generated SQL; the disposable PostgreSQL rollback/isolation test now lives alongside the auth model. This is not proof of traffic capacity or a device identity protocol.
