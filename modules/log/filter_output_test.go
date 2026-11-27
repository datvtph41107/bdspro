package log

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
)

func TestDestinationFiltering(t *testing.T) {
	var terminal, file bytes.Buffer
	components, _ := parseComponents("http,default")
	w := fanoutWriter{
		filteredSink{out: newConsoleWriter(&terminal, true, true), level: zerolog.InfoLevel, components: components},
		filteredSink{out: &file, level: zerolog.DebugLevel},
	}
	logger := zerolog.New(zerolog.SyncWriter(w)).Level(zerolog.DebugLevel)
	logger.Debug().Str("component", "sql").Msg("sql-debug")
	logger.Info().Str("component", "sql").Msg("sql-info")
	logger.Info().Str("component", "http").Msg("http-info")
	logger.Info().Msg("startup")
	logger.Warn().Str("component", "security").Msg("security-warn")
	logger.Error().Str("component", "sql").Msg("sql-error")

	for _, message := range []string{"http-info", "startup", "security-warn", "sql-error"} {
		if !strings.Contains(terminal.String(), message) {
			t.Errorf("terminal lost %s: %s", message, terminal.String())
		}
	}
	for _, message := range []string{"sql-debug", "sql-info"} {
		if strings.Contains(terminal.String(), message) {
			t.Errorf("terminal showed filtered %s", message)
		}
	}
	for _, message := range []string{"sql-debug", "sql-info", "http-info", "startup", "security-warn", "sql-error"} {
		if !strings.Contains(file.String(), message) {
			t.Errorf("file lost %s", message)
		}
	}
	t.Log("terminal:\n" + terminal.String())
	t.Log("file:\n" + file.String())
}

func TestDestinationEmissionLevel(t *testing.T) {
	for _, tc := range []struct {
		mode, console, file string
		want                zerolog.Level
	}{
		{"console,file", "info", "debug", zerolog.DebugLevel},
		{"console", "warn", "debug", zerolog.WarnLevel},
		{"file", "debug", "error", zerolog.ErrorLevel},
		{"file,console", "error", "warn", zerolog.WarnLevel},
		{"console,file", "", "", zerolog.InfoLevel},
	} {
		o := OutputOptions{Mode: tc.mode, Level: "info", ConsoleLevel: tc.console, FileLevel: tc.file}
		if got := o.emissionLevel(); got != tc.want {
			t.Errorf("%+v: got %v want %v", tc, got, tc.want)
		}
	}
}

func TestFileLevelOverride(t *testing.T) {
	previous := zerologlog.Logger
	t.Cleanup(func() { zerologlog.Logger = previous })
	dir := t.TempDir()
	manager := NewManager()
	defer manager.Close()
	err := manager.ConfigureOutputs(OutputOptions{
		Mode: "file", ConsoleFormat: "console", FileFormat: "json", Level: "info",
		ConsoleLevel: "error", FileLevel: "debug", ConsoleComponents: "http",
		RootPath: dir, FileName: "events.log", MaxSizeMB: 1, MaxBackups: 2, MaxAgeDays: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	manager.GetLogger("sql").Debug("file-debug")
	raw, err := os.ReadFile(filepath.Join(dir, "events.log"))
	if err != nil || !bytes.Contains(raw, []byte("file-debug")) {
		t.Fatalf("file override ineffective: %s %v", raw, err)
	}
}

func TestLevelFanoutContinuesAfterFailure(t *testing.T) {
	var file bytes.Buffer
	w := fanoutWriter{
		filteredSink{out: brokenWriter{}, level: zerolog.InfoLevel},
		filteredSink{out: &file, level: zerolog.DebugLevel},
	}
	p := []byte("event")
	if _, err := w.WriteLevel(zerolog.ErrorLevel, p); err == nil || !bytes.Equal(p, file.Bytes()) {
		t.Fatal("level-aware fanout lost error or event")
	}
	file.Reset()
	if n, err := w.WriteLevel(zerolog.DebugLevel, p); err != nil || n != len(p) || !bytes.Equal(p, file.Bytes()) {
		t.Fatal("filtered write was not acknowledged")
	}
}
