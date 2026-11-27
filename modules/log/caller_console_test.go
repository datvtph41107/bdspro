package log

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestCompactCallerVisibility(t *testing.T) {
	const source = "models/db/query_log.go:68"
	for _, tt := range []struct {
		level, message string
		visible        bool
	}{
		{"debug", "db.query", false},
		{"info", "http.request", false},
		{"warn", "db.query", true},
		{"error", "db.query", true},
		{"warn", "http.request", true},
		{"info", "Starting BDSPro", true},
		{"error", "custom.event", true},
	} {
		t.Run(tt.level+"/"+tt.message, func(t *testing.T) {
			event := map[string]any{
				"level": tt.level, "message": tt.message,
				"caller": source, "caller_func": "TraceQueryEnd",
				"time": "2026-11-27T10:30:05+07:00",
			}
			raw, err := json.Marshal(event)
			if err != nil {
				t.Fatal(err)
			}
			raw = append(raw, '\n')
			before := bytes.Clone(raw)
			var console, file bytes.Buffer
			if _, err := (fanoutWriter{newConsoleWriter(&console, true, true), &file}).Write(raw); err != nil {
				t.Fatal(err)
			}
			line := console.String()
			if strings.Contains(line, source+":TraceQueryEnd()") != tt.visible {
				t.Errorf("unexpected caller visibility: %q", line)
			}
			if strings.Contains(line, "caller_func=") || (!tt.visible && strings.Contains(line, source)) {
				t.Errorf("unexpected repeated source metadata: %q", line)
			}
			if !bytes.Equal(file.Bytes(), before) || !bytes.Equal(raw, before) {
				t.Fatal("console projection modified JSON")
			}
			t.Log(strings.TrimSpace(line))
		})
	}
}
