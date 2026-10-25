# ENG-GOV-03 — Formula Catalog & Capability Engineering

**Scope:** Chuẩn hóa 3 context / 18 công thức hoặc mô hình tư duy từ quá trình mentor BDSPro: (1) quyết định kỹ thuật, (2) runtime extension & domain operations, (3) tích lũy song song Platform/Domain. Đây là **catalog quyết định**, không phải framework runtime, code generator hay policy CI mới.

**Source of truth:** source trên nhánh dev của BDSPro và test, SQL, migrations của chính dự án. Reference DRX (datvtph41107/laravel-drx, main) chỉ dùng để học cơ chế Laravel; không sao chép framework, financial formula hoặc cấu trúc package vào Go khi chưa có yêu cầu.

**Liên quan:** [Development Contract](development-contract.md) · [Core Ownership](core-ownership.md) · [ENG-GOV-02](workspace-guardrails.md) · [Workbook áp dụng](capability-workbook.md).

## A. Cách đọc công thức

- Các dấu **+** trong biểu thức thiết kế là ký hiệu ghép **những điều kiện cần xét**, không phải cộng đại lượng cùng đơn vị. Không thể suy ra score/kết quả số học.
- **F05** là mô hình lựa chọn ràng buộc: chỉ so sánh những thiết kế đã đáp ứng yêu cầu, ownership, invariants; không chứng minh được optimum khi chưa có chi phí đo lường.
- **F10–F12** là phép tính minh họa có ý nghĩa khi đơn vị, precision, rounding và điều kiện nghiệp vụ đã được xác định; **F11 không mô tả đầy đủ matching engine**.
- **Gates** là bước suy luận/review tương xứng với thay đổi, **không bắt buộc tạo file hoặc interface** cho mỗi công thức.
- Năng lực được đánh dấu **Understood / Verified / Operational** với bằng chứng và phạm vi riêng. Không nâng mức chỉ vì có tài liệu hoặc CI xanh.

## B. CTX-01 — Engineering Operating Model (WHY / DESIGN)

| ID | Công thức / mô hình | Câu hỏi áp dụng | Kết quả phải chỉ ra |
| --- | --- | --- | --- |
| **F01** | Capability = Mechanism + Policy + Context + Failure Semantics | Cơ chế chung là gì? Policy khác nhau ở đâu? Lỗi phải phản ứng thế nào? | Contract hành vi và giới hạn trách nhiệm |
| **F02** | Total Cost = Execution + Change + Coupling + Failure + Maintenance | Tối ưu throughput hay giảm chi phí thay đổi? Có đo không? | Trade-off, giả định, dữ liệu đo hoặc ghi rõ chưa đo |
| **F03** | Reuse = Shared Invariant + Stable Behavior + Compatible Failure Semantics | Hai đường code giống cú pháp hay thật sự có chung invariant? | Giữ độc lập / dùng chung primitive / tách helper, kèm lý do |
| **F04** | Problem → Invariant → Owner → Reuse → Implementation → Verification → Acceptance | Có vấn đề thật và điều kiện chấp nhận cụ thể? | Một vòng thay đổi có chủ đích, không mở rộng ngoài phạm vi |
| **F05** | D* = argmin{C(D) \| D đáp ứng các bất biến và ràng buộc} | Những phương án nào hợp lệ? Phương án nào nhỏ nhất? | So sánh phương án và quyết định; không tuyên bố optimum nếu chưa định lượng |
| **F06** | Engineering Value = Verified Capability − Unnecessary Complexity | Thêm abstraction có tăng năng lực đã được kiểm chứng? | Lý do giữ/thêm/hoãn/bỏ code hoặc package |

**Source example:** IPActiveSource chặn trước parsing, IPActive chặn sau parsing theo identity; cùng sử dụng primitive Redis nhưng có điểm admission khác nhau (api/server.go; api/middleware_ipactive.go). Việc hợp nhất hai middleware theo tên giống nhau có thể phá bất biến **malformed request vẫn bị source quota**.

## C. CTX-02 — Application Runtime & Extension Contract (WHAT / HOW)

| ID | Công thức / mô hình | Câu hỏi áp dụng | Kết quả phải chỉ ra |
| --- | --- | --- | --- |
| **F07** | Application Capability = Runtime Mechanism + Extension Contract + Domain Policy + Operational Guarantees | Vendor cung cấp gì và application phải triển khai phần nào? | Ranh giới vendor/adapter/domain và non-guarantees |
| **F08** | Platform Capability = (Contract, Runtime, Adapter, Policy, Failure) | Ai khởi tạo, chạy, tương tác hạ tầng và xử lý lỗi? | Capability contract đủ để tích hợp/đọc/test |
| **F09** | Domain Operation = (Command, Validation, Calculation, Transition, Persistence, Effects) | Input tạo thay đổi nào, transaction bao phủ tới đâu? | Operation invariant, effects và chủ sở hữu |
| **F10** | Amount = Price × Quantity | Đơn vị, giá trị, scale và rounding nào hợp lệ? | Công thức domain có input/output xác định |
| **F11** | MatchQuantity = min(BuyAvailableQuantity, SellRemainingQuantity) | Lượng khớp thực còn chịu balance, order type, giá, thứ tự nào? | Một thành phần của matching logic, **không phải engine hoàn chỉnh** |
| **F12** | Payable = Quantize(BaseAmount + Fees − Discounts, Scale, RoundingMode) | Làm tròn trước/sau cộng? Ai cấp fee policy? | Công thức có rounding/precision, version policy nếu cần tái hiện |

