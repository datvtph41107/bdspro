package architecture

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAuthFileOwnership prevents a return to one-file-per-authentication-action
// handlers or multiple independent phone/oidc authentication owners.
func TestAuthFileOwnership(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	removed := []string{
		"routers/api/v1/user/phone.go",
		"routers/api/v1/user/email.go",
		"routers/api/v1/user/password.go",
		"routers/api/v1/user/verify_email.go",
		"routers/api/v1/user/token.go",
		"services/auth/phone_otp.go",
		"services/auth/oauth2_auth.go",
		"models/account/verify_email.go",
		"models/account/verification_issue.go",
		"models/organization/team_list.go",
	}
	for _, name := range removed {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Errorf("fragmented authentication ownership returned: %s, stat error: %v", name, err)
		}
	}
	for _, required := range []string{
		"routers/api/v1/user/auth.go",
		"routers/api/v1/user/oauth2.go",
		"services/auth/auth.go",
		"services/auth/oauth2.go",
		"models/account/email.go",
		"models/auth/auth.go",
		"models/auth/session.go",
	} {
		if _, err := os.Stat(filepath.Join(root, required)); err != nil {
			t.Errorf("canonical owner missing: %s: %v", required, err)
		}
	}
}
