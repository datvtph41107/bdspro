package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/templates"
)

func TestInfrastructureOpenAPI(t *testing.T) {
	contents, err := templates.Swagger.ReadFile("swagger/openapi.json")
	if err != nil { t.Fatal(err) }
	var spec struct {
		OpenAPI string `json:"openapi"`
		Paths map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(contents, &spec); err != nil { t.Fatal(err) }
	if spec.OpenAPI != "3.0.3" || len(spec.Paths) != 1 { t.Fatal("unexpected OpenAPI contract") }
	if _, ok := spec.Paths["/api/v1/health"]; !ok { t.Fatal("health path missing") }
}

func TestSwaggerDevelopmentOnly(t *testing.T) {
	for _, mode := range []string{"development", "production"} {
		r := web.NewRouter()
		RegisterSwagger(r, mode)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/swagger/openapi.json", nil))
		want := http.StatusNotFound
		if mode == "development" { want = http.StatusOK }
		if w.Code != want { t.Errorf("mode=%s status=%d want=%d", mode, w.Code, want) }
	}
}
