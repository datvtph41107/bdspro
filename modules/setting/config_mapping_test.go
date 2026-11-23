package setting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMapSettingsStrictScalarValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		bad  bool
		want int
	}{
		{"missing", "[api]\n", false, 50},
		{"valid", "[api]\nMAX_RESPONSE_ITEMS = 120\n", false, 120},
		{"invalid", "[api]\nMAX_RESPONSE_ITEMS = abc\n", true, 0},
		{"empty", "[api]\nMAX_RESPONSE_ITEMS =\n", true, 0},
		{"overflow", "[api]\nMAX_RESPONSE_ITEMS = 999999999999999999999999999\n", true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := NewConfigProviderFromData(tc.raw)
			if err != nil {
				t.Fatal(err)
			}
			options := struct{ MaxResponseItems int }{MaxResponseItems: 50}
			err = mapSettings(cfg, "api", &options)
			if (err != nil) != tc.bad {
				t.Fatalf("mapping err=%v, want error=%t", err, tc.bad)
			}
			if !tc.bad && options.MaxResponseItems != tc.want {
				t.Fatalf("mapped %d, want %d", options.MaxResponseItems, tc.want)
			}
		})
	}
}

func TestMapSettingsIsolatesChildSection(t *testing.T) {
	cfg, err := NewConfigProviderFromData("[log]\nMODE = file\n[log.file]\nFILE_NAME = output.log\n")
	if err != nil {
		t.Fatal(err)
	}
	values := struct {
		Mode     string
		FileName string
	}{Mode: "default", FileName: "default.log"}
	if err := mapSettings(cfg, "log.file", &values); err != nil {
		t.Fatal(err)
	}
	if values.Mode != "default" || values.FileName != "output.log" {
		t.Fatalf("inherited a parent key or failed to map: %+v", values)
	}
}

func TestMapSettingsAcronymUsesExplicitKeyTag(t *testing.T) {
	cfg, err := NewConfigProviderFromData("[server]\nHTTP_SERVER_ADDRESS = :8080\n")
	if err != nil {
		t.Fatal(err)
	}
	var values struct {
		HTTPServerAddress string `ini:"HTTP_SERVER_ADDRESS"`
	}
	if err := mapSettings(cfg, "server", &values); err != nil {
		t.Fatal(err)
	}
	if values.HTTPServerAddress != ":8080" {
		t.Fatalf("acronym key mapped incorrectly: %+v", values)
	}
}

func TestMapSettingsExplicitEmptyStringOverridesDefault(t *testing.T) {
	cfg, err := NewConfigProviderFromData("[log]\nMODE =\n")
	if err != nil {
		t.Fatal(err)
	}
	values := struct{ Mode string }{Mode: "console"}
	if err := mapSettings(cfg, "log", &values); err != nil {
		t.Fatal(err)
	}
	if values.Mode != "" {
		t.Fatalf("explicit empty unexpectedly became a default: %q", values.Mode)
	}
}

func TestMapSettingsRejectsZeroDurationAfterParsing(t *testing.T) {
	cfg, err := NewConfigProviderFromData("[server]\nHTTP_SHUTDOWN_TIMEOUT = 0s\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loadGracefulSettingsFrom(cfg); err == nil {
		t.Fatal("explicit zero must not silently keep the default duration")
	}
	cfg, err = NewConfigProviderFromData("[server]\nHTTP_SHUTDOWN_TIMEOUT = 12s\nGRACEFUL_HAMMER_TIMEOUT = 20s\n")
	if err != nil {
		t.Fatal(err)
	}
	value, err := loadGracefulSettingsFrom(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if value.HTTPShutdownTimeout != 12*time.Second || value.HammerTimeout != 20*time.Second {
		t.Fatalf("wrong duration values: %+v", value)
	}
}

func TestMapSettingsDoesNotMutateConfigFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "app.ini")
	content := []byte("[api]\nMAX_RESPONSE_ITEMS = 85\n")
	if err := os.WriteFile(file, content, 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewConfigProviderFromFile(file)
	if err != nil {
		t.Fatal(err)
	}
	values := struct {
		MaxResponseItems int
		DefaultPagingNum int
	}{MaxResponseItems: 50, DefaultPagingNum: 20}
	if err := mapSettings(cfg, "api", &values); err != nil {
		t.Fatal(err)
	}
	if values.MaxResponseItems != 85 || values.DefaultPagingNum != 20 {
		t.Fatalf("defaults or mapped values incorrect: %+v", values)
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(content) {
		t.Fatal("reading/mapping unexpectedly rewrote the INI file")
	}
	if cfg.Section("api").Key("DEFAULT_PAGING_NUM").IsSet() {
		t.Fatal("mapping unexpectedly inserted a missing default key")
	}
}

func TestMapSettingsErrorsDoNotEchoSecretValues(t *testing.T) {
	const bad = "not-a-number-secret"
	cfg, err := NewConfigProviderFromData("[api]\nMAX_RESPONSE_ITEMS = " + bad + "\n")
	if err != nil {
		t.Fatal(err)
	}
	var target struct{ MaxResponseItems int }
	err = mapSettings(cfg, "api", &target)
	if err == nil {
		t.Fatal("expected parse failure")
	}
	// The wrapped strconv error currently contains the untrusted literal.
	// This regression ensures the adapter does not leak it into user errors.
	if strings.Contains(err.Error(), bad) {
		t.Fatalf("parse error leaks raw configuration value: %v", err)
	}
}
