package log

import (
	"bytes"
	"strings"
	"testing"
)

func TestRouterLifecycleConsole(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{`{"level":"debug","message":"http.started","component":"http","method":"GET","path":"/health"}`, "router: started GET /health"},
		{`{"level":"warn","message":"http.slow","component":"http","method":"GET","path":"/health","duration_ms":3001,"handler":"api.Health","handler_source":"routers/api/v1/health.go:12"}`, "router: slow GET /health 3001ms @ api.Health"},
		{`{"level":"info","message":"http.request","component":"http","method":"GET","path":"/health","status":200,"duration_ms":4000,"handler":"api.Health","handler_source":"routers/api/v1/health.go:12"}`, "router: completed GET /health 200 4000ms @ api.Health"},
		{`{"level":"error","message":"http.failed","component":"http","method":"GET","path":"/health","duration_ms":5,"reason":"panic"}`, "router: failed GET /health 5ms reason=panic"},
	} {
		var out, file bytes.Buffer
		raw := []byte(tc.raw + "\n")
		if _, err := (fanoutWriter{newConsoleWriter(&out, true, true), &file}).Write(raw); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), tc.want) || strings.Contains(out.String(), "handler_source=") || !bytes.Equal(raw, file.Bytes()) {
			t.Fatalf("unexpected output: %q file=%q", out.String(), file.String())
		}
		t.Log(strings.TrimSpace(out.String()))
	}
}
