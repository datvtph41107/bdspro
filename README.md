> **Engineering Core acceptance (2026-10-17):** [Báo cáo nghiệm thu, các thay đổi và giới hạn](docs/implementation/ENGINEERING-CORE-ACCEPTANCE-2026-10-17.md) · [Quickstart hiện hành](docs/implementation/LOCAL-QUICKSTART.md). HTTP đã có `POST /auth/resend-verification` để khôi phục sau lỗi gửi email. Đây là baseline cho Lecture #13, chưa là production-ready.\n\n> **Transaction convention:** `db/sqlc/Store.ExecTx` là đường đi duy nhất cho các transaction nghiệp vụ hiện tại trong Auth, Organization, Source và `RegisterTx`. Helper chung quản lý Begin/Commit/Rollback; service callback sử dụng SQLC Queries. [Chi tiết và giới hạn](docs/implementation/TRANSACTIONS-LECTURES-06-10.md).\n\n> **Bootstrap chuẩn Lecture #12:** [`main.go`](main.go) chỉ nạp Viper config → PostgreSQL pool → Store/Services → Gin `Start`. Validate environment nằm tại [`util/config.go`](util/config.go); graceful shutdown nằm tại [`api/server.go`](api/server.go). [Xem giải thích và lệnh nghiệm thu](docs/implementation/MAIN-BOOTSTRAP-LECTURE-12.md). Bản này chưa triển khai Lecture #13.

> **Gin HTTP layer:** `api/` nay quản lý các endpoint Auth, Organization, Source và health, không còn `net/http` handlers rời trong từng domain. Xem [GIN-HTTP-MIGRATION.md](docs/implementation/GIN-HTTP-MIGRATION.md); business services và database không đổi. Các sơ đồ/mô tả handler trong checkpoint cũ là lịch sử.

> **Lecture #12 — Viper:** BDSPro hiện dùng [`app.env`](app.env) + [`util/config.go`](util/config.go) theo SimpleBank. Chạy `make server` từ root (config mặc định cho PostgreSQL Docker local), ghi đè bằng `DB_SOURCE`/`SERVER_ADDRESS`/`ENVIRONMENT` khi cần. Xem [hướng dẫn cấu hình](docs/implementation/CONFIG-LECTURE-12.md). Các hướng dẫn env cũ trong tài liệu checkpoint mang tính lịch sử.

> **Canonical working tree before Lecture Gin:** Migrations: `db/migration/`; CI: `.github/workflows/ci.yml`; commands: `make migrateup DB_URL=...`, `make sqlc`, `make test`. Historical checkpoints below may retain old paths.

> **TRỌNG TÂM REVIEW HIỆN TẠI 2026-10-15:** Xây dựng **Engineering Core Operating Model** (chưa triển khai Business Core tiếp theo): [checkpoint](docs/continuity/checkpoints/2026-10-15-engineering-core-operating-model/README.md) · [ma trận 20 chương](docs/continuity/checkpoints/2026-10-15-engineering-core-operating-model/CHAPTER-MATRIX.md) · [CH01 config/workflow](docs/continuity/checkpoints/2026-10-15-engineering-core-operating-model/CH01-CONFIG-AND-WORKFLOW.md) · [chat handoff prompt](docs/continuity/checkpoints/2026-10-15-engineering-core-operating-model/CONTINUE-PROMPT.md). Các mục "Bước kế tiếp CORE-06" bên dưới là lịch sử sản phẩm, **không phải task đang làm**. Git live là authority cho source; checkpoint mới là authority cho review focus. Không thay đổi runtime/source trong checkpoint này.

# BDSPro — Nền tảng Core từ CORE-01 đến CORE-05

