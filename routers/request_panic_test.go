package routers

import (
	"bytes"
	"io"
	stdlog "log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Exercise the real server's recovery boundary, not only a recorder with recover.
// A panic after a partial response must still interrupt the stream.
func TestHTTPPanicPrivacy(t *testing.T) {
	for _, http2 := range []bool{false, true} {
		for _, partial := range []bool{false, true} {
			for _, intentional := range []bool{false, true} {
				name := "http1"
				if http2 {
					name = "http2"
				}
				if partial {
					name += "/partial"
				} else {
					name += "/before"
				}
				if intentional {
					name += "/sentinel"
				} else {
					name += "/raw"
				}
				t.Run(name, func(t *testing.T) {
					o := lifecycleLogger(t)
					var standard bytes.Buffer
					h := requestLoggingWithThreshold(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						if partial {
							w.Header().Set("Content-Length", "100")
							w.WriteHeader(http.StatusOK)
							_, _ = io.WriteString(w, "partial")
							w.(http.Flusher).Flush()
						}
						if intentional {
							panic(http.ErrAbortHandler)
						}
						// A non-comparable panic value must also be sanitized.
						panic(map[string]string{"secret": "synthetic-private-value"})
					}), time.Hour)
					server := httptest.NewUnstartedServer(h)
					server.Config.ErrorLog = stdlog.New(&standard, "", 0)
					server.EnableHTTP2 = http2
					if http2 {
						server.StartTLS()
					} else {
						server.Start()
					}
					defer server.Close()
					client := server.Client()
					client.Timeout = 5 * time.Second
					resp, err := client.Get(server.URL)
					if resp != nil {
						_, readErr := io.ReadAll(resp.Body)
						_ = resp.Body.Close()
						if err == nil {
							err = readErr
						}
						if partial && resp.Code != http.StatusOK {
							t.Errorf("partial response status changed: %d", resp.Code)
						}
					}
					server.Close() // synchronize with recovery and its ErrorLog writes
					if err == nil {
						t.Fatal("expected interrupted HTTP request")
					}
					start := lifecycleEvent(t, o, "http.started")
					end := lifecycleEvent(t, o, "http.failed")
					wantReason, wantLevel := "panic", "error"
					if intentional {
						wantReason, wantLevel = "handler_abort", "warn"
					}
					if end["reason"] != wantReason || end["level"] != wantLevel ||
						start["request_id"] != end["request_id"] {
						t.Fatalf("unexpected terminal event: %v", end)
					}
					if partial {
						if end["status"] != float64(200) {
							t.Fatalf("lost committed status: %v", end)
						}
					} else if end["status"] != nil {
						t.Fatalf("invented status: %v", end)
					}
					if standard.Len() != 0 || strings.Contains(o.buf.String(), "synthetic-private-value") {
						t.Fatal("panic leaked through HTTP ErrorLog or structured output")
					}
					if len(o.events) != 0 {
						t.Fatal("unexpected duplicate terminal event")
					}
				})
			}
		}
	}
}
