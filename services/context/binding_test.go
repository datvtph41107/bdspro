package context

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web"
)

func TestTypedJSONBindingStopsOnMalformedInput(t *testing.T) {
	type createOption struct {
		Name string `json:"name"`
	}
	router := web.NewRouter()
	router.AfterRouting(APIContexter())
	reached := 0
	router.Post("/items",
		BindJSON[createOption](),
		func(ctx *APIContext) {
			reached++
			form := web.GetForm[*createOption](ctx)
			ctx.JSON(http.StatusOK, map[string]string{"name": form.Name})
		},
	)
	tests := []struct {
		body    string
		status  int
		key     string
		value   string
		reached int
	}{
		{body: `{"name":"Backend"}`, status: http.StatusOK, key: "name", value: "Backend", reached: 1},
		{body: `{"name":"A"} {"name":"B"}`, status: http.StatusBadRequest, key: "code", value: "invalid_request", reached: 1},
		{body: `{"name":`, status: http.StatusBadRequest, key: "code", value: "invalid_request", reached: 1},
		{body: `{"name":"` + strings.Repeat("x", int(maxJSONBodyBytes)) + `"}`, status: http.StatusRequestEntityTooLarge, key: "code", value: "request_too_large", reached: 1},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(tt.body)))
		if w.Code != tt.status {
			t.Fatalf("body-size=%d status=%d want=%d response=%s", len(tt.body), w.Code, tt.status, w.Body.String())
		}
		var body map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body[tt.key] != tt.value {
			t.Fatalf("body-size=%d response=%v want %s=%s", len(tt.body), body, tt.key, tt.value)
		}
		if reached != tt.reached {
			t.Fatalf("handler reached %d times; expected %d", reached, tt.reached)
		}
	}
}