**Reference grounded in DRX (main):** composer.json autoload modules; modules/IPActive/src/IPActiveServiceProvider.php đăng ký middleware, event provider, migrations, commands, scheduler; app/Utils/BigNumber.php đóng gói BCMath; app/Facades/AmalCalculator.php triển khai các công thức; app/Jobs/ProcessOrderRequest.php implements ShouldQueue; app/Http/Services/OrderService.php gồm tính khớp, giá/phí và các thao tác DB. Những thực thể này là **ví dụ tham chiếu**, không phải tính năng đã có trong BDSPro.

**Boundary quan trọng:** Laravel xử lý dispatch/worker; DRX vẫn phải bảo vệ retry, accounting, precision và consistency. Trong ProcessOrderRequest::handle(), exception được ghi log rồi không ném lại; điều này có thể làm worker xem lượt job đã hoàn tất. **Không copy** cấu trúc ấy để tuyên bố retry tự động.

**Go translation hiện có:** main.go là composition root; api.NewServer là HTTP composition; db/sqlc.Store và SQL queries sở hữu PostgreSQL operations; worker.TaskDistributor / TaskProcessor sử dụng Asynq; token.Maker sở hữu JWT; chưa có decimal/money package, generalized event bus, file storage capability hoặc application scheduler được chứng minh là cần thiết. Không dựng provider/container/engine trước use case.

## D. CTX-03 — Parallel Engineering Accumulation (LEARN / VERIFY / EVOLVE)

| ID | Công thức / mô hình | Câu hỏi áp dụng | Kết quả phải chỉ ra |
| --- | --- | --- | --- |
| **F13** | Platform FA = (Contract, Runtime, Lifecycle, Dependencies, Error) | Work được bắt đầu, xử lý, hủy, retry và quan sát thế nào? | Runtime owner + vận hành thất bại |
| **F14** | Domain FB = (Input, Unit, Precision, Validation, Transition, State) | Quy tắc nào biến input thành trạng thái đúng? | Business invariant và đơn vị dữ liệu |
| **F15** | Verified Capability = Platform Contract ⊗ Domain Contract ⊗ Evidence | Có test xác minh đúng cả runtime lẫn nghiệp vụ? | Evidence gắn rõ claim; ⊗ là sự phối hợp, không phải phép nhân |
| **F16** | Observe → Understand → Model → Apply → Verify → Consolidate | Điều gì còn lại để dùng cho lần kế tiếp? | Contract, test hoặc quyết định hoãn; không nhất thiết có package |
| **F17** | Knowledge Reuse / Code Reuse / Operational Reuse | Tái sử dụng kiến thức, hàm, hay cách vận hành? | Phân loại đúng loại reuse, tránh ép tạo shared module |
| **F18** | Accumulation Value = Mechanism Understanding + Invariant Understanding + Correct Application + Evidence | Developer có thể tự giải thích và kiểm chứng? | Ghi nhận hiểu biết có giá trị, giới hạn và vấn đề còn mở |

**Mức độ evidence:**

| Mức | Điều kiện tối thiểu | Không có nghĩa |
| --- | --- | --- |
| **Understood** | Đọc source, gọi đúng lifecycle/contract/failure mode | Đã có implementation hoặc test |
| **Verified** | Có test/measurement tương ứng với một claim và một scope cụ thể | Production-ready hoặc mọi code path đã được kiểm tra |
| **Operational** | Có bằng chứng runtime thực tế, owner, failure/recovery, cảnh báo và runbook theo yêu cầu | Zero downtime hoặc exactly-once mặc định |

## E. Quy tắc chọn mức tái sử dụng

1. **Existing vendor capability:** ưu tiên cách dùng chuẩn của Go, pgx, Gin, Redis/Asynq trước khi viết runtime mới.
2. **Shared primitive:** tách hàm/package chỉ khi có invariant chung bền vững hoặc việc phân tán gây lỗi cụ thể; không đưa business rule vào primitive.
3. **Application capability:** xác định Contract/Runtime/Policy/Failure; composition tại bootstrap, không cần service container động nếu constructor đủ rõ.
4. **Domain engine:** chỉ dựng khi tồn tại nghiệp vụ có state machine/calculation/concurrency thật; quyết định rounding và tính đúng trước hiệu năng.
5. **No abstraction:** một handler + SQL predicate có thể đúng nếu owner rõ, invariant được DB bảo vệ và test đủ.
6. **No premature optimization:** benchmark, query plan hay đo tải được yêu cầu trước khi tuyên bố cải thiện hiệu năng.

## F. Cổng áp dụng thực tiễn (không phải 18 checklist bắt buộc)

Với **một thay đổi**: nêu Problem, Source, Owner, Invariant, phương án nhỏ nhất, Evidence và Non-goals theo [Development Contract](development-contract.md).

Với **một capability mới**: bổ sung bảng theo mẫu ở [Capability Workbook](capability-workbook.md), chỉ rõ contract/runtime/domain/failure và acceptance. Nếu chưa có yêu cầu thật, đánh dấu **defer**.

Với **một lỗi vận hành**: xác định event thực, affected state, retry/rollback/compensation, ranh giới PG–Redis và evidence; không dùng tên framework để thay thế bảo đảm thực tế.

**Acceptance catalog:** tính nhất quán thuật ngữ được review và kiểm tra CI cho thay đổi tài liệu. Catalog tự nó **không** nghiệm thu Business Core, không thay thế WSL G0, không tạo CI enforcement hay giảm latency khi chưa đo.
