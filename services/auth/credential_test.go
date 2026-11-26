package auth

import (
	"errors"
	"testing"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
)

func TestCredentialRemovalPolicy(t *testing.T) {
	for _, tc := range []struct {
		name string
		c    authmodel.Credentials
		kind string
		want error
	}{
		{"password only", authmodel.Credentials{EmailVerified: true, PasswordSet: true}, "password", ErrLastCredential},
		{"phone only", authmodel.Credentials{PhoneVerified: true}, "phone", ErrLastCredential},
		{"Google only", authmodel.Credentials{GoogleLinked: true}, "google", ErrLastCredential},
		{"email dependency", authmodel.Credentials{EmailVerified: true, PasswordSet: true, GoogleLinked: true}, "email", ErrEmailPasswordDependency},
		{"remove extra phone", authmodel.Credentials{EmailVerified: true, PasswordSet: true, PhoneVerified: true}, "phone", nil},
		{"remove extra password", authmodel.Credentials{EmailVerified: true, PasswordSet: true, GoogleLinked: true}, "password", nil},
		{"email alone is not login", authmodel.Credentials{EmailVerified: true, GoogleLinked: true}, "google", ErrLastCredential},
		{"absent phone", authmodel.Credentials{GoogleLinked: true}, "phone", ErrCredentialAbsent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := credentialRemovalAllowed(tc.c, tc.kind); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestPasswordReauthPurpose(t *testing.T) {
	if passwordReauthAction("password.set") || passwordReauthAction("anything") || !passwordReauthAction("sessions.revoke_all") {
		t.Fatal("incorrect password reauthentication scope")
	}
}
