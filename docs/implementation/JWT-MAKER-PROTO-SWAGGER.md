# BDSPro — JWT Maker, Protobuf/gRPC và Swagger: ranh giới kỹ thuật

**Ngày:** 2026-10-21. **Phạm vi PR này:** đổi tên và tách package JWT, giữ nguyên API HTTP và không thêm gRPC/Swagger giả. Không cloud/deploy.

## 1. Vì sao chuyển `auth/access_jwt.go` → `token/jwt_maker.go`?

- `auth` là nguồn sự thật Authentication nghiệp vụ: mật khẩu, OTP, Account,
  Session trong PostgreSQL. Ký và kiểm JWT là thao tác crypto/transport độc lập.
- `token.JWTMaker` và `token.NewJWTMaker` diễn đạt đúng trách nhiệm hơn
  `auth.AccessTokens` / `NewAccessTokens`.
- `api.Server.EnableJWTMaker` là điểm wiring; không tạo interface toàn cục
  `Maker` khi mới một implementation và chưa có nhu cầu mocking thay thế.
- Thời hạn JWT **15 phút**, HS256, issuer `bdspro-auth`, audience
  `bdspro-api`, `token_type: "access"`, subject Account ID và fingerprint
  session `sid` không đổi. Mỗi HTTP bearer vẫn tra session thật để bảo toàn
  khả năng revoke ngay; role/org permissions tiếp tục lấy từ SQL.
- Không loại từ `access` trong tên JSON public (`access_token`,
  `access_token_expires_at`), token claim hoặc authorization code vì
  chúng là ngữ nghĩa API: loại Bearer token khác refresh/recovery token.
- Giữ `Issue(accountID, sessionToken)`/`Verify(raw)` vì biểu đạt đủ rõ;
  không sao chép chữ ký `CreateToken(username, role,...)` của Simple Bank,
  vốn không phù hợp nguồn sự thật về quyền của BDSPro.

## 2. Đổi biến cấu hình, lưu ý khi pull local

- Tên chuẩn mới: `TOKEN_SYMMETRIC_KEY` (>=32 bytes), trong `app.env` /
  `util.Config.TokenSymmetricKey` / `main.go`.
- Thay cho `ACCESS_TOKEN_KEY` cũ. **Không có alias tự động**. Hãy đổi key
  name trong các env override/local configurations trước khi chạy.
- Giữ nguyên **giá trị secret** nếu muốn các JWT đã phát vẫn kiểm được.
  Đổi giá trị là rotation và làm JWT cũ không hợp lệ; cookie/DB session
  không tự đổi chỉ vì đổi tên env. Production config fail-closed nếu thiếu
  signing key. Dev default chỉ sử dụng trên loopback.
- HTTP contract được giữ nguyên: response login/renew vẫn chứa
  `access_token`, `token_type: "Bearer"`,
  `access_token_expires_at` và cookie session; không bắt client đổi field.

## 3. Protobuf và gRPC: vấn đề được giải quyết

Gin đang nhận HTTP/JSON và gọi Auth/Organization/Source Services. Đây vẫn là
một kiến trúc hợp lệ, không cần gRPC để sử dụng API.

Khi cần hợp đồng API **typed**, sinh clients nhiều ngôn ngữ, stream, hoặc
dịch vụ-to-dịch vụ RPC, ta định nghĩa `proto/*.proto`. `protoc` hoặc `buf`
sinh `pb/*.pb.go`; `gapi` triển khai handler gRPC. gRPC thường dùng HTTP/2;
Protobuf là một serialization/schema language, không phải HTTP server.

```text
HTTP/JSON client -> api/ (Gin) -----+
                                    |-> auth/organization/source -> sqlc -> PostgreSQL
RPC client -------> gapi/ (gRPC) ---+
                     ^
                     |
                pb/ (generated from proto/)
```

Không xây logic Auth/permission thứ hai trong `gapi`. Gin, gRPC và gRPC
Gateway phải cùng check Session/Account và quyền SQL; một `.proto` không tự
tạo/migrate mọi route đang tồn tại trong Gin.

## 4. Swagger/OpenAPI khác gì Protobuf?

- **OpenAPI** là đặc tả HTTP API: route, method, JSON schemas, security,
  response/error code, examples.
- **Swagger UI** là giao diện tương tác đọc một bản OpenAPI spec. Không cần
  Protobuf hoặc gRPC để dùng với Gin.
- Nếu sau này dùng `gRPC-Gateway` (#43–45), một số OpenAPI có thể được sinh
  từ `proto` + HTTP annotations. Phải tránh hai nguồn định nghĩa sai khác
  cho cùng endpoint.
- Chốt thứ tự: (a) mô tả/cấp phép API Gin hiện có bằng OpenAPI;
  (b) serve Swagger UI local-only và contract tests;
  (c) khi đã xây gRPC, map một tập RPC có chủ đích vào Gateway và kiểm tra
  parity; **không thay Gin để thêm Swagger**.

**Ở PR rename JWT này Swagger UI/chức năng gRPC vẫn CHƯA triển khai.**
G03 chịu trách nhiệm OpenAPI của HTTP Gin; G06–G07 chịu trách nhiệm proto,
generated pb, gapi và Gateway. Không được đánh dấu PASS chỉ bằng tài liệu.

## 5. Acceptance

- Xóa file JWT cũ, tạo package `token` với unit tests cùng assertions.
- Chuyển wiring Gin/main và các tests về `token.JWTMaker`.
- CI PASS: Go tests/race, migrations UP-DOWN-UP, sqlc generate diff sạch,
  gofmt/vet, startup, Docker Compose.
- Không đổi JWT bytes/claims semantics, không đổi session/token wire JSON.
- Không thay cấu trúc Auth DB, không thêm queue/provider/cloud.
