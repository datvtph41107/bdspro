package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/templates"
	"github.com/datvtph41107/bdspro/modules/web"
)

func TestInfrastructureOpenAPI(t *testing.T) {
	contents, err := templates.Swagger.ReadFile("swagger/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		OpenAPI string                     `json:"openapi"`
		Paths   map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(contents, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.OpenAPI != "3.0.3" || len(spec.Paths) != 1 {
		t.Fatalf("unexpected API contract: version=%q paths=%d", spec.OpenAPI, len(spec.Paths))
	}
	if _, ok := spec.Paths["/api/v1/health"]; !ok {
		t.Fatal("infrastructure health route missing from OpenAPI")
	}
}

func TestSwaggerDevelopmentOnly(t *testing.T) {
	for _, mode := range []string{"development", "production"} {
		mux := http.NewServeMux()
		if swagger := SwaggerHandler(mode); swagger != nil {
			mux.Handle("/swagger/", swagger)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/swagger/openapi.json", nil))
		want := http.StatusNotFound
		if mode == "development" {
			want = http.StatusOK
		}
		if w.Code != want {
			t.Errorf("mode=%s status=%d want=%d", mode, w.Code, want)
		}
	}
}

func TestSwaggerIndexAndMissingAsset(t *testing.T) {
	swagger := SwaggerHandler("development")
	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/swagger/index.html", http.StatusOK},
		{"/swagger/missing.json", http.StatusNotFound},
	} {
		w := httptest.NewRecorder()
		swagger.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if w.Code != tc.status {
			t.Errorf("path=%s status=%d want=%d", tc.path, w.Code, tc.status)
		}
	}
}

func TestSwaggerRoutedThroughApplicationRouter(t *testing.T) {
	router := web.NewRouter()
	router.Get("/swagger/*filepath", SwaggerHandler("development").ServeHTTP)

	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/swagger/index.html", http.StatusOK},
		{"/swagger/openapi.json", http.StatusOK},
		{"/swagger/missing.json", http.StatusNotFound},
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != tc.status {
			t.Errorf("GET %s: status=%d want=%d", tc.path, rec.Code, tc.status)
		}
	}
}
