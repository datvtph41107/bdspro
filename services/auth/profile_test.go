package auth

import (
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/optional"
)

func TestProfilePatchValidation(t *testing.T) {
	if field, err := profileField(optional.None[string](), true); err != nil || field.Valid {
		t.Fatal("omitted field must remain absent")
	}
	if field, err := profileField(optional.Some("  Nguyễn Văn A  "), true); err != nil || field.String != "Nguyễn Văn A" || !field.Valid {
		t.Fatal("valid display name rejected")
	}
	if field, err := profileField(optional.Some(""), false); err != nil || !field.Valid || field.String != "" {
		t.Fatal("empty optional name must remain explicit")
	}
	for _, value := range []string{"", "a\nb", strings.Repeat("đ", 101)} {
		if _, err := profileField(optional.Some(value), true); err == nil {
			t.Fatal("invalid display name accepted")
		}
	}
	if _, err := settingsField(optional.Some("EUR"), "VND", "USD"); err == nil {
		t.Fatal("unsupported currency accepted")
	}
}
