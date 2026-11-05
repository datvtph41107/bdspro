package auth

import (
	"strings"
	"context"
	"errors"
	"testing"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/setting"
	"golang.org/x/crypto/bcrypt"
)

func TestUnknownPasswordUsesProductionBcryptCost(t *testing.T) {
	cost, err := bcrypt.Cost([]byte(dummyPasswordHash))
	if err != nil {
		t.Fatalf("dummy password hash must be valid bcrypt: %v", err)
	}
	if cost != bcrypt.DefaultCost {
		t.Fatalf("dummy bcrypt cost %d != production cost %d", cost, bcrypt.DefaultCost)
	}
	checkUnknownPassword("wrong-password")
}

func TestPasswordLoginMissingCredentialsUseCanonicalFailure(t *testing.T) {
	for _, tc := range []struct {
		name  string
		email string
		hash  string
	}{
		{name: "unknown email", email: "missing@example.test"},
		{name: "passwordless account", email: "account@example.test"},
		{name: "wrong password", email: "account@example.test", hash: dummyPasswordHash},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &sessionMemoryStore{
				account:      db.Account{ID: 11, Email: "account@example.test"},
				passwordHash: tc.hash,
			}
			svc := NewAuthService(store, nil, setting.Config{TokenSymmetricKey: strings.Repeat("k", 32)}, nil)
			_, err := svc.Login(context.Background(), tc.email, "wrong-password", "", "203.0.113.1")
			if !errors.Is(err, auth_model.ErrInvalidCredentials) {
				t.Fatalf("expected canonical public credential error, got %v", err)
			}
			if len(store.failures) != 1 || len(store.failures[0].CredentialMac) != 32 {
				t.Fatalf("one keyed rejection expected: %+v", store.failures)
			}
			if tc.name == "unknown email" && store.failures[0].AccountID.Valid {
				t.Fatal("unknown account audit must not invent identity")
			}
			if tc.name != "unknown email" && !store.failures[0].AccountID.Valid {
				t.Fatal("known account audit lost its account ownership")
			}
		})
	}
}
