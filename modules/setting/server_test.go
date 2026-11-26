package setting

import "testing"

func TestValidateListenerAddress(t *testing.T) {
	for _, tc := range []struct {
		address string
		valid   bool
	}{
		{"127.0.0.1:8080", true},
		{":8080", true},
		{"[::1]:8080", true},
		{"127.0.0.1", false},
		{"127.0.0.1:0", false},
		{"127.0.0.1:65536", false},
		{"127.0.0.1:abc", false},
		{" :8080", false},
	} {
		t.Run(tc.address, func(t *testing.T) {
			err := validateListenerAddress(tc.address)
			if (err == nil) != tc.valid {
				t.Fatalf("validateListenerAddress(%q) = %v, valid=%v", tc.address, err, tc.valid)
			}
		})
	}
}
