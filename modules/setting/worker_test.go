package setting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func workerConfigFile(t *testing.T, raw string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "app.ini")
	if err := os.WriteFile(file, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

const minimumWorkerConfig = "[database]\nDB_SOURCE = postgres://user:secret@localhost:5432/bdspro\n" +
	"[redis]\nADDR = 127.0.0.1:6379\n" +
	"[security]\nTOKEN_SYMMETRIC_KEY = 01234567890123456789012345678901\n"

func TestWorkerSettingsDoNotRequireHTTPOrOAuth(t *testing.T) {
	file := workerConfigFile(t, minimumWorkerConfig)
	cfg, err := LoadWorkerSettings(file)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBSource == "" || cfg.RedisAddress == "" || cfg.AuditRetentionDays != 90 {
		t.Fatalf("unexpected worker settings: %+v", cfg)
	}
	// HTTP must remain fail-closed when its own settings are absent.
	if _, err := LoadConfig(file); err == nil {
		t.Fatal("HTTP unexpectedly started with incomplete settings")
	}
}

func TestWorkerCannotBypassSigningKeyValidation(t *testing.T) {
	prod := "RUN_MODE = production\n" + minimumWorkerConfig
	file := workerConfigFile(t, prod)
	if _, err := LoadWorkerSettings(file); err == nil || !strings.Contains(err.Error(), "production signing key") {
		t.Fatalf("production example key was accepted: %v", err)
	}
	file = workerConfigFile(t, strings.Replace(minimumWorkerConfig,
		"01234567890123456789012345678901", "short", 1))
	if _, err := LoadWorkerSettings(file); err == nil || !strings.Contains(err.Error(), "TOKEN_SYMMETRIC_KEY") {
		t.Fatalf("short worker key was accepted: %v", err)
	}
}

func TestWorkerValidatesOwnDependencies(t *testing.T) {
	for _, tc := range []struct{ name, raw, want string }{
		{"missing_database", strings.Replace(minimumWorkerConfig, "DB_SOURCE = postgres://user:secret@localhost:5432/bdspro", "", 1), "DB_SOURCE"},
		{"missing_redis", strings.Replace(minimumWorkerConfig, "ADDR = 127.0.0.1:6379", "", 1), "redis.ADDR"},
		{"invalid_audit", minimumWorkerConfig + "[audit]\nRETENTION_DAYS = 5000\n", "RETENTION_DAYS"},
		{"invalid_smtp", minimumWorkerConfig + "[mailer]\nSMTP_AUTH = x\n", "SMTP_AUTH"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := workerConfigFile(t, tc.raw)
			if _, err := LoadWorkerSettings(file); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("missing fail-fast %s: %v", tc.want, err)
			}
		})
	}
}

func TestWorkerAndWebUseIdenticalRetentionAndKeyRules(t *testing.T) {
	file := workerConfigFile(t, minimumWorkerConfig+"[audit]\nRETENTION_DAYS = 0\n")
	if _, err := LoadWorkerSettings(file); err == nil {
		t.Fatal("worker audit retention accepted invalid value")
	}
	if _, err := LoadCleanupSettings(file); err == nil {
		t.Fatal("cleanup audit retention accepted invalid value")
	}
}

func TestWorkerLoadsQueueSettingsFromAppINI(t *testing.T) {
	file := workerConfigFile(t, minimumWorkerConfig+
		"[queue]\nCONCURRENCY = 3\nCRITICAL_WEIGHT = 7\nDEFAULT_WEIGHT = 2\n")
	cfg, err := LoadWorkerSettings(file)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Queue != (QueueSettings{Concurrency: 3, CriticalWeight: 7, DefaultWeight: 2}) {
		t.Fatalf("unexpected configured worker queues: %+v", cfg.Queue)
	}
	t.Setenv("BDS__QUEUE__CONCURRENCY", "4")
	cfg, err = LoadWorkerSettings(file)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Queue.Concurrency != 4 {
		t.Fatalf("queue environment override ignored: %+v", cfg.Queue)
	}
}

func TestWorkerRejectsInvalidQueuePolicyBeforeOpeningDependencies(t *testing.T) {
	for _, tc := range []struct{ name, config, key string }{
		{"zero_concurrency", "CONCURRENCY = 0", "CONCURRENCY"},
		{"negative_weight", "CRITICAL_WEIGHT = -1", "CRITICAL_WEIGHT"},
		{"nonnumeric_weight", "DEFAULT_WEIGHT = many", "DEFAULT_WEIGHT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := workerConfigFile(t, minimumWorkerConfig+"[queue]\n"+tc.config+"\n")
			if _, err := LoadWorkerSettings(file); err == nil || !strings.Contains(err.Error(), tc.key) {
				t.Fatalf("invalid queue option %q not rejected: %v", tc.config, err)
			}
		})
	}
}
