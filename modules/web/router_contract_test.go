package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

type contractTraceKey struct{}

type contractRecordingWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *contractRecordingWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *contractRecordingWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.bytes += n
	return n, err
}

func TestCanonicalMethodsRegisterAllMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()
	engine := router.GinEngine()
	router.Methods("GET, POST", "/multi", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, httptest.NewRequest(method, "/multi", nil))
		if response.Code != http.StatusNoContent {
			t.Fatalf("%s: status=%d, want 204", method, response.Code)
		}
	}
}

func TestCanonicalBeforeRoutingRewritesMethodPathAndContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()
	var wrapped *contractRecordingWriter
	router.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			wrapped = &contractRecordingWriter{ResponseWriter: w}
			req = req.WithContext(context.WithValue(req.Context(), contractTraceKey{}, "ingress"))
			req.Method = http.MethodPost
			req.URL.Path = "/target"
			next.ServeHTTP(wrapped, req)
		})
	})
	router.Post("/target", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || req.URL.Path != "/target" ||
			req.Context().Value(contractTraceKey{}) != "ingress" {
			t.Errorf("matcher lost rewritten method/path/context: %s %s", req.Method, req.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("accepted"))
	})
	response := httptest.NewRecorder()
	router.WrapIngress().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/source", nil))
	if response.Code != http.StatusAccepted || wrapped == nil ||
		wrapped.status != http.StatusAccepted || wrapped.bytes != len("accepted") {
		t.Fatalf("ingress wrapper bypass: status=%d wrapped=%+v", response.Code, wrapped)
	}
}

func TestCanonicalBeforeRoutingCanRejectBeforeMatchingAndHandleHEAD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()
	matched, seen := 0, 0
	router.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			seen++
			if req.URL.Path == "/blocked" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, req)
		})
	})
	router.Head("/head", func(w http.ResponseWriter, req *http.Request) {
		matched++
		w.WriteHeader(http.StatusNoContent)
	})
	h := router.WrapIngress()
	blocked := httptest.NewRecorder()
	h.ServeHTTP(blocked, httptest.NewRequest(http.MethodGet, "/blocked", nil))
	head := httptest.NewRecorder()
	h.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/head", nil))
	if blocked.Code != http.StatusForbidden || head.Code != http.StatusNoContent ||
		matched != 1 || seen != 2 {
		t.Fatalf("blocked=%d head=%d matched=%d seen=%d", blocked.Code, head.Code, matched, seen)
	}
}

func TestCanonicalBeforeRoutingRejectsInvalidAndLateRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, action := range []any{nil, (func(http.Handler) http.Handler)(nil), func(*gin.Context) {}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("invalid pre-matcher middleware accepted")
				}
			}()
			NewRouter().BeforeRouting(action)
		}()
	}
	router := NewRouter()
	router.Get("/route", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	func() {
		defer func() {
			if recover() == nil {
				t.Error("BeforeRouting registered after route matching configured")
			}
		}()
		router.BeforeRouting(func(next http.Handler) http.Handler { return next })
	}()
}

func TestCanonicalBeforeRoutingRejectsDuplicateContinuation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()
	router.BeforeRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req)
			next.ServeHTTP(w, req)
		})
	})
	reached := 0
	router.Get("/once", func(w http.ResponseWriter, req *http.Request) {
		reached++
		w.WriteHeader(http.StatusNoContent)
	})
	defer func() {
		if recover() == nil {
			t.Error("duplicate BeforeRouting continuation was accepted")
		}
		if reached != 1 {
			t.Errorf("endpoint ran %d times, want once", reached)
		}
	}()
	router.WrapIngress().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/once", nil))
}

func TestCanonicalBeforeRoutingCompilesProviderOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()
	compiled, entered, action := 0, 0, 0
	router.BeforeRouting(func(next http.Handler) http.Handler {
		compiled++
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			entered++
			next.ServeHTTP(w, req)
		})
	})
	router.Get("/repeated", func(w http.ResponseWriter, req *http.Request) {
		action++
		w.WriteHeader(http.StatusNoContent)
	})
	ingress := router.WrapIngress()
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		ingress.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/repeated", nil))
		if w.Code != http.StatusNoContent {
			t.Fatalf("request %d status=%d, want 204", i, w.Code)
		}
	}
	if compiled != 1 || entered != 3 || action != 3 {
		t.Fatalf("compiled=%d entered=%d action=%d, want 1/3/3", compiled, entered, action)
	}
}

func TestCanonicalAfterRoutingOrderAndContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()
	engine := router.GinEngine()
	var sequence []string
	router.AfterRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			sequence = append(sequence, "before")
			next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), contractTraceKey{}, "verified")))
			sequence = append(sequence, "after")
		})
	})
	router.Group("/teams", func() {
		router.Get("/:id", func(w http.ResponseWriter, req *http.Request) {
			sequence = append(sequence, "endpoint")
			if req.Context().Value(contractTraceKey{}) != "verified" {
				t.Error("HTTP request context was lost in AfterRouting")
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}, func(w http.ResponseWriter, req *http.Request) {
		sequence = append(sequence, "guard")
	})
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/teams/42", nil))
	want := []string{"before", "guard", "endpoint", "after"}
	if w.Code != http.StatusNoContent || !reflect.DeepEqual(sequence, want) {
		t.Fatalf("status=%d sequence=%v, want %v", w.Code, sequence, want)
	}
}

func TestCanonicalAfterRoutingResponseTermination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name   string
		action func(http.ResponseWriter, *http.Request)
		status int
		body   string
	}{
		{"explicit status", func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}, http.StatusForbidden, ""},
		{"implicit status", func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("done"))
		}, http.StatusOK, "done"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := NewRouter()
			engine := router.GinEngine()
			router.AfterRouting(tc.action)
			reached := false
			router.Get("/deny", func(w http.ResponseWriter, req *http.Request) {
				reached = true
			})
			resp := httptest.NewRecorder()
			engine.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/deny", nil))
			if reached || resp.Code != tc.status || resp.Body.String() != tc.body {
				t.Fatalf("reached=%v status=%d body=%q", reached, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestCanonicalAfterRoutingPreservesWrappedWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()
	engine := router.GinEngine()
	var observed *contractRecordingWriter
	router.AfterRouting(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			observed = &contractRecordingWriter{ResponseWriter: w}
			next.ServeHTTP(observed, req)
		})
	})
	router.Get("/wrapped", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	})
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wrapped", nil))
	if rec.Code != http.StatusCreated || observed == nil ||
		observed.status != http.StatusCreated || observed.bytes != 2 {
		t.Fatalf("HTTP response wrapper was bypassed: status=%d writer=%+v", rec.Code, observed)
	}
}

func TestCanonicalAfterRoutingRejectsInvalidProviderAtRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()
	var invalid func(http.Handler) http.Handler
	defer func() {
		if recover() == nil {
			t.Error("nil AfterRouting middleware was accepted")
		}
	}()
	router.AfterRouting(invalid)
}
