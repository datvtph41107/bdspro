package routers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/routers/common"
	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

func TestRecoveryBeforeResponseWritesOnePublicError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := web.NewRouter()
	m.AfterRouting(appctx.APIContexter())
	router := m.GinEngine()
	router.Use(common.Recovery(io.Discard))
	m.Get("/panic", func(*appctx.APIContext) { panic("private provider error") })
	record := httptest.NewRecorder()
	m.ServeHTTP(record, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if record.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", record.Code)
	}
	var response struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if err := json.Unmarshal(record.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid response: %v body=%q", err, record.Body.String())
	}
	if response.Code != "internal_error" || response.Error != "internal server error" {
		t.Fatalf("unexpected public response=%+v", response)
	}
	if bytes.Contains(record.Body.Bytes(), []byte("private provider error")) {
		t.Fatal("raw panic detail exposed in HTTP response")
	}
}

func TestRecoveryAfterCommittedResponseDoesNotAppendJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := web.NewRouter()
	m.AfterRouting(appctx.APIContexter())
	router := m.GinEngine()
	router.Use(common.Recovery(io.Discard))
	m.Get("/panic", func(ctx *appctx.APIContext) {
		ctx.JSON(http.StatusAccepted, map[string]string{"result": "accepted"})
		panic("private provider error")
	})
	record := httptest.NewRecorder()
	m.ServeHTTP(record, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if record.Code != http.StatusAccepted {
		t.Fatalf("committed status changed to %d", record.Code)
	}
	var response map[string]string
	if err := json.Unmarshal(record.Body.Bytes(), &response); err != nil {
		t.Fatalf("recovery appended a second response: %v body=%q", err, record.Body.String())
	}
	if len(response) != 1 || response["result"] != "accepted" {
		t.Fatalf("response was changed after commit: %v", response)
	}
	if bytes.Contains(record.Body.Bytes(), []byte("private provider error")) {
		t.Fatal("raw panic detail exposed in HTTP response")
	}
}
