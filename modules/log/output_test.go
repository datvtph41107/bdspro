package log

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	zerologlog "github.com/rs/zerolog/log"
)

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("unavailable") }

func TestFanoutContinuesAfterFailure(t *testing.T) {
	var file bytes.Buffer
	if _, err := (fanoutWriter{brokenWriter{}, &file}).Write([]byte("event")); err == nil {
		t.Fatal("lost write error")
	}
	if file.String() != "event" {
		t.Fatal("first sink failure suppressed second sink")
	}
}

func TestFileRotationAndNamedLogger(t *testing.T) {
	previous := zerologlog.Logger
	t.Cleanup(func() { zerologlog.Logger = previous })
	dir := t.TempDir()
	manager := NewManager()
	defer manager.Close()
	err := manager.ConfigureOutputs(OutputOptions{Mode: "file", ConsoleFormat: "console", FileFormat: "json", Level: "debug", RootPath: dir, FileName: "events.log", MaxSizeMB: 1, MaxBackups: 2, MaxAgeDays: 1})
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		manager.GetLogger("sql").Info("%s", strings.Repeat("x", 400000))
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil || len(files) != 2 {
		t.Fatalf("rotation files=%v err=%v", files, err)
	}
	current, err := os.ReadFile(filepath.Join(dir, "events.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(current, []byte(`"component":"sql"`)) || bytes.Contains(current, []byte{0x1b}) {
		t.Fatal("missing component or ANSI in file")
	}
	if len(current) >= 1<<20 {
		t.Fatal("rotation did not bound active file")
	}
	info, err := os.Stat(filepath.Join(dir, "events.log"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatal("log file accessible to other users")
	}
}

func TestInvalidOutputDoesNotReplaceActiveLogger(t *testing.T) {
	manager := NewManager()
	defer manager.Close()
	before := manager.GetLogger(DEFAULT)
	err := manager.ConfigureOutputs(OutputOptions{Mode: "file"})
	if err == nil || manager.GetLogger(DEFAULT) != before {
		t.Fatal("invalid configuration changed active logger")
	}
	// Keep io.Writer conformance explicit for the failure-isolation boundary.
	var _ io.Writer = fanoutWriter{}
}
