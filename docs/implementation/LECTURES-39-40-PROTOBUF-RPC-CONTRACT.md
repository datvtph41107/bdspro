# Lecture #39–40 — Phân tách Protobuf theo Simple Bank, giữ identity canonical của BDSPro

**Mốc:** 2026-10-21. **Phạm vi:** chuẩn hoá Protobuf, message/codegen và đặc tả gRPC-Gateway Swagger v2; không thêm gRPC listener, không AWS/deployment.

## 1. So sánh trách nhiệm với TechSchool Simple Bank

Simple Bank chia `service_simple_bank.proto` (service + HTTP/Swagger options),
`rpc_create_user.proto`, `rpc_login_user.proto` (từng request/response),
`user.proto` (message dùng lại), và vendor các file
`google/api`, `protoc-gen-openapiv2/options`. BDSPro vận dụng **nguyên tắc
phân tách**, không sao chép nghiệp vụ ngân hàng.

```text
proto/
  bdspro/v1/
    service_bdspro.proto             # RPC service và google.api.http / OpenAPI v2 options
    rpc_get_current_account.proto   # message request/response của Account
    rpc_list_workspaces.proto       # message request/response Workspace
    user.proto                      # User là read projection của Account, không phải bảng user
    workspace.proto                 # read projection membership
  google/api/                       # vendored third-party annotations
  protoc-gen-openapiv2/options/     # vendored third-party Swagger v2 options
buf.yaml                            # STANDARD lint, vendor miễn lint project rules
buf.gen.yaml                        # pinned source paths, Go + gRPC stubs + Swagger v2
pb/bdspro/v1/                       # code sinh từ *.proto, không sửa tay
docs/grpc-swagger/bdspro.swagger.json  # generated RPC-only Swagger 2.0
api/openapi.json                    # Gin HTTP REST OpenAPI 3.1 (giữ nguyên)
```

**Vì sao không để package proto ở root như Simple Bank?** BDSPro đã công bố
namespace `bdspro.v1` và Go import `pb/bdspro/v1`. Versioned proto giúp
quản lý breaking changes tốt hơn; phải đánh đổi một cấp thư mục, không tạo
thêm lớp logic nghiệp vụ. Giữ `account_id` ở field 1 của
`GetCurrentAccountResponse` để bảo toàn wire compatibility. `user.proto`
chỉ đưa ra **đối tượng trình bày** với `account_id`; không tạo bảng
`users` và không thay Party/Person/Account.

## 2. HTTP annotation có làm Gin tự động chuyển sang gRPC không?

**Không.** `google.api.http` trên service tạo mapping **dành cho future
gRPC-Gateway** (`/v1/accounts/me` và `/v1/accounts/me/workspaces`).
Chúng chưa phải đường dẫn sống và không dùng lại `/auth/me` hiện có của
Gin một cách vô hình. `gapi` ở lecture sau sẽ gọi cùng Auth/Organization
service, xác thực JWT/session và quyền trên PostgreSQL. Client không gửi
account ID vào các request RPC; actor phải lấy từ metadata JWT và session
còn sống trong DB.

## 3. Swagger v2 và Swagger UI hiện có là hai nguồn khác nhau

- `api/openapi.json`: **OpenAPI 3.1, cho HTTP Gin đang chạy**, được phục vụ
  tại `/docs/openapi.json` trên development.
- `docs/grpc-swagger/bdspro.swagger.json`: **Swagger/OpenAPI 2.0,
  generator output cho RPC-Gateway tương lai**. File này không phải
  tài liệu của toàn bộ Gin routes và không được sử dụng thay spec hiện có.
- `protoc-gen-openapiv2` với `allow_merge=true,merge_file_name=bdspro` là
  cách gộp nhiều proto khi số RPC tăng. Dùng `strategy: all` theo tài liệu
  grpc-gateway. `google/api/annotations.proto` định nghĩa HTTP mapping,
  còn `protoc-gen-openapiv2/options/annotations.proto` định nghĩa metadata
  Swagger; hai chức năng độc lập.
- Nếu cần một contract hợp nhất cho cùng route trong tương lai, phải chủ
  động chốt ownership và viết parity tests. Chưa tự động hợp nhất hai file.

## 4. Quy trình vận hành local

```bash
# Đảm bảo PATH có Go bin: export PATH="$(go env GOPATH)/bin:$PATH"
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.10
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.27.8

buf lint
buf build
make proto
git diff --exit-code -- pb docs/grpc-swagger
go test ./...
```

CI kiểm tra source generated code và schema, vẫn chạy các migrations,
SQLC, Go tests, race, Gin/Compose smoke. Nếu CI không PASS thì chưa merge.

**Vendor provenance:** Các file `proto/google/api/*` (Google APIs,
Apache-2.0) và `proto/protoc-gen-openapiv2/options/*` (gRPC-Gateway,
BSD-3-Clause) đã được đối chiếu từ TechSchool Simple Bank; không chỉnh sửa
nội dung cấu trúc của bên thứ ba. Nên nâng cấp version vendor có kiểm thử
compatibility và bằng chứng build, không chỉnh sửa trực tiếp.

**Giới hạn:** Protobuf/gRPC codegen và Swagger v2 **không bằng chứng gRPC
server đã chạy**. Listener, TLS/local trust, authentication interceptors,
error mapping, DB session checking, gRPC-Gateway runtime và end-to-end
parity tests thuộc Lecture #41–45.
