package user

import (
	"context"
	"errors"
	"testing"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

func TestUserServiceRequiresVerifiedActorAndRequestStore(t *testing.T) {
	actions := []struct {
		name string
		run  func(context.Context) error
	}{
		{"read", func(ctx context.Context) error {
			_, err := GetAuthenticatedUser(ctx)
			return err
		}},
		{"update-profile", func(ctx context.Context) error {
			_, err := UpdateProfile(ctx, ProfilePatch{})
			return err
		}},
		{"update-settings", func(ctx context.Context) error {
			_, err := UpdateUserSettings(ctx, SettingsPatch{})
			return err
		}},
	}
	for _, action := range actions {
		t.Run(action.name, func(t *testing.T) {
			if err := action.run(context.Background()); !errors.Is(err, auth_model.ErrSessionUnavailable) {
				t.Fatalf("missing actor: got %v, want session unavailable", err)
			}
			verified := auth_model.WithActor(context.Background(), auth_model.Actor{
				AccountID: 1,
				SessionID: uuid.New(),
			})
			if err := action.run(verified); !errors.Is(err, db.ErrStoreMissing) {
				t.Fatalf("missing request store: got %v, want Store missing", err)
			}
		})
	}
}
