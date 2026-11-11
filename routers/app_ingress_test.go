package routers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/gin-gonic/gin"
)

// The configured HTTP ingress, not the underlying Gin matcher, is the
// application handler. Otherwise BeforeRouting is silently skipped.
func TestServerHandlerServesComposedIngress(t *testing.T) {
	ingressCalled := false
	server := &Server{
		config: setting.Config{AllowedOrigins: []string{"https://client.example"}},
		router: gin.New(),
		httpHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ingressCalled = true
			w.Header().Set("X-Ingress", "canonical")
			w.WriteHeader(http.StatusAccepted)
		}),
	}

	req := httptest.NewRequest(http.MethodGet, "/no-gin-route", nil)
	req.Header.Set("Origin", "https://client.example")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if !ingressCalled || rec.Code != http.StatusAccepted {
		t.Fatalf("HTTP ingress bypassed: called=%t code=%d", ingressCalled, rec.Code)
	}
	if got := rec.Header().Get("X-Ingress"); got != "canonical" {
		t.Fatalf("unexpected ingress marker: %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://client.example" {
		t.Fatalf("CORS policy no longer wraps ingress: %q", got)
	}
}

func TestServerHandlerRejectsMissingIngress(t *testing.T) {
	server := &Server{router: gin.New()}
	defer func() {
		if recover() == nil {
			t.Fatal("unwired Server.Handler silently served Gin and skipped BeforeRouting")
		}
	}()
	_ = server.Handler()
}

// This exercises the actual web.Router's HTTP ingress served by Server.Handler
// against the *same* Gin route matcher, including unmatched requests.
func TestServerHandlerBeforeRoutingCoversGin404405AndHEAD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	routes := web.NewRouter()
	engine := routes.GinEngine()
	engine.HandleMethodNotAllowed = true
	routes.BeforeRouting(func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Before-Matcher", "present")
			next.ServeHTTP(w, req)
		}
	})
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}
	routes.Get("/alive", handler)
	routes.Head("/alive", handler)
	// Compile ingress before serving, as the production NewServer does.
	server := &Server{
		config:      setting.Config{AllowedOrigins: []string{"https://client.example"}},
		router:      engine,
		httpHandler: routes.WrapIngress(),
	}
	for _, tc := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/missing", http.StatusNotFound},
		{http.MethodPost, "/alive", http.StatusMethodNotAllowed},
		{http.MethodHead, "/alive", http.StatusNoContent},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("Origin", "https://client.example")
			rec := httptest.NewRecorder()
			server.Handler().ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status=%d, want=%d", rec.Code, tc.status)
			}
			if rec.Header().Get("X-Before-Matcher") != "present" {
				t.Fatal("BeforeRouting did not execute before Gin matcher")
			}
			if rec.Header().Get("Access-Control-Allow-Origin") != "https://client.example" {
				t.Fatal("outer CORS policy missing")
			}
		})
	}
}
