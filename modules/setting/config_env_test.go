package setting

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvironmentOverrides(t *testing.T) {
	cfg, err := NewConfigProviderFromData("RUN_MODE = development\n[log.file]\nFILE_NAME = old.log\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := applyEnvironmentOverrides(cfg, []string{
		"BDS____RUN_MODE=production",
		"BDS__LOG_0X2E_FILE__FILE_NAME=new.log",
	}); err != nil {
		t.Fatal(err)
	}
	if got := cfg.Section("").Key("RUN_MODE").String(); got != "production" {
		t.Fatalf("RUN_MODE=%q", got)
	}
	if got := cfg.Section("log.file").Key("FILE_NAME").String(); got != "new.log" {
		t.Fatalf("FILE_NAME=%q", got)
	}
}

func TestEnvironmentSecretFile(t *testing.T) {
	secret := filepath.Join(t.TempDir(), "key.txt")
	if err := os.WriteFile(secret, []byte("supersecret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewConfigProviderFromData("")
	if err != nil {
		t.Fatal(err)
	}
	if err := applyEnvironmentOverrides(cfg, []string{"BDS__SECURITY__TOKEN_SYMMETRIC_KEY__FILE=" + secret}); err != nil {
		t.Fatal(err)
	}
	if got := cfg.Section("security").Key("TOKEN_SYMMETRIC_KEY").String(); got != "supersecret" {
		t.Fatalf("secret file value mismatch: len=%d", len(got))
	}
}

func TestEnvironmentRejectsInvalidAndDuplicates(t *testing.T) {
	cfg, err := NewConfigProviderFromData("RUN_MODE = development")
	if err != nil {
		t.Fatal(err)
	}
	if err := applyEnvironmentOverrides(cfg, []string{"BDS__INVALID=value"}); err == nil {
		t.Fatal("malformed env name accepted")
	}
	if err := applyEnvironmentOverrides(cfg, []string{"BDS____RUN_MODE=production", "BDS____run_mode=test"}); err == nil {
		t.Fatal("duplicate env accepted")
	}
	if got := cfg.Section("").Key("RUN_MODE").String(); got != "development" {
		t.Fatal("provider changed despite invalid env")
	}
}

func TestRetiredEnvironmentOverrideRejected(t *testing.T) {
	cfg, err := NewConfigProviderFromData("RUN_MODE = development\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := applyEnvironmentOverrides(cfg, []string{"BDS__OAUTH2__CLIENT_SECRET=ignored"}); err == nil {
		t.Fatal("removed section env override must be rejected")
	}
}
