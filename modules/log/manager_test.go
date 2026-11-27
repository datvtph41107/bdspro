package log

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	zerologlog "github.com/rs/zerolog/log"
)

func TestFileOutputHasNoANSI(t *testing.T) {
	for _, format := range []string{"console", "json"} {
		for _, mode := range []string{"file", "console,file", "file,console"} {
			t.Run(format+"/"+mode, func(t *testing.T) {
				previous := zerologlog.Logger
				defer func() { zerologlog.Logger = previous }()
				manager := NewManager()
				defer manager.Close()
				dir := t.TempDir()
				if err := manager.ConfigureOutputs(OutputOptions{Mode: mode, ConsoleFormat: format, FileFormat: format, Level: "info", RootPath: dir, FileName: "test.log", MaxSizeMB: 1, MaxBackups: 2, MaxAgeDays: 1}); err != nil {
					t.Fatal(err)
				}
				manager.GetLogger(DEFAULT).Error("email delivery unavailable")
				contents, err := os.ReadFile(filepath.Join(dir, "test.log"))
				if err != nil {
					t.Fatal(err)
				}
				if bytes.Contains(contents, []byte{0x1b}) {
					t.Fatalf("ANSI escape in file: %q", contents)
				}
				if !strings.Contains(string(contents), "email delivery unavailable") {
					t.Fatalf("missing event: %q", contents)
				}
				if format == "json" && !json.Valid(bytes.TrimSpace(contents)) {
					t.Fatalf("invalid JSON: %q", contents)
				}
			})
		}
	}
}
