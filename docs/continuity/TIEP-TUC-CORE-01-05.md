> **LƯU TRỮ LỊCH SỬ — 2026-10-15:** Prompt CORE-01→05 / GATE-LOCAL-01 / CORE-06 bên dưới là chỉ dẫn của giai đoạn cũ. Hướng hiện tại đã chuyển sang [Engineering Core checkpoint](checkpoints/2026-10-15-engineering-core-reorientation/README.md) và [prompt mới cho chat tiếp theo](checkpoints/2026-10-15-engineering-core-reorientation/CONTINUE-PROMPT.md). Không dùng tài liệu này làm next-task tự động.

---

# Prompt tiếp tục BDSPro — bản tiếng Việt

Tôi đang tiếp tục BDSPro tại repository `datvtph41107/bdspro`, **chỉ làm việc trên nhánh `dev`**. Đọc và ưu tiên trạng thái mới nhất trong:

1. `docs/continuity/CHECKPOINT.md`
2. `README.md`
3. `docs/implementation/BAO-CAO-GIAI-DOAN-CORE-01-05.md`
4. `docs/implementation/QUY-TRINH-LAM-VIEC.md`
5. `docs/implementation/CORE-01-TO-05.md`
6. `docs/implementation/LOCAL-ACCEPTANCE.md`

CORE-01 → CORE-05 đã được code trên GitHub `dev` với migration 000001–000005, Auth, Organization và Source. [CI 38139052788](https://github.com/datvtph41107/bdspro/actions/runs/38139052788) đã chạy thành công PostgreSQL 18, migrate, sqlc, 15 integration tests và server smoke. Đừng nói local đã đồng bộ hoặc app đã production-ready khi chưa có output.

Phương pháp: mô tả tình huống thật → trạng thái trước/sau → bất biến → cơ chế SQL/Go → phương án tương đương và chi phí → kiểm tra âm tính và concurrency khi cần → code theo module lớn, chia commit có trách nhiệm trên `dev` → CI thực → cập nhật báo cáo, checkpoint bằng tiếng Việt. Người dùng sẽ sync local khi kết thúc module; không ghi đè local chưa commit.

Việc kế tiếp: hoàn tất GATE-LOCAL-01 (clone riêng, kiểm tra database LAB và HTTP manual acceptance), sau đó triển khai CORE-06 CRM Inquiry & Follow-up: khách ngoài hệ thống hỏi một Source, nhân viên tổ chức ghi nhận yêu cầu và lịch sử chăm sóc; không tự ép Contact thành Party/Account. Chủ động tiếp tục theo bằng chứng, không hỏi lại các quyết định đã rõ.
