package auth

import (
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
			svc := NewAuthService(store, nil, setting.Config{})
			_, err := svc.Login(context.Background(), tc.email, "wrong-password", "", "")
			if !errors.Is(err, auth_model.ErrInvalidCredentials) {
				t.Fatalf("expected canonical public credential error, got %v", err)
			}
		})
	}
}
