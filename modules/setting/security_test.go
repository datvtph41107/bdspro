package setting

import "testing"

func TestSigningKeyPolicy(t *testing.T) {
	example := "01234567890123456789012345678901"
	if err := validateTokenSigningKey("development", example); err != nil {
		t.Fatal(err)
	}
	if err := validateTokenSigningKey("production", example); err == nil {
		t.Fatal("development signing key accepted in production")
	}
	if err := validateTokenSigningKey("production", "short"); err == nil {
		t.Fatal("short signing key accepted")
	}
}
