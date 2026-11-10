package routers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/setting"
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
