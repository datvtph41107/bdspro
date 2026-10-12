# CORE-01 — Khóa loại danh tính Party (2026-10-12)

## Quyết định

Giữ `parties` + `persons` + `organizations` làm baseline hiện hành vì Account/Membership cần FK đúng subtype, đồng thời có những nghiệp vụ thương mại dự kiến cần Party ID chung. Đây là lựa chọn có điều kiện, không coi thêm bảng là lợi ích tự thân. Chưa có benchmark định lượng A/B/C; không suy luận hiệu năng từ việc đếm bảng.

## Lỗ hổng ngữ nghĩa trong 000001

Constraint trigger C01 kiểm tra mỗi Party có một subtype phù hợp ở COMMIT. Khi chưa có guard bất biến, một transaction có thể xóa `persons(101)`, đổi `parties(101).kind` sang ORGANIZATION và tạo `organizations(101)`: C01 cuối transaction vẫn đúng nhưng cùng một ID đã thay đổi bản chất chủ thể.

Một Person gia nhập/rời ABC chỉ thay đổi Membership, không thay đổi Person hoặc Party ID. Một Person lập công ty phải tạo Party mới cho công ty.

## Cơ chế

`000006_party_kind_immutability.up.sql` thêm một `BEFORE UPDATE` trigger trên `parties`. Khi OLD.kind khác NEW.kind, từ chối ngay bằng SQLSTATE 23514, kể cả khi transaction đang chuyển toàn bộ subtype. UPDATE không đổi kind vẫn được phép. Không sửa lịch sử `000001` đã được triển khai trên các DB thử nghiệm.

Chi phí: một trigger chạy khi cập nhật Party (đường ghi rất ít xảy ra), một function và một migration rollback tương ứng. Không thêm bảng hoặc index. Một DB role đủ quyền DDL/disable trigger có thể vượt guard; cần least-privilege trong production.

## Phản chứng và kiểm thử

- Tạo PERSON với subtype, COMMIT được; tạo ORGANIZATION với subtype, COMMIT được.
- UPDATE kind không đổi thành công.
- Xóa subtype rồi cố đổi kind trong cùng transaction phải nhận 23514 và ROLLBACK nguyên trạng; thử hai chiều.
- Một transaction giữ row lock và transaction khác cố đổi kind: transaction thứ hai không được đổi kind.
- Giữ các test C01 hiện có: thiếu subtype ở COMMIT bị từ chối.
- `up → down 1 → up 1` trên DB dùng riêng; kiểm tra rollback không xóa Party data.

Những bài kiểm thử này không chứng minh mọi lịch trình concurrency/DDL có đặc quyền đã được bao phủ. Không kết luận production-ready chỉ từ CI.

## Phần chưa khóa

Tính bất biến của `parties.id`, chính sách xóa Identity, migration sang loại Party mới và chi phí benchmark A/B/C là các bài toán độc lập. Không phát sinh thêm subtype do User có nghề nghiệp hoặc Role mới. Source cá nhân là ranh giới quản lý tài nguyên, không thay đổi bản chất Person/Organization.
