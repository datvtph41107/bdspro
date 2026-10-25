## Problem / Intent

<!-- Vấn đề thật là gì? Muốn thay đổi hành vi nào? Đã đọc những source/test nào? -->

## Owner / Invariant

- **Capability và Data Owner:** <!-- ví dụ: auth + db/query/session.sql -->
- **Actor / Action / Resource:** <!-- nếu áp dụng -->
- **Bất biến phải giữ:** <!-- kể cả deny, rollback, retry, concurrency -->
- **Route / bảng / config bị ảnh hưởng:** <!-- chỉ liệt kê phần thực sự thay đổi -->

## Implementation / Scope

<!-- Vì sao đây là thay đổi nhỏ nhất hợp lý? Nếu thêm interface/package/migration, lý do cụ thể? -->
<!-- Nếu tạo capability/abstraction: contract, runtime owner, domain invariant, reuse boundary, retry/rollback; xem docs/engineering/capability-workbook.md. Không bắt buộc cho thay đổi nhỏ. -->

## Evidence / Acceptance

- **Test/check đã chạy và kết quả:** <!-- nêu skip nếu có -->
- **CI:** <!-- link và trạng thái khi có -->
- **Runtime / integration:** <!-- đã kiểm tra, chưa kiểm tra hoặc không áp dụng -->
- **Rủi ro còn lại / ngoài phạm vi:** <!-- nhất là data, volume, external providers -->
- **Người review và quyết định:** <!-- tên reviewer thực tế nếu đã được chỉ định -->

Đối chiếu [Core Ownership Matrix](../docs/engineering/core-ownership.md) và [Development Contract](../docs/engineering/development-contract.md). Không tuyên bố WSL, provider E2E hoặc khả năng recovery đã nghiệm thu chỉ dựa trên CI.
