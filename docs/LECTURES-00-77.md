# Đối chiếu 78 lecture TechSchool với BDSPro

Mốc: [TechSchool SimpleBank README](https://github.com/techschool/simplebank/blob/97f000fe58ad01a0774179ffa8884ac7784cf263/README.md). Trạng thái ghi đúng những gì có trong BDSPro, không nhận đã xong toàn bộ nếu chưa có code/test hay hạ tầng AWS.

| # | Lecture gốc | BDSPro |
|---:|---|---|
| 0 | Setup development environment on Windows: WSL2 + Go + VSCode + Docker + Make + Sqlc | Setup người dùng |
| 1 | Design DB schema and generate SQL code with dbdiagram.io | Một phần |
| 2 | Install & use Docker + Postgres + TablePlus to create DB schema | Một phần |
| 3 | How to write & run database migration in Golang | Một phần |
| 4 | Generate CRUD Golang code from SQL / Compare db/sql, gorm, sqlx & sqlc | Một phần |
| 5 | Write unit tests for database CRUD with random data in Golang | Một phần |
| 6 | A clean way to implement database transaction in Golang | Một phần |
| 7 | DB transaction lock & How to handle deadlock in Golang | Một phần |
| 8 | How to avoid deadlock in DB transaction? Queries order matters! | Một phần |
| 9 | Deeply understand transaction isolation levels & read phenomena in MySQL & PostgreSQL | Một phần |
| 10 | Setup Github Actions for Golang + Postgres to run automated tests | Một phần |
| 11 | Implement RESTful HTTP API in Go using Gin | Một phần |
| 12 | Load config from file & environment variables in Go with Viper | Một phần |
| 13 | Mock DB for testing HTTP API in Go and achieve 100% coverage | Một phần |
| 14 | Implement transfer money API with a custom params validator | Một phần |
| 15 | Add users table with unique & foreign key constraints in PostgreSQL | Một phần |
| 16 | How to handle DB errors in Golang correctly | Một phần |
| 17 | How to securely store passwords? Hash password in Go with Bcrypt! | Một phần |
| 18 | How to write stronger unit tests with a custom gomock matcher | Một phần |
| 19 | Why PASETO is better than JWT for token-based authentication? | Một phần |
| 20 | How to create and verify JWT & PASETO token in Golang | Một phần |
| 21 | Implement login user API that returns PASETO or JWT access token in Go | Một phần |
| 22 | Implement authentication middleware and authorization rules in Golang using Gin | Một phần |
| 23 | Build a minimal Golang Docker image with a multistage Dockerfile | Docker/Compose nền tảng |
| 24 | How to use docker network to connect 2 stand-alone containers | Docker/Compose nền tảng |
| 25 | How to write docker-compose file and control service start-up orders with wait-for.sh | Docker/Compose nền tảng |
| 26 | How to create a free tier AWS account | AWS chưa triển khai |
| 27 | Auto build & push docker image to AWS ECR with Github Actions | AWS chưa triển khai |
| 28 | How to create a production DB on AWS RDS | AWS chưa triển khai |
| 29 | Store & retrieve production secrets with AWS secrets manager | AWS chưa triển khai |
| 30 | Kubernetes architecture & How to create an EKS cluster on AWS | AWS chưa triển khai |
| 31 | How to use kubectl & k9s to connect to a kubernetes cluster on AWS EKS | AWS chưa triển khai |
| 32 | How to deploy a web app to Kubernetes cluster on AWS EKS | AWS chưa triển khai |
| 33 | Register a domain name & set up A-record using Route53 | AWS chưa triển khai |
| 34 | How to use Ingress to route traffics to different services in Kubernetes | AWS chưa triển khai |
| 35 | Automatic issue TLS certificates in Kubernetes with Let's Encrypt | AWS chưa triển khai |
| 36 | Automatic deploy to Kubernetes with Github Action | AWS chưa triển khai |
| 37 | How to manage user session with refresh token - Golang | Session hiện có |
| 38 | Generate DB documentation page and schema SQL dump from DBML | Đã có DBML và mô tả schema; chưa xuất bản dbdocs |
| 39 | Introduction to gRPC | Đã có gRPC server |
| 40 | Define gRPC API and generate Go code with protobuf | Đã có Proto contract và generated Go code |
| 41 | How to run a golang gRPC server and call its API | Đã có listener gRPC cổng 9090 |
| 42 | Implement gRPC API to create and login users in Go | Đã có RPC Register/Login + test |
| 43 | gRPC gateway: write code once, serve both gRPC & HTTP requests | Đã có HTTP Gateway /v1 |
| 44 | How to extract info from gRPC metadata | Có đọc metadata user-agent, x-forwarded-for |
| 45 | Automatic generate & serve Swagger docs from Go server | Đã sinh OpenAPI và phục vụ /docs/openapi.json |
| 46 | Embed static frontend files inside Golang backend server's binary | Đã embed OpenAPI JSON, chưa embed frontend |
| 47 | Validate gRPC parameters and send human/machine friendly response | Đã có status codes và kiểm tra input tối thiểu |
| 48 | Run DB migrations directly inside Golang code | Compose áp dụng migration qua service migrate; chưa migrate trong Go |
| 49 | Partial update DB record with SQLC nullable parameters | SQLC narg + owner-only UPDATE Organization |
| 50 | Build gRPC update API with optional parameters | gRPC UpdateOrganization + Gateway PATCH |
| 51 | Add authorization to protect gRPC API | JWT metadata + owner authorization trong SQL |
| 52 | Write structured logs for gRPC APIs | Đã có structured gRPC logging |
| 53 | How to write HTTP logger middleware in Go | Đã có HTTP logger middleware |
| 54 | Implement background worker in Go with Redis and Asynq | Đã có Redis/Asynq worker |
| 55 | Integrate async worker to Go web server | Đã tích hợp worker vào main.go |
| 56 | Send async tasks to Redis within a DB transaction | Đã enqueue trong callback RegisterTx |
| 57 | How to handle errors and print logs for Go Asynq workers | Có worker error logger |
| 58 | A bit of delay might be good for your async tasks | Có ProcessIn 10 giây |
| 59 | How to send emails in Go via Gmail | Đã có Gmail sender, cần SMTP credentials thật |
| 60 | How to skip test in Go and config test flag in vscode | Integration test DB skip local, chạy CI |
| 61 | Email verification in Go: design DB and send email | Đã có verify_emails và worker |
| 62 | Implement email verification API in Go | Đã có VerifyEmail API Gin/gRPC |
| 63 | Unit test gRPC API with mock DB & Redis | Đã có mock DB/worker test, chưa gRPC multi-mock tổng quát |
| 64 | How to test a gRPC API that requires authentication | Chưa có gRPC authenticated update test |
| 65 | Config sqlc version 2 for Go and Postgres | Đã có sqlc v2 |
| 66 | Switch DB driver from lib/pq to pgx | Đã có pgx/v5 |
| 67 | How to handle DB errors with PGX driver | Có xử lý lỗi pgconn/pgx |
| 68 | Docker compose: port + volume mapping | Đã có Compose port+volume |
| 69 | How to install & use binary packages in Go | Tooling |
| 70 | Implement role-based access control (RBAC) in Go | Có membership owner/member qua Gin + SQL |
| 71 | Grant AWS EKS cluster access to Postgres and Redis using security group | AWS chưa triển khai |
| 72 | Deploy gRPC + HTTP server to AWS EKS cluster | AWS chưa triển khai |
| 73 | Don't lose money on AWS | AWS chưa triển khai |
| 74 | Graceful shutdown gRPC/HTTP servers and async worker | Đã có graceful HTTP/gRPC/Asynq |
| 75 | Go 1.22 fixed the most common for-loop trap | Go mới |
| 76 | Setup CORS policy with Go and VueJS | Đã có CORS middleware |
| 77 | Upgrade golang JWT package to v5 | Đã có JWT v5 |
