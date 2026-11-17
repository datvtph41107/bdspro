package apiv1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

type r05Form struct {
	Name string `json:"name" binding:"required"`
	Age  int    `json:"age"`
}

func newR05Router(endpoint func(*appctx.APIContext)) *web.Router {
	r := web.NewRouter()
	r.AfterRouting(appctx.APIContexter())
	r.Post("/form", bind(r05Form{}), endpoint)
	return r
}

func TestR05APIBindPublishesTypedFormBeforeEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	calls := 0
	r := newR05Router(func(ctx *appctx.APIContext) {
		calls++
		form := web.GetForm[*r05Form](ctx)
		if form.Name != "Backend" || form.Age != 4 {
			t.Errorf("bound form=%+v", form)
		}
		ctx.JSON(http.StatusCreated, map[string]string{"name": form.Name})
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/form", bytes.NewBufferString(`{"name":"Backend","age":4}`)))
	if calls != 1 || rec.Code != http.StatusCreated {
		t.Fatalf("calls=%d status=%d body=%s", calls, rec.Code, rec.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil || payload["name"] != "Backend" {
		t.Fatalf("payload=%v error=%v", payload, err)
	}
}

func TestR05APIBindFailuresPreventEndpointExecution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, body, code string
	}{
		{"missing required field", `{"age":1}`, "validation_failed"},
		{"malformed JSON", `{"name":`, "malformed_json"},
		{"empty body", "", "request_body_required"},
		{"wrong type", `{"name":12}`, "validation_failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			r := newR05Router(func(ctx *appctx.APIContext) {
				calls++
				ctx.Status(http.StatusNoContent)
			})
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/form", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest || calls != 0 {
				t.Fatalf("status=%d calls=%d body=%s", rec.Code, calls, rec.Body.String())
			}
			var payload struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("invalid error JSON: %v body=%s", err, rec.Body.String())
			}
			if payload.Code != tc.code {
				t.Errorf("code=%q want=%q", payload.Code, tc.code)
			}
		})
	}
}