> **CORE-01 bổ sung 12/10/2026:** migration `000006_party_kind_immutability` khóa không cho đổi loại Party; hai test mới về chuyển loại và cạnh tranh cập nhật. [CI 38221355862 thành công](https://github.com/datvtph41107/bdspro/actions/runs/38221355862); [quyết định và giới hạn](docs/implementation/CORE-01-KIND-IMMUTABILITY.md). Không sửa lịch sử migration `000001`.
>
> **Ngôn ngữ tài liệu đang hoạt động: tiếng Việt.** Tên bảng, cột, API, câu lệnh và mã nguồn giữ nguyên tiếng Anh để đối chiếu trực tiếp với PostgreSQL/Go.  
> **Nhánh triển khai duy nhất:** `dev`. **Phạm vi đã hoàn thành:** baseline kỹ thuật Core, **không phải** toàn bộ sản phẩm BDSPro hay bản phát hành production.

## 1. Chúng ta đang xây dựng gì?

BDSPro xây dựng nền tảng lưu giữ sự thật nghiệp vụ lâu dài (**Write Core**) và các cách đọc thuận tiện cho ứng dụng (**Read Convenience**). Nền tảng được thiết kế để có thể tái sử dụng các nguyên tắc nhận dạng, tài khoản và phân quyền ở nhiều lĩnh vực; quản lý nguồn bất động sản là tình huống nghiệp vụ đầu tiên dùng Core.

Luồng đã triển khai và kiểm thử:

```text
Một người đăng ký tài khoản
  → xác minh email, đăng nhập, nhận Session
  → yêu cầu đăng ký hoặc tiếp quản tổ chức
  → reviewer nền tảng xem xét, phê duyệt theo quyền riêng
  → tạo quyền quản trị workspace và mời thành viên
  → thành viên chấp nhận lời mời và được gán Role
  → thành viên tiếp nhận nguồn bất động sản chưa xác minh
  → ghi nhận giá ban đầu, bổ sung báo giá và đọc lịch sử
```

**Ranh giới quan trọng:** một tin báo từ Zalo không biến thành `Property` đã xác minh hoặc `Listing` được phép đăng. Quyền quản trị workspace không tự động là quyền đại diện pháp lý.

## 2. Đã triển khai những module nào?

| Giai đoạn | Sự thật được lưu và bảo vệ | Migration |
| --- | --- | --- |
| CORE-01 — Identity | `parties`, `persons`, `organizations`, `person_profiles`; C01: đúng một subtype khi COMMIT | `000001_party_identity` |
| CORE-02 — Authentication | `accounts`, email, phone, `password_credentials`, email verification, `sessions`; schema nền cho passkey/đăng nhập ngoài/service account | `000002_account_authentication` |
| CORE-03 — Authorization | Membership, Role, Permission, gán quyền, khóa thành viên, lời mời và quản trị viên workspace | `000003_organization_authorization` |
| CORE-04 — Onboarding | Yêu cầu đăng ký tổ chức, claim tổ chức đã tồn tại, quyền reviewer và định danh đăng ký được xác minh | `000004_organization_onboarding` |
| CORE-05 — Source | Nguồn tin chưa xác minh, lịch sử báo giá dạng thêm mới, khóa phạm vi Organization, retry an toàn | `000005_source_intake` |
| CORE-01 follow-up — Party kind | Guard PostgreSQL chặn Person bị đổi thành Organization (và ngược lại) tại cùng Party ID | `000006_party_kind_immutability` |

**Stack thực tế:** Go 1.27.1, PostgreSQL 18, `pgx/v5`, `sqlc`, `golang-migrate`. Auth và Organization sử dụng SQL/pgx tường minh; đường dữ liệu Source sử dụng Go sinh từ sqlc.

**Chưa triển khai đầy đủ:** luồng xác thực passkey/OIDC, service-account runtime, Property/Listing mới, CRM, thanh toán/gói dịch vụ, CMS, và các bảng logical còn lại trong canonical V0.4 (164 bảng ứng viên).

## 3. Bằng chứng nghiệm thu kỹ thuật

[GitHub Actions — Core End-to-End CI đã thành công, có migration 000006](https://github.com/datvtph41107/bdspro/actions/runs/38221355862).

- PostgreSQL 18 thực thi chuỗi migration 000001–000006.
- CLI `golang-migrate` kiểm tra phiên bản, `UP → DOWN 1 → UP 1` trên database dùng riêng.
- `sqlc generate`, `go mod tidy`, `gofmt`, `go vet` và kiểm tra generated files không bị lệch.
- **17 test đạt:** 4 kiểm thử tích hợp Account → Organization → Source, 11 kiểm thử chuyên sâu Source/Price, 2 kiểm thử Party kind.
- Server khởi động thành công trên database CI.

**Không suy diễn:** CI không chứng minh máy local của bạn đã đồng bộ, dữ liệu cũ đã được migrate, doanh nghiệp đã được xác minh ngoài thực tế, hoặc hệ thống đã sẵn sàng triển khai production.

## 4. Cấu trúc nguồn chính

```text
db/migration/                  # DDL theo thứ tự 000001 → 000006
auth/                        # Đăng ký, xác minh email, password, Session, HTTP
organization/                # Review, claim, invitations, membership, HTTP
source/                      # Tiếp nhận nguồn, lịch sử giá, HTTP và tests
db/source/query/source.sql   # Các câu lệnh SQL có scope/permission
db/sourcesqlc/              # Code do sqlc sinh, đã commit
sqlc.source.yaml             # Cấu hình sinh Go
main.go                      # Khởi chạy HTTP server, gắn các module
integration_test.go          # Luồng kiểm thử xuyên module
.github/workflows/ci.yml
docs/implementation/        # Báo cáo, quy trình và nghiệm thu bằng tiếng Việt
docs/continuity/CHECKPOINT.md
```

## 5. Bắt đầu tại local mà không ghi đè source cũ

**Khuyến nghị:** trước hết lấy một bản clone **riêng**, để nguyên `~/projects/bdspro` đến khi so sánh được local với remote.

```bash
cd ~/projects
git clone --branch dev --single-branch \
  https://github.com/datvtph41107/bdspro.git bdspro-core-complete
cd bdspro-core-complete
git status --short --branch
git log -1 --oneline
```

Chỉ dùng database PostgreSQL **mới, dành cho thử nghiệm** (ví dụ `bdspro_core_dev`); không áp dụng migration lên `bdspro` đang mang lịch sử Listing cũ khi chưa đối chiếu `schema_migrations`.

```bash
export DB_URL='postgresql://root:secret@localhost:5432/bdspro_core_dev?sslmode=disable'
migrate -path db/migration -database "$DB_URL" -verbose up
migrate -path db/migration -database "$DB_URL" version
sqlc generate -f sqlc.source.yaml
go test ./...
```

Database phải được tạo **trước** câu lệnh trên; `root/secret` chỉ là cấu hình Docker LAB, không sử dụng trong production. Xem hướng dẫn chi tiết: [LOCAL-ACCEPTANCE.md](docs/implementation/LOCAL-ACCEPTANCE.md).

## 6. Danh mục tài liệu tiếng Việt

- [Báo cáo tổng kết giai đoạn đầu](docs/implementation/BAO-CAO-GIAI-DOAN-CORE-01-05.md): đã làm gì, bằng chứng, phạm vi đã nghiệm thu và các việc còn mở.
- [Thiết kế và quyết định kỹ thuật](docs/implementation/CORE-01-TO-05.md): sự thật nghiệp vụ, constraint, giải pháp thay thế, chi phí, rủi ro.
- [Phong cách/quy trình làm việc](docs/implementation/QUY-TRINH-LAM-VIEC.md): phân tích → thực thi → quan sát → commit → checkpoint.
- [Nghiệm thu và đồng bộ local](docs/implementation/LOCAL-ACCEPTANCE.md): lệnh chạy thử đầy đủ từ Account đến Source.
- [Checkpoint đang hoạt động](docs/continuity/CHECKPOINT.md): mốc tiếp tục; phần tiếng Anh phía dưới là lịch sử.
- [Thiết kế canonical DBML](docs/database/final/BDSPro-CANONICAL-DATABASE.dbml): ứng viên logical, không phải 164 migration thực tế.

## 7. Nguyên tắc an toàn và công việc tiếp theo

Không `reset --hard`, `clean -fd`, `force push` hoặc di chuyển schema chỉ để đồng bộ nhanh khi chưa biết local có gì. Không coi email đã xác minh là xác minh con người/đại diện pháp lý. Chưa mở hệ thống cho khách hàng thật nếu chưa đánh giá CSRF, rate limiting, khôi phục tài khoản, retry email, an toàn quyền reviewer, race thu hồi quyền, DB least privilege và dữ liệu cá nhân.

**Bước kế tiếp:** đối chiếu remote `dev` ↔ local không mất dữ liệu, nghiệm thu manual workflow cùng PostgreSQL local, ghi bằng chứng. Sau đó ưu tiên CORE-06 — CRM Inquiry & Follow-up theo đúng cảnh khách ngoài hệ thống hỏi về nguồn hàng, **không ép khách phải có Account BDSPro**.
