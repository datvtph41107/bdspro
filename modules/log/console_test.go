package log

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestCompactConsolePreservesOriginalEvent(t *testing.T) {
	const id = "8b6424f1-a182-41a0-b8e7-62bd054c979f"
	tests := []struct {
		name string
		raw  string
		want []string
		hide []string
	}{
		{
			name: "http",
			raw:  `{"level":"info","message":"http.request","component":"http","method":"POST","path":"/api/v1/auth/login","status":200,"duration_ms":8.732,"bytes":482}`,
			want: []string{"[I]", "router: completed POST /api/v1/auth/login 200 8.732ms"},
			hide: []string{"bytes=", "component=", "duration_ms="},
		},
		{
			name: "sql named",
			raw:  `{"level":"debug","message":"db.query","component":"sql","query":"CreateSession","duration_ms":1.382,"rows":1,"query_id":"83d7b1ca912b"}`,
			want: []string{"[D]", "sql: CreateSession 1.382ms", "rows=1"},
			hide: []string{"query_id=", "component="},
		},
		{
			name: "sql unnamed error",
			raw:  `{"level":"error","message":"db.query","component":"sql","query":"unnamed","duration_ms":2.4,"query_id":"83d7b1ca912b","sqlstate":"23505","reason":"postgres_error"}`,
			want: []string{"[E]", "sql: unnamed 2.4ms", "query_id=83d7b1ca912b", "sqlstate=23505", "reason=postgres_error"},
		},
		{
			name: "delivery",
			raw:  `{"level":"error","message":"phone.delivery","component":"delivery","channel":"sms","accepted":false,"duration_ms":5000,"reason":"timeout","provider_status":503}`,
			want: []string{"delivery: sms accepted=false 5000ms", "reason=timeout", "provider_status=503"},
			hide: []string{"component="},
		},
		{
			name: "admission",
			raw:  `{"level":"warn","message":"admission.denied","component":"security","action":"login","scope":"source","attempts":5,"limit":5,"retry_after":60}`,
			want: []string{"security: admission.denied", "action=login", "scope=source", "retry_after=60"},
		},
		{
			name: "authorization denied",
			raw:  `{"level":"warn","message":"auth.denied","component":"security","reason":"browser_origin","code":"forbidden"}`,
			want: []string{"security: auth.denied", "reason=browser_origin", "code=forbidden"},
		},
		{
			name: "unknown event",
			raw:  `{"level":"info","message":"custom.event","component":"custom","extra":"preserved"}`,
			want: []string{"custom.event", "component=custom", "extra=preserved"},
		},
		{
			name: "incomplete event",
			raw:  `{"level":"info","message":"http.request","method":"GET"}`,
			want: []string{"http.request", "method=GET"},
		},
		{
			name: "control characters in path",
			raw:  `{"level":"info","message":"http.request","method":"GET","path":"/a\nb","status":200,"duration_ms":1}`,
			want: []string{`router: completed GET "/a\nb" 200 1ms`},
			hide: []string{"/a\nb"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event map[string]any
			if err := json.Unmarshal([]byte(tt.raw), &event); err != nil {
				t.Fatal(err)
			}
			event["request_id"] = id
			event["time"] = "2026-11-27T10:30:05+07:00"
			raw, err := json.Marshal(event)
			if err != nil {
				t.Fatal(err)
			}
			raw = append(raw, '\n')
			before := bytes.Clone(raw)
			var console, file bytes.Buffer
			w := fanoutWriter{newConsoleWriter(&console, true, true), &file}
			if _, err := w.Write(raw); err != nil {
				t.Fatal(err)
			}
			line := console.String()
			for _, want := range append(tt.want, "rid=8b6424f1") {
				if !strings.Contains(line, want) {
					t.Errorf("missing %q in %q", want, line)
				}
			}
			for _, hidden := range append(tt.hide, id, "request_id=", "\x1b") {
				if strings.Contains(line, hidden) {
					t.Errorf("unexpected %q in %q", hidden, line)
				}
			}
			if !bytes.Equal(raw, before) || !bytes.Equal(file.Bytes(), before) {
				t.Fatal("console changed original JSON")
			}
			t.Log(strings.TrimSpace(line))
		})
	}
}

func TestFullTextFileKeepsMetadata(t *testing.T) {
	const raw = `{"level":"info","message":"http.request","request_id":"8b6424f1-a182-41a0-b8e7-62bd054c979f","bytes":482,"time":"2026-11-27T10:30:05+07:00"}` + "\n"
	var out bytes.Buffer
	if _, err := newConsoleWriter(&out, true, false).Write([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"request_id=8b6424f1-a182-41a0-b8e7-62bd054c979f", "bytes=482", "http.request", "2026-11-27"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in %q", want, out.String())
		}
	}
}
