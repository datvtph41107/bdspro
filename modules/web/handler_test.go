package web

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"

	web_types "github.com/datvtph41107/bdspro/modules/web/types"
	"testing"
)

var errHijackProbe = errors.New("hijack attempted")

type hijackProbeWriter struct{ http.ResponseWriter }

func (w hijackProbeWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errHijackProbe
}

func TestResponseWriterPreservesHijacker(t *testing.T) {
	writer := &responseWriter{ResponseWriter: hijackProbeWriter{httptest.NewRecorder()}}
	_, _, err := writer.Hijack()
	if !errors.Is(err, errHijackProbe) {
		t.Fatalf("Hijack delegation error = %v, want sentinel", err)
	}
}

func TestResponseWriterHijackUnsupported(t *testing.T) {
	writer := &responseWriter{ResponseWriter: httptest.NewRecorder()}
	_, _, err := writer.Hijack()
	if !errors.Is(err, http.ErrNotSupported) {
		t.Fatalf("Hijack unsupported error = %v, want ErrNotSupported", err)
	}
}

func TestResponseWriterStatusFollowsHTTPCommit(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := &responseWriter{ResponseWriter: recorder}
	if writer.WrittenStatus() != 0 {
		t.Fatalf("new writer reported status = %d", writer.WrittenStatus())
	}
	if _, err := writer.Write([]byte("body")); err != nil {
		t.Fatal(err)
	}
	if writer.WrittenStatus() != http.StatusOK || recorder.Code != http.StatusOK {
		t.Fatalf("write status = %d, recorder = %d", writer.WrittenStatus(), recorder.Code)
	}
}

func TestWrapMiddlewareAndHandlerPrecedence(t *testing.T) {
	var steps []string
	middleware := func(name string) middlewareProvider {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				steps = append(steps, name+"-before")
				next.ServeHTTP(w, req)
				steps = append(steps, name+"-after")
			})
		}
	}
	pre := func(name string) web_types.PreMiddlewareProvider {
		return web_types.PreMiddlewareProvider(middleware(name))
	}
	providers, endpoint, hasPre := wrapMiddlewareAndHandler(
		[]any{middleware("after"), pre("pre-after")},
		[]any{middleware("group"), pre("pre-group")},
		[]any{middleware("route"), pre("pre-route"), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			steps = append(steps, "endpoint")
			w.WriteHeader(http.StatusAccepted)
		})},
	)
	if !hasPre {
		t.Fatal("pre middleware was not recognized")
	}
	w := httptest.NewRecorder()
	executeMiddlewaresHandler(w, httptest.NewRequest(http.MethodGet, "/", nil), providers, endpoint)
	want := []string{
		"pre-after-before", "pre-group-before", "pre-route-before",
		"after-before", "group-before", "route-before", "endpoint",
		"route-after", "group-after", "after-after",
		"pre-route-after", "pre-group-after", "pre-after-after",
	}
	if w.Code != http.StatusAccepted || !reflect.DeepEqual(steps, want) {
		t.Fatalf("status=%d steps=%v want=%v", w.Code, steps, want)
	}
}

func TestWrapMiddlewareAndHandlerStopsOnWrittenResponse(t *testing.T) {
	reached := false
	providers, endpoint, _ := wrapMiddlewareAndHandler(nil, nil, []any{
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}),
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }),
	})
	w := httptest.NewRecorder()
	executeMiddlewaresHandler(w, httptest.NewRequest(http.MethodGet, "/", nil), providers, endpoint)
	if w.Code != http.StatusUnauthorized || reached {
		t.Fatalf("status=%d endpoint called=%v", w.Code, reached)
	}
}

func TestWrapMiddlewareAndHandlerRejectsInvalidEndpoint(t *testing.T) {
	cases := []struct {
		name     string
		handlers []any
	}{
		{"missing", nil},
		{"nil endpoint", []any{http.HandlerFunc(nil)}},
		{"pre as endpoint", []any{web_types.PreMiddlewareProvider(func(next http.Handler) http.Handler { return next })}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid endpoint was accepted")
				}
			}()
			wrapMiddlewareAndHandler(nil, nil, tc.handlers)
		})
	}
}
