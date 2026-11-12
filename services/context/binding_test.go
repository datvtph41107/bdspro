package context

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web"
)

func TestTypedJSONBindingUsesSameRequestContext(t *testing.T) {
	type createOption struct { Name string `json:"name"` }
	router := web.NewRouter()
	router.AfterRouting(APIContexter())
	router.Post("/items",
		web.BindJSON[createOption](),
		func(ctx *APIContext) {
			if err := web.BindingError(ctx); err != nil {
				ctx.APIError(http.StatusBadRequest, "invalid request")
				return
			}
			form := web.GetForm[*createOption](ctx)
			ctx.JSON(http.StatusOK, map[string]string{"name": form.Name})
		},
	)
	for _, tc := range []struct{ body string; status int; key, value string }{
		{body: `{"name":"Backend"}`, status: http.StatusOK, key: "name", value: "Backend"},
		{body: `{"name":"A"} {"name":"B"}`, status: http.StatusBadRequest, key: "code", value: "invalid_request"},
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(tc.body)))
		if w.Code != tc.status {
			t.Fatalf("body=%q status=%d want=%d response=%s", tc.body, w.Code, tc.status, w.Body.String())
		}
		var body map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body[tc.key] != tc.value {
			t.Fatalf("body=%q response=%v want %s=%s", tc.body, body, tc.key, tc.value)
		}
	}
}
