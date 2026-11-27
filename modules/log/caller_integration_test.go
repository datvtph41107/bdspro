package log_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/log"
	zerologlog "github.com/rs/zerolog/log"
)

// Verify line numbers on each public logging API, not just a nonempty caller.
func TestCallerLoggerPaths(t *testing.T) {
	previous := zerologlog.Logger
	m := log.GetManager()
	previousLevel := log.GetLevel()
	t.Cleanup(func() {
		if err := log.SetupStderrLogger(previousLevel); err != nil {
			t.Error(err)
		}
		zerologlog.Logger = previous
	})
	var out bytes.Buffer
	if err := m.Configure(&out, "json", log.INFO); err != nil {
		t.Fatal(err)
	}

	_, _, globalLine, _ := runtime.Caller(0)
	log.Info("global")
	_, _, namedLine, _ := runtime.Caller(0)
	m.GetLogger("test").Error("named")

	const id = "8b6424f1-a182-41a0-b8e7-62bd054c979f"
	request := log.With().Str("request_id", id).Logger()
	ctx := request.WithContext(context.Background())
	logger := log.For(ctx, "sql")
	_, _, contextLine, _ := runtime.Caller(0)
	logger.Error().Msg("context")
	_, _, eventLine, _ := runtime.Caller(0)
	log.Event(log.ERROR).Msg("event")
	log.Debug("disabled")

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 emitted events, got %q", out.String())
	}
	for i, line := range []int{globalLine, namedLine, contextLine, eventLine} {
		var event map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &event); err != nil {
			t.Fatal(err)
		}
		want := fmt.Sprintf("modules/log/caller_integration_test.go:%d", line+1)
		if event["caller"] != want || event["caller_func"] != "TestCallerLoggerPaths" {
			t.Errorf("event %d: want %q / TestCallerLoggerPaths, got %v / %v", i, want, event["caller"], event["caller_func"])
		}
		if i == 2 && (event["request_id"] != id || event["component"] != "sql") {
			t.Errorf("lost request correlation: %v", event)
		}
		t.Log(lines[i])
	}
}
