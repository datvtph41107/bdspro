package setting

import (
	"path/filepath"
	"testing"
)

func TestLoadLogGlobalFrom(t *testing.T) {
	provider, err := NewConfigProviderFromData(`
[log]
MODE = file
FORMAT = json
LEVEL = warn
ROOT_PATH = data/log

[log.file]
FILE_NAME = application.log
`)
	if err != nil {
		t.Fatal(err)
	}

	workPath := t.TempDir()

	cfg, err := loadLogGlobalFrom(
		provider,
		workPath,
	)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Mode != "file" {
		t.Fatalf("mode = %q", cfg.Mode)
	}

	if cfg.Level != "warn" {
		t.Fatalf("level = %q", cfg.Level)
	}

	wantPath := filepath.Join(
		workPath,
		"data",
		"log",
	)

	if cfg.RootPath != wantPath {
		t.Fatalf(
			"root path = %q, want %q",
			cfg.RootPath, wantPath,
		)
	}

	if cfg.FileName != "application.log" {
		t.Fatalf(
			"filename = %q",
			cfg.FileName,
		)
	}
}

func TestLoadLogGlobalDefaults(t *testing.T) {
	provider, err := NewConfigProviderFromData("")
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := loadLogGlobalFrom(
		provider,
		t.TempDir(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Mode != "console" ||
		cfg.Format != "json" ||
		cfg.Level != "info" {
		t.Fatalf(
			"unexpected defaults: %+v",
			cfg,
		)
	}
}

func TestInvalidLogConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "unknown output",
			content: "[log]\nMODE = unsupported\n",
		},
		{
			name:    "invalid level",
			content: "[log]\nLEVEL = verbose\n",
		},
		{
			name:    "invalid filename",
			content: "[log.file]\nFILE_NAME = ../app.log\n",
		},
		{
			name:    "explicit empty mode",
			content: "[log]\nMODE =\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err :=
				NewConfigProviderFromData(tt.content)
			if err != nil {
				t.Fatal(err)
			}

			_, err = loadLogGlobalFrom(
				provider,
				t.TempDir(),
			)
			if err == nil {
				t.Fatal(
					"invalid logging configuration accepted",
				)
			}
		})
	}
}
