package setting

import (
	"strings"
	"testing"
)

func TestTokenSigningKeyPolicy(t *testing.T) {
	const example = "01234567890123456789012345678901"
	if MinTokenSymmetricKeyBytes != 32 {
		t.Fatal("existing key size contract changed")
	}
	for _, tc := range []struct {
		name string
		mode string
		key  string
		ok   bool
	}{
		{"empty", "development", "", false},
		{"short", "development", strings.Repeat("a", 31), false},
		{"minimum", "production", strings.Repeat("a", 32), true},
		{"long", "production", strings.Repeat("a", 64), true},
		{"byte boundary", "production", strings.Repeat("đ", 16), true},
		{"development example", "development", example, true},
		{"production example", "production", example, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateTokenSigningKey(tc.mode, tc.key); (err == nil) != tc.ok {
				t.Fatal("key policy mismatch")
			}
		})
	}
}
