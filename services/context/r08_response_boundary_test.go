package context

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/gin-gonic/gin"
)

func TestR08JSONCommitIsSingleResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := web.NewRouter()
	router.AfterRouting(APIContexter())
	router.Get("/one", func(api *APIContext) {
		api.JSON(http.StatusCreated, map[string]string{"value": "first"})
		if !api.Written() {
			t.Fatal("JSON response did not commit")
		}
		api.JSON(http.StatusAccepted, map[string]string{"value": "second"})
		api.Status(http.StatusConflict)
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/one", nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("response status=%d, want=201", rec.Code)
	}
	var data map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil || data["value"] != "first" {
		t.Fatalf("body=%s error=%v", rec.Body.String(), err)
	}
	if strings.Contains(rec.Body.String(), "second") {
		t.Fatal("second JSON was appended")
	}
}

func TestR08ErrorAfterCommittedResponseAbortsWithoutOverwriting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := web.NewRouter()
	router.AfterRouting(APIContexter())
	reached := false
	router.Get("/one", func(api *APIContext) {
		api.JSON(http.StatusOK, map[string]string{"status": "done"})
		api.APIErrorAuto(errors.New("private-driver-message"))
		if !api.IsAborted() {
			t.Fatal("late API error did not abort")
		}
	}, func(*APIContext) {
		reached = true
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/one", nil))
	if reached || rec.Code != http.StatusOK {
		t.Fatalf("endpoint continuation=%t status=%d", reached, rec.Code)
	}
	if strings.Contains(rec.Body.String(), "internal_error") || strings.Contains(rec.Body.String(), "private-driver-message") {
		t.Fatalf("late error polluted success response: %s", rec.Body.String())
	}
	var data map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil || data["status"] != "done" {
		t.Fatalf("response=%s error=%v", rec.Body.String(), err)
	}
}

func TestR08JSONEncodingErrorReturnsSafe500AndStopsChain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := web.NewRouter()
	router.AfterRouting(APIContexter())
	reached := false
	router.Get("/fail-encode", func(api *APIContext) {
		api.JSON(http.StatusOK, struct{
			Unsupported chan int `json:"unsupported"`
		}{Unsupported: make(chan int)})
	}, func(*APIContext) {
		reached = true
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fail-encode", nil))
	if reached || rec.Code != http.StatusInternalServerError {
		t.Fatalf("continued=%t status=%d", reached, rec.Code)
	}
	var data struct {
		Code string `json:"code"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil || data.Code != "internal_error" || data.Error != "internal server error" {
		t.Fatalf("unexpected failed-encode response: body=%s err=%v", rec.Body.String(), err)
	}
}
