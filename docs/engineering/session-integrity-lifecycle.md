# ENG-CORE-08 — Shared Session Integrity & Lifecycle

**Source baseline:** BDSPro `dev@b1f91885cbb6645b3b982d91abe6407ed5fa396e` (25/10/2026). Kế thừa [ENG-CORE-07](dependency-module-ownership.md), [ENG-CORE-04](application-foundation.md) và [Development Contract](development-contract.md).

**Purpose:** kiểm chứng hợp đồng `IssueSession` đã được ba phương thức Password/Google/Phone OTP dùng lại, cùng vòng đời Refresh → Revoke và failure semantics của OTP. Đây là giai đoạn **harden existing contract**, không tạo `SessionService`, DI provider, session cache, schema hay API mới.

## 1. Source-backed operation map

| Entrypoint | Identity proof | Session issuing owner | Boundary |
| --- | --- | --- | --- |
| Password `POST /users/login` | `auth.Login`: account + password hash | `auth.IssueSession` | Gin handler sở hữu JSON/status |
| Google OAuth callback (login) | state, PKCE, ID token, Google-subject → account lookup | `auth.IssueSession` | HTTP/OIDC callback; không gộp chứng thực với Password |
| Phone OTP verify (development) | `PhoneOTP.VerifyLogin` ConsumeChallenge rồi GetAccount | `auth.IssueSession` | OTP consumption theo chính sách hiện hành, chưa có atomic session handoff |
| Refresh `POST /tokens/renew_access` | signed refresh JWT + matching active PG session | `token.Maker.CreateToken` access only | Không tạo thêm refresh session |
| Revoke `POST /tokens/revoke` | signed refresh JWT + matching active PG session | `db.Store.BlockSession` | DB làm nguồn sự thật cho revoke |

## 2. Invariants có chủ sở hữu

**`auth.IssueSession`:** token access/refresh được phát hành đúng purpose và AccountID; refresh payload ID khớp `sessions.id`; refresh token được lưu cùng AccountID, UserAgent, ClientIP và expiry. Chỉ khi `CreateSession` thành công mới trả `LoginResult` chứa tokens. Khi ký bất kỳ token nào lỗi, không được ghi Session. Khi ghi Session lỗi, không được trả token ra caller như một lần login thành công.

**Refresh:** kiểm tra signature/purpose của refresh JWT và đối chiếu `session.id/account_id/refresh_token/is_blocked/expires_at` từ DB; response chỉ phát access token. `pgx.ErrNoRows` hay session revoked -> HTTP 401; lỗi hạ tầng không lộ chi tiết -> HTTP 500.

**Revoke:** PostgreSQL `BlockSession` kiểm tra ID/AccountID/token/block/expiry trong một UPDATE; lần revoke đầu thành công HTTP 204, gọi lại với cùng token trả 401. Revoke không làm access JWT đã cấp hết hạn ngay; các access JWT còn hợp lệ theo expiry và signature hiện tại. Nếu muốn revoke access tức thời là một thay đổi policy/architecture riêng.

**Google context:** `api/google.go` lấy account từ Store bằng `ctx.Request.Context()` như các lời gọi I/O cùng request; việc đổi này không xác minh OIDC provider E2E.

**Phone OTP:** `ConsumeChallenge` xảy ra **trước** `GetAccountByID` và `IssueSession`. Nếu `CreateSession` lỗi, OTP đã được tiêu thụ, không có `LoginResult` thành công, và phải xin challenge mới. Đây là **hành vi hiện hành được kiểm chứng bằng fake one-use challenge**, chưa phải lựa chọn tối ưu đã được phê duyệt. Không tự đưa code vào trạng thái có thể replay để “rollback OTP”. Phương án atomic consumption+session transaction chỉ được đánh giá khi chốt yêu cầu user experience, replay protection, lock/retry và data owner.

## 3. Source changes + evidence

| File | Phạm vi |
| --- | --- |
| `api/google.go` | Production change duy nhất: dùng `ctx.Request.Context()` trong GetAccountByID của Google callback |
| `auth/session_contract_test.go` | IssueSession success, purpose mismatch, ID/expiry/UA/IP, token maker failure, DB failure |
| `auth/phone_otp_session_failure_test.go` | OTP consume rồi CreateSession lỗi; không trả session, replay bị từ chối (fake repository) |
| `api/session_revoke_contract_test.go` | Revoke response: success/PG no rows/infrastructure/wrong-token, context propagation |
| `api/session_lifecycle_integration_test.go` | CI disposable PG: IssueSession → Renew → Revoke → blocked refresh, access JWT vẫn hợp lệ |

**Runtime scope:** Integration test chỉ chạy trên GitHub Actions và `BUSINESS_TEST_DB_URL` trỏ `127.0.0.1:5432/bdspro`. Sử dụng bản ghi Account/Session ngẫu nhiên, không xóa hay reset DB/volume. Ghi chú độ chính xác: PG timestamptz lưu microseconds, Go `time.Time` có nanoseconds; so sánh persisted expiry theo tolerance microsecond, không so sánh bytes thời gian.

**Existing evidence:** các test `api/token_test.go` kiểm chứng Refresh Token invalid purpose, missing/blocked/wrong account/wrong token/expired session/DB error; `db/sqlc/phone_otp_test.go` kiểm chứng OTP consumed single-use, failed attempts và concurrency; token package kiểm chứng JWT maker.

## 4. Acceptance & non-goals

- **RUN/PASS:** test mới, API/token tests hiện có, BIZ Organization và Register handoff test, Go build, migration guard, Compose/runtime smoke, go vet, race detector.
- **Không thay đổi:** token claims, JWT secret/TTL, HTTP endpoints/status/body, Session/OTP SQL, schema, migrations, policy sản phẩm, code generator, worker.
- **Không kết luận:** Google provider E2E, phone SMS delivery, WSL acceptance, production recovery hoặc immediate access-token revocation.
- **Decision:** giữ một `IssueSession` duy nhất cho các login methods. Tách credential proof khỏi session issuance, giữ Refresh/Revoke theo database contract đã có; nghiên cứu OTP atomicity chỉ khi có yêu cầu đúng/sai nghiệp vụ rõ.
