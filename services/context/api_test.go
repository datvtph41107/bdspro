package context

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/gin-gonic/gin"
)

func TestWrapPreservesGinAbortAndPublicError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := web.NewRouter()
	r.AfterRouting(APIContexter())
	reached := false
	r.Get("/test", func(ctx *APIContext) {
		ctx.APIErrorAuto(errors.New("private-driver-secret"))
		if !ctx.IsAborted() {
			t.Fatal("public error must abort pending Gin handlers")
		}
	}, func(*APIContext) { reached = true })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))
	if reached {
		t.Fatal("Gin executed handler after API error")
	}
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d want=500", w.Code)
	}
	var body struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "internal_error" || body.Error != "internal server error" {
		t.Fatalf("private error leaked or contract changed: %+v", body)
	}
}
