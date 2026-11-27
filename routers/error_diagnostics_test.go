package routers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
)

func TestAPIErrorDiagnosticsPreserveResponseAndPrivacy(t *testing.T) {
	previous := zerologlog.Logger
	t.Cleanup(func() { zerologlog.Logger = previous })
	var output bytes.Buffer
	zerologlog.Logger = zerolog.New(&output).Level(zerolog.DebugLevel)
	routes := web.NewRouter()
	routes.AfterRouting(appctx.APIContexter())
	routes.Get("/internal", func(ctx *appctx.APIContext) {
		err := util.WithOperation("auth.session.create", &pgconn.PgError{
			Code: "23505", Message: "secret-message", Detail: "secret-detail",
		})
		ctx.APIErrorAuto(fmt.Errorf("secret-wrapper: %w", err))
	})
	routes.Get("/public", func(ctx *appctx.APIContext) {
		err := util.WithPublic(util.ErrUnauthenticated, "secret-model", "invalid_credentials", "invalid credentials")
		ctx.APIErrorAuto(util.WithOperation("auth.login.transaction", err))
	})
	handler := requestLogging(routes.WrapIngress())
	for _, tc := range []struct {
		path, code, message string
		status              int
	}{
		{"/internal", "internal_error", "internal server error", 500},
		{"/public", "invalid_credentials", "invalid credentials", 401},
	} {
		output.Reset()
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		var response appctx.APIErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if w.Code != tc.status || response.Code != tc.code || response.Error != tc.message {
			t.Fatalf("changed public error: %d %s", w.Code, w.Body.String())
		}
		count := 0
		for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
			var event map[string]any
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				t.Fatal(err)
			}
			if event["message"] != "http.application_error" {
				continue
			}
			count++
			if event["request_id"] != w.Header().Get("X-Request-ID") || event["error_chain"] == nil || event["stack_frames"] == nil || !strings.Contains(line, "23505") {
				t.Fatalf("incomplete diagnostics: %s", line)
			}
		}
		if (tc.status == 500 && count != 1) || (tc.status == 401 && count != 0) {
			t.Fatalf("unexpected internal diagnostic count %d", count)
		}
		if strings.Contains(output.String(), "secret-") || strings.Contains(w.Body.String(), "secret-") {
			t.Fatal("private error data exposed")
		}
	}
}
