package setting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOAuth2ConfigurationRejectsLegacyProviderKeys(t *testing.T) {
	file := filepath.Join(t.TempDir(), "app.ini")
	config := `RUN_MODE = development
[database]
DB_SOURCE = postgresql://root:secret@localhost:5432/bdspro?sslmode=disable
MIGRATION_URL = file://db/migration
[server]
HTTP_SERVER_ADDRESS = 127.0.0.1:8080
[security]
TOKEN_SYMMETRIC_KEY = 01234567890123456789012345678901
ACCESS_TOKEN_DURATION = 15m
REFRESH_TOKEN_DURATION = 24h
[oauth2]
ISSUER = https://issuer.example
AUTHORIZATION_URL = https://issuer.example/authorize
TOKEN_URL = https://issuer.example/token
JWKS_URL = https://issuer.example/keys
CLIENT_ID = oauth-client
CLIENT_SECRET = oauth-secret
REDIRECT_URI = http://localhost:8080/users/oauth2/callback
[oauth]
GOOGLE_CLIENT_ID = old-client
GOOGLE_CLIENT_SECRET = old-secret
GOOGLE_REDIRECT_URI = http://localhost:8080/users/google/callback
`
	if err := os.WriteFile(file, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(file)
	if err == nil || !strings.Contains(err.Error(), "legacy OAuth provider settings are unsupported") {
		t.Fatalf("legacy OAuth configuration accepted: %v", err)
	}
}
