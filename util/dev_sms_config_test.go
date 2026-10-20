package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// A plaintext OTP delivery transport must fail closed outside development.
func TestDevSMSOutboxForbiddenInProduction(t *testing.T) {
	dir := t.TempDir()
	config := "ENVIRONMENT=production\nDB_DRIVER=postgres\nDB_SOURCE=postgresql://localhost/bdspro\nSERVER_ADDRESS=0.0.0.0:8080\n" +
		"PHONE_OTP_KEY=fixture-key-with-at-least-thirty-two-bytes\nDEV_SMS_OUTBOX=true\n" +
		"ACCESS_TOKEN_KEY=production-fixture-secret-key-more-than-32-bytes\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.env"), []byte(config), 0600))
	_, err := LoadConfig(dir)
	require.ErrorContains(t, err, "DEV_SMS_OUTBOX is allowed only in development")
}

func TestDevSMSOutboxRequiresOTPKey(t *testing.T) {
	dir := t.TempDir()
	config := "ENVIRONMENT=development\nDB_DRIVER=postgres\nDB_SOURCE=postgresql://localhost/bdspro\n" +
		"SERVER_ADDRESS=127.0.0.1:8080\nDEV_SMS_OUTBOX=true\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.env"), []byte(config), 0600))
	_, err := LoadConfig(dir)
	require.ErrorContains(t, err, "DEV_SMS_OUTBOX requires PHONE_OTP_KEY")
}
