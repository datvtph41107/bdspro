package log

import (
	"io"
	"testing"
)

// Compare projections of the same JSON event, without disk, network, caller
// extraction or a debugger. This is not an HTTP throughput benchmark.
func BenchmarkDeveloperConsole(b *testing.B) {
	raw := []byte(`{"level":"info","time":"2026-11-27T10:30:05+07:00","message":"http.request","component":"http","method":"GET","path":"/api/v1/health","status":200,"duration_ms":3.2,"bytes":15,"handler":"v1.Health","handler_source":"routers/api/v1/api.go:112","request_id":"8b6424f1-a182-41a0-b8e7-62bd054c979f"}` + "\n")
	for _, compact := range []bool{false, true} {
		name := "full-text"
		if compact {
			name = "compact"
		}
		b.Run(name, func(b *testing.B) {
			w := newConsoleWriter(io.Discard, true, compact)
			b.ReportAllocs()
			b.SetBytes(int64(len(raw)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if n, err := w.Write(raw); err != nil || n != len(raw) {
					b.Fatalf("write: %d %v", n, err)
				}
			}
		})
	}
}
