package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func writeAppConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.env"), []byte(content), 0600))
	return dir
}

func exampleAppConfig() string {
	return "DB_SOURCE=postgres://root:secret@localhost:5432/bdspro?sslmode=disable\n" +
		"HTTP_SERVER_ADDRESS=127.0.0.1:8080\n" +
		"MIGRATION_URL=file://db/migration\n" +
		"GRPC_SERVER_ADDRESS=127.0.0.1:9090\n" +
		"ALLOWED_ORIGINS=http://localhost:3000,https://example.com\n" +
		"TOKEN_SYMMETRIC_KEY=" + strings.Repeat("a", 32) + "\n" +
		"ACCESS_TOKEN_DURATION=15m\nREFRESH_TOKEN_DURATION=24h\n"
}

func TestLoadConfig(t *testing.T) {
	for _, key := range []string{"DB_SOURCE", "HTTP_SERVER_ADDRESS", "TOKEN_SYMMETRIC_KEY", "ACCESS_TOKEN_DURATION", "REFRESH_TOKEN_DURATION"} {
		t.Setenv(key, "")
	}
	config, err := LoadConfig(writeAppConfig(t, exampleAppConfig()))
	require.NoError(t, err)
	require.Contains(t, config.DBSource, "/bdspro?")
	require.Equal(t, "127.0.0.1:8080", config.HTTPServerAddress)
	require.Equal(t, 15*time.Minute, config.AccessTokenDuration)
	require.Equal(t, 24*time.Hour, config.RefreshTokenDuration)
	require.Equal(t, []string{"http://localhost:3000", "https://example.com"}, config.AllowedOrigins)
}

func TestLoadConfigProcessEnvOverridesFile(t *testing.T) {
	for _, key := range []string{"DB_SOURCE", "HTTP_SERVER_ADDRESS", "TOKEN_SYMMETRIC_KEY", "ACCESS_TOKEN_DURATION", "REFRESH_TOKEN_DURATION"} {
		t.Setenv(key, "")
	}
	t.Setenv("HTTP_SERVER_ADDRESS", "127.0.0.1:9090")
	config, err := LoadConfig(writeAppConfig(t, exampleAppConfig()))
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:9090", config.HTTPServerAddress)
}

func TestLoadConfigWithoutGRPCAddress(t *testing.T) {
	t.Setenv("GRPC_SERVER_ADDRESS", "")
	content := strings.Replace(exampleAppConfig(), "GRPC_SERVER_ADDRESS=127.0.0.1:9090\n", "", 1)
	config, err := LoadConfig(writeAppConfig(t, content))
	require.NoError(t, err)
	require.Empty(t, config.GRPCServerAddress)
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := LoadConfig(t.TempDir())
	require.Error(t, err)
}

func TestLoadConfigInvalidKey(t *testing.T) {
	for _, key := range []string{"DB_SOURCE", "HTTP_SERVER_ADDRESS", "TOKEN_SYMMETRIC_KEY", "ACCESS_TOKEN_DURATION", "REFRESH_TOKEN_DURATION"} {
		t.Setenv(key, "")
	}
	configFile := strings.Replace(exampleAppConfig(), strings.Repeat("a", 32), "short", 1)
	_, err := LoadConfig(writeAppConfig(t, configFile))
	require.Error(t, err)
}

func TestGoogleOAuthConfiguration(t *testing.T) {
 for _,tc:=range []struct{name,id,secret,redirect string;valid bool}{
  {name:"disabled",valid:true},
  {name:"valid localhost",id:"client",secret:"secret",redirect:"http://localhost:8080/users/google/callback",valid:true},
  {name:"partial",id:"client"},
  {name:"wrong path",id:"client",secret:"secret",redirect:"http://localhost:8080/other"},
  {name:"remote http",id:"client",secret:"secret",redirect:"http://example.com/users/google/callback"},
  {name:"https",id:"client",secret:"secret",redirect:"https://auth.example.com/users/google/callback",valid:true},
 } {
  t.Run(tc.name,func(t *testing.T) {
   t.Setenv("GOOGLE_CLIENT_ID",tc.id)
   t.Setenv("GOOGLE_CLIENT_SECRET",tc.secret)
   t.Setenv("GOOGLE_REDIRECT_URI",tc.redirect)
   config,err:=LoadConfig(writeAppConfig(t,exampleAppConfig()))
   if tc.valid {require.NoError(t,err);require.Equal(t,tc.id,config.GoogleClientID)} else {require.Error(t,err)}
  })
 }
}
