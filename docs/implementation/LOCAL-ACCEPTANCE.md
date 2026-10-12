# Hướng dẫn đồng bộ và nghiệm thu CORE-01 → CORE-05 tại local

> Hướng dẫn dành cho **PostgreSQL 18 và code branch `dev`**, trong môi trường thử nghiệm. Tính đến 12/10/2026, nhánh `dev` có thêm migration `000006_party_kind_immutability` (bảo vệ loại Party không đổi); lệnh `migrate ... up` phải đi đến version **6** trên database LAB mới. CI [38221355862](https://github.com/datvtph41107/bdspro/actions/runs/38221355862) đã thành công; local vẫn phải tự nghiệm thu. Chỉ thực hiện thao tác có thể làm thay đổi dữ liệu sau khi đã đọc đúng database/branch. **Không ghi đè thư mục `~/projects/bdspro` cũ.**

## A. Lấy đầy đủ code từ remote về thư mục mới

Trên terminal WSL/Linux:

```bash
cd ~/projects
git clone --branch dev --single-branch \
  https://github.com/datvtph41107/bdspro.git bdspro-core-complete

cd bdspro-core-complete
git status --short --branch
git log -1 --oneline
```

Mục tiêu: nhận được một worktree sạch từ remote. Không yêu cầu xóa hoặc ghi đè bất kỳ thay đổi chưa commit nào trong `~/projects/bdspro`. Nếu tên `bdspro-core-complete` đã tồn tại, **không chạy lệnh clone đè**. Ghi nhận SHA HEAD thực tế trước khi thử nghiệm.

## B. Dùng database mới, không sử dụng nhầm database cũ

Trước tiên kiểm tra PostgreSQL và tên database:

```bash
docker ps --filter name=postgres
docker exec postgres psql -U root -d postgres \
  -c "SELECT datname FROM pg_database WHERE datname='bdspro_core_dev';"
```

**Chỉ khi chưa có** `bdspro_core_dev`, tạo một database thử nghiệm mới:

```bash
docker exec postgres createdb -U root bdspro_core_dev
```

Thiết lập connection URL và xác nhận database đích:

```bash
export DB_URL='postgresql://root:secret@localhost:5432/bdspro_core_dev?sslmode=disable'
psql "$DB_URL" -c "SELECT current_database(), current_user;"
```

Nếu đã có `bdspro_core_dev`, phải kiểm tra `schema_migrations`/các bảng trong đó trước; đừng giả định DB trống. Không tự động reset, drop hoặc chạy `force` migration.

### Chạy migration và kiểm tra code

Yêu cầu CLI `migrate` (golang-migrate) và `sqlc` đã cài. Tại clone mới:

```bash
migrate -path migrations -database "$DB_URL" -verbose up
migrate -path migrations -database "$DB_URL" version

sqlc generate -f sqlc.source.yaml
go test ./...
git status --short --branch
```

Trong một checkout hoàn toàn khớp CI, sinh lại sqlc không nên làm thay đổi các file generated đã commit. Chạy `git diff -- db/sourcesqlc go.mod go.sum` nếu phát hiện thay đổi. `root/secret` chỉ là credential trong Docker LAB, tuyệt đối không dùng cho production.

**Phân biệt test:** `go test ./...` khi không có biến `SOURCE_TEST_DATABASE_URL` và `CORE_TEST_DATABASE_URL` sẽ **SKIP** các bài integration test có PostgreSQL (thay vì tự tạo bảng/dữ liệu). Để tự chạy toàn bộ ca giống CI trên local, cần tạo **hai database thử nghiệm trống, riêng biệt, đúng tên** `bdspro_source_ci` và `bdspro_core_ci`, áp dụng đủ migration vào từng database, rồi đặt hai biến môi trường tương ứng theo chỉ dẫn trong `.github/workflows/bdspro-core.yml`. Các bài test có lệnh `TRUNCATE ... CASCADE`: **tuyệt đối không gán chúng vào database có dữ liệu cần giữ**, kể cả khi tên trùng. Nếu chưa xác nhận DB trống, chỉ dựa vào CI làm bằng chứng integration và tiến hành hành trình HTTP thủ công trên `bdspro_core_dev`.

## C. Khởi chạy API

```bash
export DATABASE_URL="$DB_URL"
export APP_ENV=development
export BIND_HOST=127.0.0.1
export PORT=8080
go run .
```

Trong cấu hình development, app chỉ cho chạy trên loopback để tránh lộ token xác minh trong response. Không chạy hướng dẫn này trên máy chủ công khai.

Các endpoint chính:

| Nhóm | Đường dẫn |
| --- | --- |
| Account | `POST /auth/register`, `POST /auth/verify-email`, `POST /auth/login`, `GET /auth/me`, `POST /auth/logout` |
| Organization | `POST /organizations/registrations`, `POST /organizations/{id}/claims` |
| Reviewer | `POST /platform/organizations/registrations/{id}/approve`, `POST /platform/organizations/claims/{id}/approve` |
| Membership | `POST /organizations/{id}/invitations`, `POST /organizations/invitations/{id}/accept`, `GET /organizations/{id}/members` |
| Source | `POST /organizations/{id}/sources`, `GET /organizations/{id}/sources/{sourceID}` |
| Price | `POST /organizations/{id}/sources/{sourceID}/price-reports`, `GET /organizations/{id}/sources/{sourceID}/price-reports` |

## D. Hành trình thủ công: An, Lan và reviewer

Các email bên dưới thuộc miền `.test` và chỉ dùng làm ví dụ. Dùng 3 Account riêng, không gộp chủ thể.

### D1. Đăng ký An, xác minh rồi đăng nhập

```bash
curl -sS -X POST http://127.0.0.1:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"an@example.test","password":"LabLongPassword1234!"}'
```

Response môi trường development bao gồm `account_id` và `verification_token`. Token **không tự động được gửi về đây trong production**.

Dùng token vừa nhận để xác minh:

```bash
curl -i -X POST http://127.0.0.1:8080/auth/verify-email \
  -H 'Content-Type: application/json' \
  -d '{"token":"TOKEN_VUA_NHAN"}'
```

Làm tương tự cho `lan@example.test` và `reviewer@example.test`, lấy **token riêng** từng Account. Trước khi lưu cookie file, giới hạn quyền của terminal hiện tại:

```bash
umask 077
curl -i -c /tmp/bdspro-an.cookies \
  -X POST http://127.0.0.1:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"an@example.test","password":"LabLongPassword1234!"}'

curl -b /tmp/bdspro-an.cookies http://127.0.0.1:8080/auth/me
```

Đăng nhập riêng reviewer/Lan, dùng `/tmp/bdspro-reviewer.cookies` và `/tmp/bdspro-lan.cookies`. Cookie file chứa bí mật phiên đăng nhập, không commit hoặc gửi lên mạng.

### D2. Cấp quyền reviewer CHỈ cho LAB

Một người dùng bình thường hoặc admin Organization **không** tự tạo reviewer nền tảng. Trong LAB riêng, quản trị viên PostgreSQL có thể tạo dữ liệu test bằng lệnh:

```bash
docker exec postgres psql -U root -d bdspro_core_dev -v ON_ERROR_STOP=1 \
  -c "INSERT INTO platform_organization_reviewers(account_id,authorization_reference)
      SELECT a.id, 'LAB-ONLY-operator-approval'
      FROM accounts a
      JOIN account_emails ae ON ae.account_id=a.id
      WHERE ae.email='reviewer@example.test' AND ae.verified_at IS NOT NULL;"
```

Lệnh này **chỉ** cho phép Account reviewer thực hiện hành vi phê duyệt ở LAB, không chứng minh tư cách pháp lý hoặc thẩm quyền ngoài đời. Không tạo API công khai để chạy câu lệnh ấy.

### D3. An khai báo Công ty ABC; reviewer duyệt

An gửi yêu cầu:

```bash
curl -b /tmp/bdspro-an.cookies \
  -H 'Content-Type: application/json' \
  -X POST http://127.0.0.1:8080/organizations/registrations \
  -d '{"name":"ABC Realty Lab","jurisdiction":"VN","registration_code":"LAB-ABC-2026"}'
```

Lưu `request_id`. **Trước khi reviewer duyệt**, yêu cầu tồn tại nhưng Organization chưa được tạo. Trong LAB kiểm tra:

```sql
SELECT id, decision_code, matched_organization_id
FROM organization_registration_requests ORDER BY id;

SELECT party_id, name FROM organizations ORDER BY party_id;
```

Reviewer duyệt sau khi đã kiểm tra hồ sơ LAB:

```bash
curl -b /tmp/bdspro-reviewer.cookies \
  -H 'Content-Type: application/json' \
  -X POST http://127.0.0.1:8080/platform/organizations/registrations/REQUEST_ID/approve \
  -d '{"review_reference":"LAB-ONLY-reviewed-company-identity"}'
```

Thay `REQUEST_ID` bằng số từ response thực; ghi lại `organization_id` trong kết quả. Transaction tạo Party/Organization, registry identifier, Membership và Role quản trị workspace. Không được diễn giải reviewer LAB là xác minh pháp lý thực tế.

### D4. An mời Lan

Đọc Role ID từ LAB database:

```bash
docker exec postgres psql -U root -d bdspro_core_dev \
  -c "SELECT organization_id, id, name FROM organization_roles ORDER BY id;"
```

An mời Lan (thay `ORG_ID`, `ROLE_ID` bằng số thực):

```bash
curl -b /tmp/bdspro-an.cookies \
  -H 'Content-Type: application/json' \
  -X POST http://127.0.0.1:8080/organizations/ORG_ID/invitations \
  -d '{"email":"lan@example.test","role_id":ROLE_ID}'
```

Lưu `invitation_id`. Lan chấp nhận:

```bash
curl -b /tmp/bdspro-lan.cookies \
  -X POST http://127.0.0.1:8080/organizations/invitations/INVITATION_ID/accept
```

Kiểm tra danh sách:

```bash
curl -b /tmp/bdspro-an.cookies \
  http://127.0.0.1:8080/organizations/ORG_ID/members
```

Điểm quan sát: lời mời **chưa** đồng nghĩa Membership; Membership/Role chỉ xuất hiện sau lần chấp nhận thành công.

### D5. Lan tiếp nhận nguồn và bổ sung lịch sử giá

```bash
curl -b /tmp/bdspro-lan.cookies \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: lab-capture-001' \
  -X POST http://127.0.0.1:8080/organizations/ORG_ID/sources \
  -d '{"channel":"zalo","label":"Tin báo về một căn nhà","area_m2":"75","initial_price":{"amount":"4200000000","currency":"VND","price_basis":"asking"}}'
```

Lưu `id` Source từ response, thay vào `SOURCE_ID` bên dưới. Gửi lại **đúng request và key**: kỳ vọng HTTP 200/replay, **không thêm bản ghi**. Đổi payload nhưng giữ key: kỳ vọng HTTP 409.

Ghi nhận người cung cấp báo giá khác:

```bash
curl -b /tmp/bdspro-lan.cookies \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: lab-price-002' \
  -X POST http://127.0.0.1:8080/organizations/ORG_ID/sources/SOURCE_ID/price-reports \
  -d '{"amount":"4100000000","currency":"VND","source_basis":"reconfirmed_call"}'
```

Xem lịch sử:

```bash
curl -b /tmp/bdspro-lan.cookies \
  http://127.0.0.1:8080/organizations/ORG_ID/sources/SOURCE_ID/price-reports
```

Đầu ra cần giữ cả **4,2 tỷ** và **4,1 tỷ** dưới dạng hai thông tin đã được báo. Không có Property/Listing được tự động sinh. Thời điểm ghi vào DB không luôn là thời điểm người cung cấp đưa ra báo giá.

## E. Thử nghiệm phản chứng và điểm bảo vệ

- Dùng cookie của một thành viên **không thuộc ABC** gọi vào Source ABC: không được đọc/ghi.
- Kết thúc Membership của Lan hoặc ghi Block còn hiệu lực: Lan không đọc được Source ABC.
- Đăng ký lặp email khác hoa/thường: không phát sinh thêm Account/Party vô chủ.
- Thử thiếu `sources.price_report` khi capture có giá: Source và Price phải cùng rollback.
- Thử Session sau logout: `/auth/me` trả 401.
- Ghi một Price liên kết sai `organization_id` với `intake_id`: composite FK từ chối.

Chỉ chạy các thay đổi Membership/Role bằng SQL trong **LAB**. Dùng `SELECT` đối chiếu trước/sau. Không tùy tiện `DELETE` dữ liệu muốn giữ.

## F. Ghi biên bản nghiệm thu

Mỗi ca lưu: ngày giờ, `git rev-parse HEAD`, tên database, migration version, actor, input (không chứa token/password), output/status, số hàng trước/sau, nhận định cơ chế và phạm vi điều **chưa** được chứng minh. Xoá cookie file thử nghiệm khi hoàn thành:

```bash
rm -f /tmp/bdspro-an.cookies /tmp/bdspro-lan.cookies /tmp/bdspro-reviewer.cookies
```

**Nghiệm thu GitHub CI khác nghiệm thu local/production.** Quy trình này là mốc xác minh local sau khi clone; review pháp lý, bảo mật, tải thực tế, email outbox, CSRF/rate limiting và đồng bộ dữ liệu cũ vẫn cần công việc riêng.
