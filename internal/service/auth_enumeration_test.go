package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/internal/config"
	db "github.com/datvtph41107/bdspro/internal/database"
	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/datvtph41107/bdspro/internal/util"
	"golang.org/x/crypto/bcrypt"
)

func TestUnknownPasswordUsesProductionBcryptCost(t *testing.T) {
	cost, err := bcrypt.Cost([]byte(dummyPasswordHash))
	if err != nil {
		t.Fatalf("dummy hash is not valid bcrypt: %v", err)
	}
	if cost != bcrypt.DefaultCost {
		t.Fatalf("dummy cost = %d, want production cost %d", cost, bcrypt.DefaultCost)
	}
	if err := util.CheckPassword("dummy-secret-string", dummyPasswordHash); err != nil {
		t.Fatalf("dummy bcrypt hash is not usable: %v", err)
	}
}

func TestLoginDoesNotDiscloseMissingEmailOrPasswordCredential(t *testing.T) {
	for _, tt := range []struct {
		name string
		email string
		passwordHash string
	}{
		{"unknown email", "other@example.test", ""},
		{"no password credential", "known@example.test", ""},
		{"wrong password", "known@example.test", dummyPasswordHash},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := &sessionMemoryStore{
				account: db.Account{ID: 12, PersonID: 33, Email: "known@example.test"},
				passwordHash: tt.passwordHash,
			}
			service := NewAuthService(store, nil, config.Config{})
			_, err := service.Login(context.Background(), strings.TrimSpace(tt.email), "not-the-password", "", "")
			if !errors.Is(err, apperrors.InvalidCredentials) {
				t.Fatalf("expected same InvalidCredentials error, got %v", err)
			}
		})
	}
}
