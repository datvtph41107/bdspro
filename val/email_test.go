package val

import (
	"errors"
	"testing"
)

func TestEmail(t *testing.T) {
	cases := []struct {
		input string
		want  string
		valid bool
	}{
		{"  USER@Example.COM ", "user@example.com", true},
		{"person@example.test", "person@example.test", true},
		{"Display <person@example.test>", "", false},
		{"without-at", "", false},
		{"a @example.test", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, err := Email(tc.input)
		if tc.valid {
			if err != nil || got != tc.want {
				t.Fatalf("Email(%q) = %q, %v; want %q", tc.input, got, err, tc.want)
			}
			continue
		}
		if !errors.Is(err, ErrInvalidEmail) {
			t.Fatalf("Email(%q) accepted: %q, %v", tc.input, got, err)
		}
	}
}
