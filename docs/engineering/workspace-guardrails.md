# ENG-GOV-02 — Nền làm việc kiểm chứng được

**Mục tiêu:** Chuyển quy chuẩn ENG-GOV-01 thành các hành vi làm việc lặp lại và cổng kiểm tra tự động ở mức phù hợp. Nền làm việc chắc nghĩa là **có cách phát hiện và từ chối sai lệch**, không phải sinh thêm package hoặc service.

**Baseline:** ENG-GOV-01 tại `dev@8bb178b`. Tài liệu này chỉ bổ sung kiểm tra thay đổi source; không đổi schema, chạy migrations trên máy người dùng hay đụng volume.

## 1. Nguồn sự thật và phạm vi

| Nội dung | Nguồn sự thật | Cơ chế hiện hành |
| --- | --- | --- |
| Hành vi ứng dụng | Source trên `dev`, tests, SQL migrations | Review theo [Core Ownership Matrix](core-ownership.md) |
| Nguyên tắc làm việc | [Development Contract](development-contract.md) và [ADR-0001](decisions/0001-core-ownership.md) | Developer và reviewer thực thi trong PR |
| Thay đổi migration lịch sử | `db/migration/` trên base commit của PR | **CI tự kiểm tra** bằng `scripts/check-migration-history.sh` |
| Go/migration/runtime smoke | `.github/workflows/test.yml` | CI migrations, tests, build, Compose config, HTTP smoke, vet, race |
| WSL/volume/backup | Môi trường thực tế và [G0 runbook](../operations/g0-runtime-acceptance.md) | Nghiệm thu riêng; CI không thể kết luận thay |

Không lập `CODEOWNERS`, ruleset hoặc review team giả. **CI có thể báo đỏ nhưng không tự ngăn merge** nếu repository chưa cấu hình branch protection bắt buộc status check. Đó là việc quản trị GitHub cần thực hiện riêng bởi người có quyền Admin sau khi thống nhất ai được review.

## 2. Vòng làm việc chuẩn cho một thay đổi

```text
Source baseline (đã đọc) -> Problem / Intent
-> Capability + Data Owner -> Invariant
-> Branch nhỏ -> Implementation + tests
-> PR diff + CI -> Review -> Merge
-> Runtime gate riêng (nếu có tác động) -> Checkpoint
```

**Trước khi sửa:**

```sh
git fetch origin
git status --short --branch
git rev-parse --short HEAD
git rev-parse --short origin/dev
git rev-list --left-right --count HEAD...origin/dev
```

Nếu worktree bẩn, không `reset --hard`, checkout đè hay tự ý pull trước khi xác định thay đổi cần giữ. Không push trực tiếp vào `dev`. Tạo feature branch từ baseline đã xác minh và mô tả affected owner/invariant trong PR.

**Sau khi sửa, trước khi PR** (chạy trên môi trường có Go, Git, Docker Compose CLI; không cần khởi động container của ứng dụng cho cấu hình validation):

```sh
git diff --check
go test ./...
go build ./...
go vet ./...
docker compose config --quiet
```

Lưu ý: `go test ./...` chạy local bình thường có thể **skip integration test có ghi database**; nó không thay thế CI có DB disposable. Một số lệnh chỉ kiểm tra config, không chứng minh API đã vận hành trên WSL.

**Riêng khi sửa schema:** đã áp dụng/đã commit migration thì không sửa file cũ; thêm migration mới. So sánh branch của mình với `origin/dev` đã fetch:

```sh
sh scripts/check-migration-history.sh "$(git merge-base origin/dev HEAD)" HEAD
```

Script trên chỉ đọc các Git tree và exit khác 0 nếu một migration thuộc base bị sửa hoặc xóa; migration mới được phép. Dù pass, vẫn phải kiểm tra migration hợp lệ, tương thích với dữ liệu hiện có, constraint và rollback phù hợp. Chạy migration/test có ghi dữ liệu **chỉ trên DB disposable**, không dùng volume nghiệp vụ để thử.

## 3. Những gì CI cưỡng chế tự động

`pull_request`:

1. Fetch commit base và so sánh Git tree của PR với base.
2. **FAIL** nếu migration đã tồn tại trong base ở `db/migration/` bị modify, delete, rename (rename được xem như xóa đường dẫn cũ).
3. Cho phép migration mới; nội dung, thứ tự version và SQL correctness vẫn phụ thuộc migrate/tests/review.
4. Chạy toàn bộ CI hiện có để phát hiện regression.

`push` vào `dev`: giữ workflow test hiện có; không dùng migration guard PR vì không có PR base để so sánh.

Script không chạy SQL, Docker, network hoặc thao tác ghi trên working tree. Lệnh `git fetch` được gọi **bởi workflow trong GitHub runner**, không phải script; nó không thay đổi repository ở WSL của developer.

## 4. Cổng nghiệm thu — không được đánh đồng

| Gate | Chấp nhận khi | Không được tuyên bố |
| --- | --- | --- |
| GOV-01 — ownership | Owner và invariant xác định rõ, PR có người review thực tế | Tự động bắt buộc reviewer nếu chưa có ruleset |
| GOV-02 — source guard | CI migration guard và các bước test PASS | SQL cũ sẽ không bao giờ sai; có thể vẫn có lỗi concurrency |
| G0 — WSL runtime | Có kết quả local, log/migrations, volume, recovery phù hợp | CI runner bằng WSL thật |
| G1 — security | Test route-by-route + policy được nghiệm thu | Có WAF/hardening production chỉ nhờ middleware |
| Business Core | Domain contract, use case và invariant thực tế | Business Core đã tồn tại chỉ vì có Organization |

## 5. Ngoại lệ có chủ đích

Nếu thực sự phải sửa một migration lịch sử, **không bỏ qua gate bằng cách đổi CI**. Cần tài liệu phương án chuyển dữ liệu, lý do bất khả kháng, approval và môi trường chịu tác động. Mặc định giải pháp đúng là migration mới.

Không tăng số lượng gate nếu không có failure mode và bằng chứng cần bảo vệ. Khi phát hiện bất nhất mới giữa docs và source, cập nhật ma trận ownership và acceptance contract cùng với thay đổi tương ứng.
