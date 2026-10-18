# Lecture #70 — RBAC: thu hồi membership và đồng bộ với Source writes

## Mục tiêu

BDSPro không coi session hợp lệ là có quyền đối với mọi tổ chức. Bài này kiểm chứng
và triển khai một hành động thực: chủ thể có permission `members.remove` thu hồi
membership của người khác; quyền tenant ngừng hiệu lực, nhưng session đăng nhập
không bị xóa.

Tất cả thay đổi nằm trong Go/Gin/SQLC/PostgreSQL hiện có; không cần AWS, Redis,
JWT hay một framework policy ngoài.

## Hợp đồng HTTP

`DELETE /organizations/:id/members/:membershipID`

- 401: thiếu hoặc session không hợp lệ.
- 503: không thể xác minh session vì backend lỗi.
- 400: `id` hoặc `membershipID` không phải số nguyên dương hợp lệ.
- 404: tổ chức/thành viên không tồn tại, đã kết thúc, người gọi thiếu quyền,
  truy cập sai tổ chức, hoặc đối tượng là primary administrator.
- 204: thu hồi thành công, không có body.
- Lỗi DB bất ngờ: 500.

Không trả về quyền hay sự tồn tại của tài nguyên tenant cho người không được phép.

## Authority và kiểm soát quyền

**Authentication:** `api.requireSession` kiểm chứng cookie thông qua
`auth.Service.AccountFromToken` và cung cấp account ID trong `gin.Context`.

**Authorization:** `EndAuthorizedOrganizationMembership` áp dụng điều kiện SQL
theo account -> person -> active membership -> active role assignment ->
`organization_role_permissions`, từ chối membership bị block và role đã ended,
đồng thời từ chối thu hồi `organization_primary_administrators`.

**Atomicity:** `organization.Service.RemoveMember` chạy trong `Store.ExecTx`,
khóa organization `FOR UPDATE`, kết thúc membership và các role assignment
trong cùng transaction. Nếu có lỗi thì rollback cả hai phần.

**Least privilege:** `members.remove` khác `members.invite`; chỉ được cấp
tự động cho Workspace Administrator khi bootstrap. Migration `000007` cấp bù
cho role active của primary admin đã tồn tại, không tự cấp cho mọi role cùng tên.

## Thứ tự khóa và cạnh tranh

Ghi Source (`Capture`, `ReportPrice`) mở transaction và khóa hàng tổ chức
`FOR SHARE` **trước** khi kiểm tra authorization và ghi dữ liệu. Thu hồi
member khóa cùng hàng `FOR UPDATE` trước khi đổi quyền.

| Giao dịch nào đến trước? | Thứ tự |
|---|---|
| Source giữ `FOR SHARE` trước | Source hoàn tất/rollback → removal được chạy |
| Removal giữ `FOR UPDATE` trước | Removal COMMIT → Source lấy SHARE và kiểm tra quyền ở statement mới → từ chối |

Vì `FOR SHARE` tương thích với `FOR SHARE`, nhiều Source writes trong cùng
tổ chức có thể chạy đồng thời. `FOR UPDATE` dùng để đặt mốc tuần tự hóa
với thao tác thu hồi. Không cache permissions xuyên requests.

**Giới hạn:** Điều này chỉ bảo đảm thứ tự giữa endpoint removal ở đây và hai
Source writes đã thực hiện khóa. Các thao tác khác thay role, block member hoặc
update role_permissions trực tiếp qua SQL chưa có chung giao thức khóa.
Read-only SELECT dùng snapshot theo statement, không đồng bộ với removal
đang diễn ra; kết quả read đã bắt đầu trước COMMIT có thể vẫn trả dữ liệu.
Mọi admin mutation bổ sung sau này cần tuân thủ lock order tương ứng.

## Tests và nghiệp vụ

- `TestRBACRemoveMemberDoesNotInvalidateSession`: anonymous 401, không có
  permission 404, cross-tenant 404, primary admin được bảo vệ, removal 204,
  membership/role cùng kết thúc, session còn hợp lệ, Source đọc/ghi bị từ chối,
  chủ sở hữu còn quyền.
- `TestRBACSourceWriterLockBlocksMemberRemoval`: PostgreSQL `55P03` chứng minh
  SHARE và UPDATE conflict; removal không được vượt qua transaction Source đang giữ khóa.
- Các integration tests về Auth, Organization, Source và CI phải tiếp tục PASS.

## Nâng cấp schema

Migration `000007_membership_removal` thêm permission và grant; không sửa lịch sử
`000001`–`000006`. Chạy migration qua quy trình `make migrateup DB_URL=...`
đã có, chỉ trên database chủ động chỉ định. Không chỉnh sửa trực tiếp
database đang lưu dữ liệu nếu chưa kiểm tra migration version và backup.

DOWN `000007` hủy grants của permission này, chỉ dùng trong CI/LAB riêng,
không phải phương án rollback production.

## Tiêu chuẩn nghiệm thu

Trạng thái cuối cùng phải kiểm tra từ GitHub Actions HEAD: migrations UP-DOWN-UP,
SQLC generated diff sạch, Go tests, race tests và Docker smoke. CI xanh chứng
minh đúng các kịch bản đã chạy, không chứng minh production-ready hay mọi
kiểu cạnh tranh phân quyền đều đã được giải quyết.
