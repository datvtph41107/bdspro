# ADR-0001 — Sở hữu Core dùng chung mà không xây dựng thêm framework

- **Quyết định:** Chấp nhận khi ENG-GOV-01 merge vào `dev`
- **Ngày:** 24/10/2026
- **Phạm vi:** Quy chuẩn Engineering; không thay đổi hành vi, DB schema hoặc triển khai Business Core

## Bối cảnh

BDSPro chạy Go/Gin, PostgreSQL/sqlc và Redis/Asynq. Source hiện có `api`, `auth`, `token`, `db/sqlc`, `worker`, `mail`, `main.go`, các route Authentication và Organization membership authorization.

Hai repository tham chiếu cung cấp hai hướng tổ chức:

- **DRX Laravel:** Middleware theo hành động (`IPActive`, `PreLogin`, `AuthenticateUser`, `SufLogin`). Tuy nhiên, trong source tham chiếu, `PreLogin` còn kiểm tra password, `SufLogin` có thể xử lý OTP/device và thay đổi token sau khi handler chạy. BDSPro chỉ kế thừa ngôn ngữ kỹ thuật khi trách nhiệm thực sự khớp tên gọi.
- **Java DRWash:** Gradle module `core` được `api`, `admin`, `store`, `IoT` dùng chung. `OrderCoreService` minh họa shared domain ownership, nhưng Core quá lớn có thể gom những trách nhiệm không liên quan. BDSPro chưa có lý do chứng minh phải chia nhiều Go module/ứng dụng.
- **SimpleBank:** Tinh thần viết Go, SQL và transaction trực tiếp, dễ đọc và kiểm chứng.

BDSPro đang thiếu contract thống nhất để developer xác định ownership, vị trí thực thi bất biến và tiêu chí nghiệm thu. Dựng thêm framework trước Business Core sẽ tăng độ phức tạp mà chưa có giá trị chứng minh được.

## Quyết định

1. Ownership dựa trên **hành vi và trạng thái**, không dựa trên bộ khung package cố định. Dùng [Core Ownership Matrix](../core-ownership.md) và [Development Contract](../development-contract.md), cập nhật khi source thay đổi.
2. **Giữ một Go application/module hiện tại** cho Gin HTTP và worker. Không bắt buộc tạo hierarchy `core/domain/repository/service/adapter`.
3. `api` sở hữu HTTP và request policy; `auth`/`token` sở hữu danh tính và cơ chế session; SQL/constraints sở hữu bất biến persisted data và authorization trên resource; `worker`/`mail` xử lý async/delivery.
4. Khi quyền phải nhất quán với mutation, ưu tiên predicate authorization ngay trong SQL/transaction, không mặc định tin role từ client hoặc gắn owner flag vào JWT.
5. Chỉ bổ sung interface/abstraction khi có reuse thật, nhiều implementation, nhu cầu test hoặc ranh giới trách nhiệm rõ. Không loại bỏ interface hiện có chỉ để chứng minh nguyên tắc tối giản.
6. Dùng PR template ngắn gọn. Không tự tạo `CODEOWNERS` khi chưa phân công maintainer và trách nhiệm review thực tế.
7. CI xanh xác minh **phạm vi được test**, không đồng nghĩa WSL runtime, khả năng phục hồi data, provider thật hoặc production đã nghiệm thu.

## Hệ quả

**Lợi ích:** developer tìm đúng nơi sửa code, reviewer truy ra invariant, và Business Core tương lai dùng được Identity/Organization chung mà không tạo hệ thống authentication riêng.

**Chi phí:** ma trận ownership cần được giữ đúng với source, discipline phụ thuộc vào review và test. Một handler đơn giản gọi Store trực tiếp vẫn có thể là đúng; không có service layer cưỡng bức để che giấu độ phức tạp.

**Chưa giải quyết:** G0 WSL, atomic PG–Redis handoff, chính sách unverified email, Google/SMS production E2E và mô hình Business Core.

## Điều kiện xem xét lại

Chỉ mở quyết định khi xuất hiện nhu cầu cụ thể: hai entrypoint phải dùng chung quy tắc nghiệp vụ, package có owner xung đột, code lặp gây lỗi, hoặc tách deployment có lợi ích đo được. Phải mô tả tác động, phương án chuyển đổi và bằng chứng trước khi tách module.

## Các phương án không chọn

- **Sao chép DRX middleware:** có thể trộn xác minh credential với xử lý OTP/token sau controller.
- **Sao chép multi-module DRWash:** tăng phụ thuộc/build khi BDSPro chưa có ứng dụng thứ hai cần chia sẻ.
- **Tạo Clean Architecture skeleton cho mọi CRUD:** buộc interface/service trước khi có trách nhiệm thật.
- **Để ownership ngầm định:** developer dễ định nghĩa lại session, policy hoặc quyền ở những nơi khác nhau.
