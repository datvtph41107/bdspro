# BDSPro — Source Architecture

**Source:** `refactor/explicit-routes-cmd-20261026`.

## Lý do học hỏi Laravel DRX

DRX tạo phản xạ tốt cho developer vì route là mục lục hành vi: nhìn tên URL, mở action tương ứng, biết request input, service và dữ liệu từ đâu. Điểm cốt lõi là **phân trách nhiệm và một cách viết lặp lại được**, không phải các facade, magic method, ORM hoặc annotation của PHP.

BDSPro hiện dùng Gin để khai báo trực tiếp routes; controller nhận biến Body/Query đã khai báo rõ và gọi hàm parse một lần. Go không có cơ chế đặt `Route::post(...)` trực tiếp ở package top level; vì vậy các lời gọi Gin được đặt trong các hàm đăng ký rõ ràng. Không giải quyết khác biệt cú pháp bằng registry, reflection hoặc DSL.

## Trách nhiệm chuẩn

- [Route, Request, Middleware, Controller và Response](http.md)
- [SQLC, Store và transaction](data.md)
- [Điểm chạy, PostgreSQL, Redis, HTTP server và worker](runtime.md)
- [Công thức viết một API](workflow.md)
- [Trạng thái và giới hạn nghiệm thu](status.md)

Các package có đường phụ thuộc rõ ràng: `routes` biết controller methods; controller biết request, resources, service; service biết truy vấn; mã SQL nằm tại `db/query`. Không đưa thao tác PostgreSQL sang Controller chỉ để giảm số file.

Không tạo Admin, Facade, CacheManager, BaseRepository, FilterBuilder hoặc các điểm mở rộng rỗng khi chưa có consumer thực tế.
