package setting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeEnvironmentKey(t *testing.T) {
	tests := []struct {
		name    string
		section string
		key     string
		file    bool
		valid   bool
	}{
		{
			name:    "BDS__database__DB_SOURCE",
			section: "database",
			key:     "DB_SOURCE",
			valid:   true,
		},
		{
			name:    "BDS____RUN_MODE",
			section: "",
			key:     "RUN_MODE",
			valid:   true,
		},
		{
			name:    "BDS__log_0X2E_file__FILE_NAME",
			section: "log.file",
			key:     "FILE_NAME",
			valid:   true,
		},
		{
			name:    "BDS__security__TOKEN_KEY__FILE",
			section: "security",
			key:     "TOKEN_KEY",
			file:    true,
			valid:   true,
		},
		{
			name:  "BDS__security",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section, key, file, valid :=
				decodeEnvironmentKey(tt.name)

			if section != tt.section ||
				key != tt.key ||
				file != tt.file ||
				valid != tt.valid {
				t.Fatalf(
					"decode = (%q,%q,%v,%v)",
					section, key, file, valid,
				)
			}
		})
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	secretFile := filepath.Join(
		t.TempDir(), "token",
	)

	if err := os.WriteFile(
		secretFile,
		[]byte("a-secret-value\n"),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	cfg, err := NewConfigProviderFromData(`
[cache]
ADAPTER = memory
[security]
TOKEN_KEY = old
`)
	if err != nil {
		t.Fatal(err)
	}

	err = applyEnvironmentOverrides(
		cfg,
		[]string{
			"BDS__cache__ADAPTER=redis",
			"BDS__security__TOKEN_KEY__FILE=" +
				secretFile,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if got := cfg.Section("cache").
		Key("ADAPTER").String(); got != "redis" {
		t.Fatalf("cache adapter = %q", got)
	}

	if got := cfg.Section("security").
		Key("TOKEN_KEY").String(); got != "a-secret-value" {
		t.Fatal("secret override failed")
	}
}

func TestInvalidSecretDoesNotApplyOverrides(
	t *testing.T,
) {
	cfg, err := NewConfigProviderFromData(
		"[cache]\nADAPTER = memory\n",
	)
	if err != nil {
		t.Fatal(err)
	}

	err = applyEnvironmentOverrides(
		cfg,
		[]string{
			"BDS__cache__ADAPTER=redis",
			"BDS__security__TOKEN_KEY__FILE=" +
				filepath.Join(
					t.TempDir(), "missing",
				),
		},
	)

	if err == nil {
		t.Fatal("missing secret was accepted")
	}

	// The earlier override must not have been
	// applied during the validation phase.
	if got := cfg.Section("cache").
		Key("ADAPTER").String(); got != "memory" {
		t.Fatalf(
			"partial override applied: %q",
			got,
		)
	}
}

func TestDuplicateEnvironmentOverride(
	t *testing.T,
) {
	cfg, err := NewConfigProviderFromData("")
	if err != nil {
		t.Fatal(err)
	}

	err = applyEnvironmentOverrides(
		cfg,
		[]string{
			"BDS__cache__ADAPTER=memory",
			"BDS__CACHE__adapter=redis",
		},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "duplicate") {
		t.Fatalf(
			"duplicate override not rejected: %v",
			err,
		)
	}
}
