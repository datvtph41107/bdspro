# Architecture evidence — 50 GitHub repositories

Checked: 2026-10-28. Scope: Go/Gin REST APIs, Laravel/PHP conventions, and Java Spring applications.

## Method and limits

The **50 repositories below were checked against GitHub repository metadata** to
confirm their identity and existence. This does **not** mean each repository
received a full code audit. Deep conclusions below are grounded in **specific
source files actually read** in selected representative repositories. Metadata-only
entries are a breadth inventory, not evidence for a BDSPro architectural rule.

"Laravel DRX" is not a uniquely verified public repository from this name
alone. Do not attribute Laravel framework behavior to a DRX project without
a concrete repository URL and revision.

## Source files read and conclusions

| Source | File | Technical observation | Relevance to BDSPro |
| --- | --- | --- | --- |
| SimpleBank (Go) | [api/server.go](https://github.com/techschool/simplebank/blob/master/api/server.go) | Gin routes call handlers directly | Prefer native Gin registration |
| SimpleBank (Go) | [db/sqlc/exec_tx.go](https://github.com/techschool/simplebank/blob/master/db/sqlc/exec_tx.go) | One explicit SQLC transaction helper | Keep execTx for atomic operations |
| Apache Answer (Go) | [question_controller.go](https://github.com/apache/answer/blob/main/internal/controller/question_controller.go) | BindAndCheck / HandleResponse centralize application-specific validation and output | Do not copy unless extra application contract justifies it |
| Apache Answer (Go) | [question_service.go](https://github.com/apache/answer/blob/main/internal/service/content/question_service.go) | Service coordinates repository and related operations | Service for actual business orchestration |
| Gotify (Go) | [api/application.go](https://github.com/gotify/server/blob/master/api/application.go) | Handler directly calls database for simple operation | Service not obligatory for every endpoint |
| Gogs (Go) | [web.go](https://github.com/gogs/gogs/blob/main/cmd/gogs/internal/web/web.go) | Binding occurs in framework route machinery | Avoid inventing custom Gin binding framework |
| Gogs (Go) | [setting.go](https://github.com/gogs/gogs/blob/main/internal/route/user/setting.go) | Explicit database errors at route handler | Map domain-specific errors at operation owner |
| Fider (Go) | [apiv1/post.go](https://github.com/getfider/fider/blob/main/app/handlers/apiv1/post.go) | Command bus coordinates post + attachment + vote + notification operations | Extra indirection has a real cost; not needed by default |
| Coder (Go) | [coderd/workspaces.go](https://github.com/coder/coder/blob/main/coderd/workspaces.go) | httpapi.Read and httperror.WriteResponseError consolidate a much larger HTTP contract | BDSPro should consolidate only known repeated rules |
| Coder (Go) | [workspaces.sql](https://github.com/coder/coder/blob/main/coderd/database/queries/workspaces.sql) | Query definitions are maintained in SQL | Preserve SQLC query source of truth |
| BookStack (Laravel) | [routes/api.php](https://github.com/BookStackApp/BookStack/blob/development/routes/api.php) | Direct route-controller declaration | Route-first discoverability |
| BookStack (Laravel) | [PageApiController.php](https://github.com/BookStackApp/BookStack/blob/development/app/Entities/Controllers/PageApiController.php) | Controller uses repository directly, with local catches for specific failures | Service is not a required layer |
| BookStack (Laravel) | [PageRepo.php](https://github.com/BookStackApp/BookStack/blob/development/app/Entities/Repos/PageRepo.php) | Publishing a draft is a unit of database work | Protect true invariants with transactions |
| Laravel framework | [FormRequest.php](https://github.com/laravel/framework/blob/13.x/src/Illuminate/Foundation/Http/FormRequest.php) | Framework-owned request validation on resolution | Gin does not provide the same container lifecycle; stay explicit |
| Laravel framework | [Handler.php](https://github.com/laravel/framework/blob/13.x/src/Illuminate/Foundation/Exceptions/Handler.php) | Application-wide exception rendering | One BDSPro mapping for public app errors |
| Laravel framework | [ManagesTransactions.php](https://github.com/laravel/framework/blob/13.x/src/Illuminate/Database/Concerns/ManagesTransactions.php) | Transaction wrapper owns rollback/commit | Keep db.execTx() |
| Spring PetClinic | [OwnerController.java](https://github.com/spring-projects/spring-petclinic/blob/main/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java) | Spring MVC controller injects and calls OwnerRepository directly | Spring does not force Service for trivial operations |
| Spring PetClinic | [OwnerRepository.java](https://github.com/spring-projects/spring-petclinic/blob/main/src/main/java/org/springframework/samples/petclinic/owner/OwnerRepository.java) | JpaRepository defines persistence methods | SQLC is BDSPro's persistence interface, not Eloquent/JPA |
| Spring Framework | [ExceptionHandlerExceptionResolver.java](https://github.com/spring-projects/spring-framework/blob/main/spring-webmvc/src/main/java/org/springframework/web/servlet/mvc/method/annotation/ExceptionHandlerExceptionResolver.java) | Exceptions resolved by Spring MVC infrastructure | Central HTTP error translation is framework-specific |
| Spring Docs | [Declarative transactions](https://docs.spring.io/spring-framework/reference/data-access/transaction/declarative/tx-decl-explained.html) | Transaction interceptors and proxies are framework infrastructure | No hidden transaction magic in BDSPro |

## Verified GitHub inventory (50)

### 20 Go / Go-centric infrastructure projects

| # | Repository | Review scope |
| --- | --- | --- |
| 1 | [gin-gonic/gin](https://github.com/gin-gonic/gin) | Verified repository identity |
| 2 | [techschool/simplebank](https://github.com/techschool/simplebank) | Verified repository identity |
| 3 | [apache/answer](https://github.com/apache/answer) | Verified repository identity |
| 4 | [gotify/server](https://github.com/gotify/server) | Verified repository identity |
| 5 | [gogs/gogs](https://github.com/gogs/gogs) | Verified repository identity |
| 6 | [go-gitea/gitea](https://github.com/go-gitea/gitea) | Verified repository identity |
| 7 | [coder/coder](https://github.com/coder/coder) | Verified repository identity |
| 8 | [getfider/fider](https://github.com/getfider/fider) | Verified repository identity |
| 9 | [owncast/owncast](https://github.com/owncast/owncast) | Verified repository identity |
| 10 | [syncthing/syncthing](https://github.com/syncthing/syncthing) | Verified repository identity |
| 11 | [grafana/grafana](https://github.com/grafana/grafana) | Verified repository identity |
| 12 | [prometheus/prometheus](https://github.com/prometheus/prometheus) | Verified repository identity |
| 13 | [caddyserver/caddy](https://github.com/caddyserver/caddy) | Verified repository identity |
| 14 | [minio/minio](https://github.com/minio/minio) | Verified repository identity |
| 15 | [pocketbase/pocketbase](https://github.com/pocketbase/pocketbase) | Verified repository identity |
| 16 | [mattermost/mattermost](https://github.com/mattermost/mattermost) | Verified repository identity |
| 17 | [portainer/portainer](https://github.com/portainer/portainer) | Verified repository identity |
| 18 | [restic/restic](https://github.com/restic/restic) | Verified repository identity |
| 19 | [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes) | Verified repository identity |
| 20 | [hashicorp/consul](https://github.com/hashicorp/consul) | Verified repository identity |

### 15 Laravel / PHP repositories

| # | Repository | Review scope |
| --- | --- | --- |
| 1 | [laravel/framework](https://github.com/laravel/framework) | Verified repository identity |
| 2 | [laravel/laravel](https://github.com/laravel/laravel) | Verified repository identity |
| 3 | [laravel/sanctum](https://github.com/laravel/sanctum) | Verified repository identity |
| 4 | [laravel/passport](https://github.com/laravel/passport) | Verified repository identity |
| 5 | [laravel/horizon](https://github.com/laravel/horizon) | Verified repository identity |
| 6 | [laravel/telescope](https://github.com/laravel/telescope) | Verified repository identity |
| 7 | [laravel/fortify](https://github.com/laravel/fortify) | Verified repository identity |
| 8 | [BookStackApp/BookStack](https://github.com/BookStackApp/BookStack) | Verified repository identity |
| 9 | [bagisto/bagisto](https://github.com/bagisto/bagisto) | Verified repository identity |
| 10 | [akaunting/akaunting](https://github.com/akaunting/akaunting) | Verified repository identity |
| 11 | [monicahq/monica](https://github.com/monicahq/monica) | Verified repository identity |
| 12 | [filamentphp/filament](https://github.com/filamentphp/filament) | Verified repository identity |
| 13 | [invoiceninja/invoiceninja](https://github.com/invoiceninja/invoiceninja) | Verified repository identity |
| 14 | [laravelio/laravel.io](https://github.com/laravelio/laravel.io) | Verified repository identity |
| 15 | [spatie/laravel-permission](https://github.com/spatie/laravel-permission) | Verified repository identity |

### 15 Spring / Java repositories

| # | Repository | Review scope |
| --- | --- | --- |
| 1 | [spring-projects/spring-framework](https://github.com/spring-projects/spring-framework) | Verified repository identity |
| 2 | [spring-projects/spring-boot](https://github.com/spring-projects/spring-boot) | Verified repository identity |
| 3 | [spring-projects/spring-security](https://github.com/spring-projects/spring-security) | Verified repository identity |
| 4 | [spring-projects/spring-data-jpa](https://github.com/spring-projects/spring-data-jpa) | Verified repository identity |
| 5 | [spring-projects/spring-petclinic](https://github.com/spring-projects/spring-petclinic) | Verified repository identity |
| 6 | [spring-projects/spring-batch](https://github.com/spring-projects/spring-batch) | Verified repository identity |
| 7 | [spring-projects/spring-integration](https://github.com/spring-projects/spring-integration) | Verified repository identity |
| 8 | [spring-projects/spring-session](https://github.com/spring-projects/spring-session) | Verified repository identity |
| 9 | [spring-projects/spring-hateoas](https://github.com/spring-projects/spring-hateoas) | Verified repository identity |
| 10 | [spring-projects/spring-modulith](https://github.com/spring-projects/spring-modulith) | Verified repository identity |
| 11 | [spring-cloud/spring-cloud-gateway](https://github.com/spring-cloud/spring-cloud-gateway) | Verified repository identity |
| 12 | [eugenp/tutorials](https://github.com/eugenp/tutorials) | Verified repository identity |
| 13 | [spring-projects/spring-data-commons](https://github.com/spring-projects/spring-data-commons) | Verified repository identity |
| 14 | [spring-projects/spring-restdocs](https://github.com/spring-projects/spring-restdocs) | Verified repository identity |
| 15 | [spring-projects/spring-amqp](https://github.com/spring-projects/spring-amqp) | Verified repository identity |

## Evidence-to-change trace

| Source evidence | Existing BDSPro behavior | Implemented design |
| --- | --- | --- |
| SimpleBank native Gin routing | routes.Handlers / routes.Policies duplicated fields | One routes.go calling existing handlers directly |
| Go error-as-value + Laravel/Spring public error mapping | response.Send had five mixed success/error parameters | Gin for success; response.Invalid and response.Error for two nontrivial mappings |
| BookStack transaction and SimpleBank execTx | CreateOrganizationTx already atomic | Retain transaction, add rollback integration regression |
| PetClinic/Gotify simple persistence path | Unnecessary interfaces considered | No mandatory Repository wrapper after SQLC |
| Coder/Apache Answer non-public error handling | Some HTTP paths returned local database errors | Known errors are explicit and unknown failures are safe generic HTTP 500 |
| Gin validation vs bcrypt 72 bytes | RegisterBody counts runes while bcrypt limits bytes | AuthService validates 8 runes minimum and 72 bytes maximum |

## Remaining evidence gap

Registration now records a durable outbox row in the same PostgreSQL
transaction as the account and dispatches it after commit. This resolves the
early-worker race and provides retry after a Redis outage. Asynq TaskID limits
duplicate *queued* tasks; the Redis/SMTP boundary still has at-least-once
semantics and does not guarantee exactly-once email delivery.
