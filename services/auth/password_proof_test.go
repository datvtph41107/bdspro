package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/datvtph41107/bdspro/models/db"
	password "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/jackc/pgx/v5"
)

type fakePasswordReader struct {
	email     string
	account   db.ReadAccountByEmailRow
	hash      string
	lookupErr error
	hashErr   error
}

func (f *fakePasswordReader) ReadAccountByEmail(_ context.Context, email string) (db.ReadAccountByEmailRow, error) {
	f.email = email
	return f.account, f.lookupErr
}
func (f *fakePasswordReader) GetPasswordHash(_ context.Context, accountID int64) (string, error) {
	if accountID != f.account.ID {
		panic("wrong account ID")
	}
	return f.hash, f.hashErr
}
func TestVerifyPasswordProof(t *testing.T) {
	hash, err := password.HashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	reader := &fakePasswordReader{account: db.ReadAccountByEmailRow{ID: 42}, hash: hash}
	id, err := verifyPasswordProof(context.Background(), reader, " USER@Example.COM ", "correct-password")
	if err != nil || id != 42 || reader.email != "user@example.com" {
		t.Fatalf("id=%d lookup=%q err=%v", id, reader.email, err)
	}
	_, err = verifyPasswordProof(context.Background(), reader, "USER@example.com", "incorrect-password")
	if !errors.Is(err, ErrInvalidPasswordProof) {
		t.Fatalf("wrong password err=%v", err)
	}
}
func TestVerifyPasswordProofUnknownAndPasswordless(t *testing.T) {
	for _, tc := range []struct {
		name               string
		lookupErr, hashErr error
	}{
		{"unknown account", pgx.ErrNoRows, nil},
		{"no password credential", nil, pgx.ErrNoRows},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &fakePasswordReader{account: db.ReadAccountByEmailRow{ID: 42}, lookupErr: tc.lookupErr, hashErr: tc.hashErr}
			_, err := verifyPasswordProof(context.Background(), reader, "user@example.com", "something")
			if !errors.Is(err, ErrInvalidPasswordProof) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
func TestVerifyPasswordProofDatabaseError(t *testing.T) {
	outage := errors.New("database unavailable")
	reader := &fakePasswordReader{lookupErr: outage}
	_, err := verifyPasswordProof(context.Background(), reader, "user@example.com", "anything")
	if !errors.Is(err, outage) {
		t.Fatalf("database outage incorrectly mapped: %v", err)
	}
}
