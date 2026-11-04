package validation

import "testing"

func TestPasswordLength(t *testing.T) {
	for _, tt := range []struct {
		password string
		want bool
	}{
		{"1234567", false},
		{"12345678", true},
		{"123456789012345678901234567890123456789012345678901234567890123456789012", true},
		{"1234567890123456789012345678901234567890123456789012345678901234567890123", false},
	} {
		if got := PasswordLength(tt.password); got != tt.want {
			t.Errorf("PasswordLength(%q) = %t, want %t", tt.password, got, tt.want)
		}
	}
}

func TestEmailAddress(t *testing.T) {
	for _, tt := range []struct {
		input, canonical string
		valid bool
	}{
		{" A@Example.COM ", "a@example.com", true},
		{"not-an-email", "not-an-email", false},
		{"Jane <jane@example.com>", "jane <jane@example.com>", false},
	} {
		canonical, ok := EmailAddress(tt.input)
		if canonical != tt.canonical || ok != tt.valid {
			t.Errorf("EmailAddress(%q) = (%q,%t), want (%q,%t)", tt.input, canonical, ok, tt.canonical, tt.valid)
		}
	}
}
