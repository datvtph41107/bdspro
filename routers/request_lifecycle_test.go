package routers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
)

type lifecycleOutput struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	events chan map[string]any
}

func (o *lifecycleOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	var event map[string]any
	if err := json.Unmarshal(p, &event); err != nil {
		return 0, err
	}
	n, err := o.buf.Write(p)
	o.events <- event
	return n, err
}

func lifecycleLogger(t *testing.T) *lifecycleOutput {
	t.Helper()
	previous := zerologlog.Logger
	t.Cleanup(func() { zerologlog.Logger = previous })
	o := &lifecycleOutput{events: make(chan map[string]any, 16)}
	zerologlog.Logger = zerolog.New(o).Level(zerolog.DebugLevel)
	return o
}

func lifecycleEvent(t *testing.T, o *lifecycleOutput, message string) map[string]any {
	t.Helper()
	select {
	case e := <-o.events:
		if e["message"] != message {
			t.Fatalf("want %s, got %v", message, e)
		}
		return e
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", message)
		return nil
	}
}

func TestRequestLifecycleSlowBeforeCompletion(t *testing.T) {
	o := lifecycleLogger(t)
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	entered := make(chan struct{})
	router := web.NewRouter()
	router.Get("/slow", func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusNoContent)
	})
	h := requestLoggingWithThreshold(router.WrapIngress(), 250*time.Millisecond)
	done := make(chan struct{})
	resp := httptest.NewRecorder()
	go func() {
		defer close(done)
		h.ServeHTTP(resp, httptest.NewRequest("GET", "/slow?token=secret-query", nil))
	}()
	defer func() { unblock(); <-done }()
	start := lifecycleEvent(t, o, "http.started")
	select {
	case <-entered:
	case <-time.After(5*time.Second):
		t.Fatal("handler did not enter")
	}
	slow := lifecycleEvent(t, o, "http.slow")
	if slow["handler"] == nil || slow["handler_source"] == nil {
		t.Fatalf("missing handler metadata: %v", slow)
	}
	if slow["status"] != nil {
		t.Fatalf("slow event invented a response status: %v", slow)
	}
	unblock()
	<-done
	end := lifecycleEvent(t, o, "http.request")
	if end["status"] != float64(204) || resp.Code != 204 || end["handler"] != slow["handler"] {
		t.Fatalf("incorrect completion: %v", end)
	}
	for _, event := range []map[string]any{start, slow, end} {
		if event["request_id"] != resp.Header().Get("X-Request-ID") || event["path"] != "/slow" {
			t.Fatalf("incorrect correlation: %v", event)
		}
		t.Log(event)
	}
	if strings.Contains(o.buf.String(), "secret-query") || len(o.events) != 0 {
		t.Fatal("unexpected data or duplicate event")
	}
}

func TestRequestLifecycleGuardAndMissingRoute(t *testing.T) {
	o := lifecycleLogger(t)
	router := web.NewRouter()
	router.Get("/guard", lifecycleGuard, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("blocked endpoint ran")
		w.WriteHeader(http.StatusOK)
	})
	h := requestLoggingWithThreshold(router.WrapIngress(), time.Hour)
	for _, tc := range []struct {
		path string
		status int
	}{{"/guard", 403}, {"/missing", 404}} {
		resp := httptest.NewRecorder()
		h.ServeHTTP(resp, httptest.NewRequest("GET", tc.path, nil))
		lifecycleEvent(t, o, "http.started")
		e := lifecycleEvent(t, o, "http.request")
		if e["status"] != float64(tc.status) || resp.Code != tc.status {
			t.Fatalf("wrong response: %v", e)
		}
		if name, _ := e["handler"].(string); tc.path == "/guard" && !strings.HasSuffix(name, ".lifecycleGuard") {
			t.Fatalf("expected guard, got %v", e)
		}
		if tc.path == "/missing" && e["handler"] != nil {
			t.Fatalf("invented handler for unmatched route: %v", e)
		}
		if len(o.events) != 0 {
			t.Fatal("unexpected duplicate or slow event")
		}
	}
}

func lifecycleGuard(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusForbidden)
}

func TestRequestLifecyclePanicPropagates(t *testing.T) {
	o := lifecycleLogger(t)
	h := requestLoggingWithThreshold(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("secret-panic-value")
	}), time.Hour)
	func() {
		defer func() {
			if recover() != "secret-panic-value" {
				t.Error("panic was not propagated")
			}
		}()
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	}()
	lifecycleEvent(t, o, "http.started")
	e := lifecycleEvent(t, o, "http.failed")
	if e["reason"] != "panic" || e["status"] != nil || strings.Contains(o.buf.String(), "secret-panic-value") || len(o.events) != 0 {
		t.Fatalf("incorrect panic event: %v", e)
	}
}
