package setting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateKnownConfigKeys(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		wantErr bool
	}{
		{"known", "RUN_MODE = development\n[server]\nHTTP_SERVER_ADDRESS = :8080\n[log.file]\nFILE_NAME = test.log\n", false},
		{"ip-policy", "[ipactive]\nTEAM_CREATE_LIMIT = 2\n", false},
		{"unknown-key", "[server]\nHTTP_SERVER_ADRESS = :8080\n", true},
		{"unknown-section", "[servre]\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider, err := NewConfigProviderFromData(tc.content)
			if err != nil {
				t.Fatal(err)
			}
			err = validateKnownKeys(provider)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validation error = %v, want error %v", err, tc.wantErr)
			}
		})
	}
}

func TestCheckWebConfigDoesNotPublishGlobals(t *testing.T) {
	saveGlobals(t)
	AppWorkPath = "old-work"
	Server = ServerSettings{HTTPServerAddress: "old-server"}
	Common = CommonSettings{RunMode: "test"}
	CfgProvider = nil
	content := "RUN_MODE = development\n[server]\nHTTP_SERVER_ADDRESS = 127.0.0.1:8080\n" +
		"[security]\nTOKEN_SYMMETRIC_KEY = 01234567890123456789012345678901\n" +
		"ACCESS_TOKEN_DURATION = 15m\nREFRESH_TOKEN_DURATION = 24h\n" +
		"[database]\nDB_SOURCE = postgres://localhost/db\n[redis]\nADDR = 127.0.0.1:6379\n"
	file := filepath.Join(t.TempDir(), "app.ini")
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CheckWebConfig(PathOptions{Config: file}, nil); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	if AppWorkPath != "old-work" || Server.HTTPServerAddress != "old-server" ||
		Common.RunMode != "test" || CfgProvider != nil {
		t.Fatal("config check mutated process globals")
	}
	if err := CheckWebConfig(PathOptions{Config: file}, []string{"BDS__SERVRE__HTTP_SERVER_ADDRESS=:9090"}); err == nil {
		t.Fatal("unknown override accepted")
	}
	if err := CheckWebConfig(PathOptions{Config: file}, []string{"BDS__SERVER__HTTP_SERVER_ADDRESS=invalid"}); err == nil {
		t.Fatal("invalid listener accepted")
	}
}

func TestSecretFileBound(t *testing.T) {
	secret := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secret, []byte(strings.Repeat("x", int(maxSecretFileBytes)+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readEnvironmentFile(secret); err == nil || !strings.Contains(err.Error(), "size limit") {
		t.Fatalf("expected secret size error, got %v", err)
	}
}

func TestEnvironmentRejectsUnknownKeyWithoutMutation(t *testing.T) {
	provider, err := NewConfigProviderFromData("[server]\nHTTP_SERVER_ADDRESS = :8080\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := applyEnvironmentOverrides(provider, []string{"BDS__SERVRE__HTTP_SERVER_ADDRESS=:9090"}); err == nil {
		t.Fatal("unknown override accepted")
	}
	if got := provider.Section("server").Key("HTTP_SERVER_ADDRESS").String(); got != ":8080" {
		t.Fatalf("provider changed on failed override: %q", got)
	}
}
