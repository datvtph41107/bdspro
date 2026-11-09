package account

import (
	"context"
	"errors"
	"strings"
	"testing"

	db "github.com/datvtph41107/bdspro/models/db"
)

type profileQueriesStub struct {
	profileWrites  int
	settingsWrites int
	profileArgs    db.UpdatePersonProfileParams
	settingsArgs   db.UpdateAccountSettingsParams
	affected       int64
}

func (q *profileQueriesStub) GetAccountProfile(context.Context, int64) (db.GetAccountProfileRow, error) {
	return db.GetAccountProfileRow{}, nil
}
func (q *profileQueriesStub) UpdatePersonProfile(_ context.Context, arg db.UpdatePersonProfileParams) (int64, error) {
	q.profileWrites++
	q.profileArgs = arg
	return q.affected, nil
}
func (q *profileQueriesStub) UpdateAccountSettings(_ context.Context, arg db.UpdateAccountSettingsParams) (int64, error) {
	q.settingsWrites++
	q.settingsArgs = arg
	return q.affected, nil
}

func TestProfileUpdateModelOwnsValidationAndSQLCMapping(t *testing.T) {
	ctx := context.Background()
	q := &profileQueriesStub{affected: 1}

	name := "  Alice  "
	if err := UpdateProfile(ctx, q, 7, ProfileUpdateOptions{DisplayName: &name}); err != nil {
		t.Fatal(err)
	}
	if q.profileWrites != 1 || q.profileArgs.AccountID != 7 ||
		!q.profileArgs.DisplayName.Valid || q.profileArgs.DisplayName.String != "Alice" ||
		q.profileArgs.FirstName.Valid || q.profileArgs.LastName.Valid {
		t.Fatalf("incorrect partial update mapping: %+v", q.profileArgs)
	}

	empty := ""
	if err := UpdateProfile(ctx, q, 7, ProfileUpdateOptions{FirstName: &empty}); err != nil {
		t.Fatal(err)
	}
	if !q.profileArgs.FirstName.Valid || q.profileArgs.FirstName.String != "" ||
		q.profileArgs.DisplayName.Valid {
		t.Fatalf("explicit empty must clear only one name: %+v", q.profileArgs)
	}

	invalid := "control\u0001"
	for _, opts := range []ProfileUpdateOptions{
		{},
		{FirstName: &invalid},
		{LastName: ptrString(strings.Repeat("x", 121))},
	} {
		if err := UpdateProfile(ctx, q, 7, opts); !errors.Is(err, ErrInvalidProfile) {
			t.Fatalf("invalid patch %+v: %v", opts, err)
		}
	}
	if q.profileWrites != 2 {
		t.Fatalf("invalid patches reached SQLC writer %d times", q.profileWrites)
	}
}

func TestSettingsUpdateModelOwnsAllowedValuesAndSQLCMapping(t *testing.T) {
	q := &profileQueriesStub{affected: 1}
	ctx := context.Background()
	locale := "en"
	if err := UpdateSettings(ctx, q, 9, SettingsUpdateOptions{Locale: &locale}); err != nil {
		t.Fatal(err)
	}
	if q.settingsWrites != 1 || q.settingsArgs.AccountID != 9 ||
		!q.settingsArgs.Locale.Valid || q.settingsArgs.Locale.String != "en" ||
		q.settingsArgs.Timezone.Valid || q.settingsArgs.Currency.Valid {
		t.Fatalf("incorrect partial settings mapping: %+v", q.settingsArgs)
	}
	bad := "invalid"
	for _, opts := range []SettingsUpdateOptions{{}, {Timezone: &bad}, {Currency: &bad}} {
		if err := UpdateSettings(ctx, q, 9, opts); !errors.Is(err, ErrInvalidSettings) {
			t.Fatalf("invalid settings %+v: %v", opts, err)
		}
	}
	if q.settingsWrites != 1 {
		t.Fatalf("invalid settings reached SQLC writer %d times", q.settingsWrites)
	}
}

func TestModelRejectsMissingAccountOrUncommittedWrite(t *testing.T) {
	q := &profileQueriesStub{affected: 0}
	display := "User"
	if err := UpdateProfile(context.Background(), q, 0, ProfileUpdateOptions{DisplayName: &display}); !errors.Is(err, ErrAccountUnavailable) {
		t.Fatalf("invalid account ID: %v", err)
	}
	if err := UpdateProfile(context.Background(), q, 7, ProfileUpdateOptions{DisplayName: &display}); !errors.Is(err, ErrAccountUnavailable) {
		t.Fatalf("missing updated account row: %v", err)
	}
}

func ptrString(value string) *string { return &value }
