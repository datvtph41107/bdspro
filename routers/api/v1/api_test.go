package apiv1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInfrastructureHealth(t *testing.T) {
	r := Routes()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want=200", w.Code)
	}
	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["status"] != "ok" {
		t.Fatalf("unexpected response: %v", response)
	}
}
