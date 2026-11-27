package routers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/modules/web/middleware"
	authrouter "github.com/datvtph41107/bdspro/routers/api/v1/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
)

func TestRequestLoggingErrorsAndPrivacy(t *testing.T) {
	previousLogger, previousMailer := zerologlog.Logger, setting.Mailer
	t.Cleanup(func() {
		zerologlog.Logger, setting.Mailer = previousLogger, previousMailer
	})
	setting.Mailer = setting.MailerSettings{}
	var output bytes.Buffer
	zerologlog.Logger = zerolog.New(&output)

	routes := web.NewRouter()
	routes.AfterRouting(appctx.APIContexter())
	// Exercise the actual registration handler without Redis admission.
	routes.Post("/register", func(ctx *appctx.APIContext) {
		form, err := middleware.BindJSON[structs.RegistrationStart](ctx)
		if err != nil {
			ctx.APIInvalid(err)
			return
		}
		web.SetForm(ctx, form)
	}, authrouter.RegisterStart)
	routes.Post("/forbidden", func(ctx *appctx.APIContext) {
		log.Ctx(ctx).Debug().Msg("context.logger.attached")
		ctx.APIError(http.StatusForbidden, "permission_denied", "browser origin verification failed")
	})
	routes.Post("/limited", func(ctx *appctx.APIContext) {
		ctx.Admit(ipactive.SMSDestinationCooldown, "subject", ipactive.Event{
			Allowed: false, Attempts: 0, Limit: 1, RetryAfter: 60, SourceID: "secret-destination-digest",
		}, nil)
	})
	routes.Post("/limiter-unavailable", func(ctx *appctx.APIContext) {
		ctx.Admit(ipactive.StartPhoneRegister, "ip", ipactive.Event{}, ipactive.ErrUnavailable)
	})
	routes.Get("/ok", func(ctx *appctx.APIContext) { ctx.Status(http.StatusNoContent) })
	handler := requestLogging(routes.WrapIngress())
	for _, tc := range []struct {
		path                   string
		status                 int
		level, message, reason string
	}{
		{"/register", 503, "error", "email delivery unavailable", "mailer_not_configured"},
		{"/forbidden", 403, "warn", "browser origin verification failed", ""},
		{"/ok", 204, "info", "", ""},
		{"/limited", 429, "warn", "too many requests", "rate_limit:subject:sms_destination_cooldown"},
		{"/limiter-unavailable", 503, "error", "request admission unavailable", "limiter_unavailable"},
		{"/missing", 404, "warn", "", ""},
	} {
		t.Run(tc.path, func(t *testing.T) {
			output.Reset()
			method := http.MethodPost
			if tc.path == "/ok" || tc.path == "/missing" {
				method = http.MethodGet
			}
			req := httptest.NewRequest(method, tc.path+"?token=secret-query",
				strings.NewReader(`{"email":"private@example.invalid"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer secret-bearer")
			req.Header.Set("Cookie", "session=secret-cookie")
			req.Header.Set("X-Request-ID", "untrusted-client-id")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != tc.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			id := response.Header().Get("X-Request-ID")
			if _, err := uuid.Parse(id); err != nil {
				t.Fatalf("missing server request ID: %q", id)
			}
			lines := strings.Split(strings.TrimSpace(output.String()), "\n")
			var event map[string]any
			if err := json.Unmarshal([]byte(lines[len(lines)-1]), &event); err != nil {
				t.Fatal(err)
			}
			if event["request_id"] != id || event["status"] != float64(tc.status) || event["level"] != tc.level || event["path"] != tc.path {
				t.Fatalf("incorrect event: %v", event)
			}
			if tc.message != "" && event["error"] != tc.message {
				t.Fatalf("missing public error: %v", event)
			}
			if tc.reason != "" && event["reason"] != tc.reason {
				t.Fatalf("missing reason: %v", event)
			}
			if tc.path == "/limited" {
				if response.Header().Get("Retry-After") != "60" || !strings.Contains(output.String(), "admission.denied") || !strings.Contains(output.String(), "sms_destination_cooldown") {
					t.Fatal("rate limit is not actionable from response/log")
				}
			}
			if tc.path == "/forbidden" && !strings.Contains(output.String(), "context.logger.attached") {
				t.Fatal("request context logger is disabled")
			}
			for _, secret := range []string{"secret-query", "secret-bearer", "secret-cookie", "private@example.invalid", "untrusted-client-id", "secret-destination-digest"} {
				if strings.Contains(output.String(), secret) {
					t.Fatalf("sensitive data logged: %s", secret)
				}
			}
		})
	}
}
