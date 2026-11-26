package organization

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeOrganizationName(t *testing.T) {
	if got, err := NormalizeOrganizationName("  Công ty BDSPro  "); err != nil || got != "Công ty BDSPro" {
		t.Fatal("name normalization failed")
	}
	for _, raw := range []string{" ", "invalid\nname", strings.Repeat("a", 101)} {
		if _, err := NormalizeOrganizationName(raw); !errors.Is(err, ErrInvalidOrganizationName) {
			t.Fatal("invalid organization name accepted")
		}
	}
}
